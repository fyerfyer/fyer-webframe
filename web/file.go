package web

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var defaultMaxSize = int64(32 << 20)

type FileUploder struct {
	FileField    string
	DestPath     string
	maxSize      int64
	allowedTypes map[string]bool
}

func validatePath(path string) bool {
	// 检查路径是否包含 .. 或 . 或以 \ 开头
	if strings.Contains(path, "..") || strings.Contains(path, "/.") || strings.HasPrefix(path, "\\") {
		return false
	}
	return true
}

type FileOption func(*FileUploder)

func WithFileMaxSize(size int64) FileOption {
	return func(fu *FileUploder) {
		fu.maxSize = size
	}
}

func WithAllowedTypes(types []string) FileOption {
	return func(fu *FileUploder) {
		for _, typ := range types {
			fu.allowedTypes[typ] = true
		}
	}
}

func NewFileUploader(fileField string, destPath string, opts ...FileOption) *FileUploder {
	uploader := &FileUploder{
		FileField:    fileField,
		DestPath:     destPath,
		maxSize:      defaultMaxSize,
		allowedTypes: make(map[string]bool, 10),
	}

	for _, opt := range opts {
		opt(uploader)
	}

	return uploader
}

func (fu FileUploder) HandleUpload() HandlerFunc {
	return func(ctx *Context) {
		if !validatePath(fu.DestPath) {
			ctx.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid destination path",
			})
			return
		}

		// 确保目标目录存在
		if err := os.MkdirAll(fu.DestPath, 0766); err != nil {
			ctx.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to create destination directory",
			})
			return
		}

		src, header, err := ctx.Req.FormFile(fu.FileField)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
			return
		}
		defer src.Close()

		if header.Size > fu.maxSize {
			ctx.JSON(http.StatusBadRequest, map[string]string{
				"error": fmt.Sprintf("file size exceeds the limit of %d bytes", fu.maxSize),
			})
			return
		}

		// 检查文件名是否安全
		if !validatePath(header.Filename) {
			ctx.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid file name",
			})
			return
		}

		// 检查文件类型是否允许
		if len(fu.allowedTypes) > 0 {
			contentType := header.Header.Get("Content-Type")
			if contentType == "" {
				// Try to detect content type from file content
				buffer := make([]byte, 512)
				_, err = src.Read(buffer)
				if err != nil && err != io.EOF {
					ctx.JSON(http.StatusInternalServerError, map[string]string{
						"error": "failed to detect file type",
					})
					return
				}
				// Reset the file pointer
				if _, err = src.Seek(0, 0); err != nil {
					ctx.JSON(http.StatusInternalServerError, map[string]string{
						"error": "failed to process file",
					})
					return
				}
				contentType = http.DetectContentType(buffer)
			}

			if _, ok := fu.allowedTypes[contentType]; !ok {
				ctx.JSON(http.StatusBadRequest, map[string]string{
					"error": fmt.Sprintf("file type %s is not allowed", contentType),
				})
				return
			}
		}

		// 构建目标文件路径
		dstPath := filepath.Join(fu.DestPath, header.Filename)

		// 创建目标文件
		dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
			return
		}
		defer dst.Close()

		// 将文件内容拷贝到目标文件
		_, err = io.CopyBuffer(dst, src, nil)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
			return
		}

		ctx.JSON(http.StatusOK, map[string]string{
			"update_filename": header.Filename,
			"message":         "upload success",
		})
	}
}

type FileDownloader struct {
	DestPath string
}

func (f FileDownloader) HandleDownload() HandlerFunc {
	return func(ctx *Context) {
		if !validatePath(f.DestPath) {
			ctx.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid destination path",
			})
			return
		}

		req := ctx.PathParam("file").Value
		if req == "" {
			ctx.JSON(http.StatusBadRequest, map[string]string{
				"error": "missing file name",
			})
			return
		}

		// 检查请求的文件名是否安全
		if !validatePath(req) {
			ctx.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid file name",
			})
			return
		}

		path := filepath.Join(f.DestPath, filepath.Clean(req))
		f := filepath.Base(path)
		header := ctx.Resp.Header()
		header.Set("Content-Disposition", "attachment; filename="+f)
		header.Set("Content-Type", "application/octet-stream")
		header.Set("Content-Description", "File Transfer")
		header.Set("Content-Transfer-Encoding", "binary")
		header.Set("Expires", "0")
		header.Set("Cache-Control", "must-revalidate")
		header.Set("Pragma", "public")
		http.ServeFile(ctx.Resp, ctx.Req, path)
	}
}

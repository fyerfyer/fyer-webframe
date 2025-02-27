package web

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type FileUploder struct {
	FileField string
	DestPath  string
}

func validatePath(path string) bool {
	// 检查路径是否包含 .. 或 . 或以 \ 开头
	if strings.Contains(path, "..") || strings.Contains(path, "/.") || strings.HasPrefix(path, "\\") {
		return false
	}
	return true
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

		// 检查文件名是否安全
		if !validatePath(header.Filename) {
			ctx.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid file name",
			})
			return
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
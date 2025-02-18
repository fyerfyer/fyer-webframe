package resource

import (
	"github.com/fyerfyer/fyer-webframe/web"
	"github.com/patrickmn/go-cache"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type StaticResource struct {
	maxSize        int
	destPath       string
	pathPrefix     string
	extContentType map[string]string
	cache          *cache.Cache
}

type cacheItem struct {
	fileSize    int
	fileName    string
	contentType string
	data        []byte
}

var defaultPrefix = "default"

type StaticResourceOption func(*StaticResource)

func WithMaxSize(size int) StaticResourceOption {
	return func(sr *StaticResource) {
		sr.maxSize = size
	}
}

func WithPathPrefix(prefix string) StaticResourceOption {
	return func(sr *StaticResource) {
		sr.pathPrefix = prefix
	}
}

func WithCache(expiration, cleanupInterval time.Duration) StaticResourceOption {
	return func(sr *StaticResource) {
		sr.cache = cache.New(expiration, cleanupInterval)
	}
}

func WithExtContentTypes(types map[string]string) StaticResourceOption {
	return func(sr *StaticResource) {
		for k, v := range types {
			sr.extContentType[k] = v
		}
	}
}

func NewStaticResource(destPath string) *StaticResource {
	return &StaticResource{
		destPath:       destPath,
		pathPrefix:     defaultPrefix,
		extContentType: make(map[string]string),
		cache:          cache.New(time.Duration(10)*time.Minute, time.Duration(10)*time.Minute),
	}
}

func (sr *StaticResource) Handle() web.HandlerFunc {
	return func(ctx *web.Context) {
		// 获取请求路径
		req := ctx.PathParam("file").Value
		if req == "" {
			ctx.JSON(http.StatusBadRequest, map[string]string{
				"error": "missing file name",
			})
			return
		}

		// 从缓存中读取文件
		if item, ok := sr.readCache(req); ok {
			ctx.Resp.Header().Set("Content-Type", item.contentType)
			ctx.Resp.Write(item.data)
			return
		}

		path := filepath.Join(sr.destPath, filepath.Clean(req))
		f, err := os.Open(path)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
			return
		}
		defer f.Close()
		ext := filepath.Ext(path)
		t, ok := sr.extContentType[ext]
		if !ok {
			ctx.JSON(http.StatusBadRequest, map[string]string{
				"error": "unsupported file type",
			})
			return
		}

		data, err := io.ReadAll(f)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
			return
		}

		item := cacheItem{
			fileSize:    len(data),
			fileName:    filepath.Base(path),
			contentType: t,
			data:        data,
		}

		sr.writeCache(req, item)
		ctx.Resp.Header().Set("Content-Type", t)
	}
}

func (sr *StaticResource) readCache(req string) (cacheItem, bool) {
	item, ok := sr.cache.Get(req)
	if !ok {
		return cacheItem{}, false
	}
	return item.(cacheItem), true
}

func (sr *StaticResource) writeCache(req string, item cacheItem) {
	if item.fileSize <= sr.maxSize {
		sr.cache.Set(req, item, cache.DefaultExpiration)
	}
}

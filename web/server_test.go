package web

import (
	"context"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	s := NewHTTPServer()

	t.Run("server basic functionality", func(t *testing.T) {
		// 测试路由注册
		s.Get("/", func(ctx *Context) {
			ctx.Resp.Write([]byte("hello"))
		})

		// 测试 404
		s.Get("/user", func(ctx *Context) {
			ctx.Resp.Write([]byte("user"))
		})

		// 启动服务器
		go func() {
			err := s.Start(":8081")
			assert.NoError(t, err)
		}()

		time.Sleep(time.Second) // 等待服务器启动

		// 发送请求测试
		resp, err := http.Get("http://localhost:8081/")
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		resp, err = http.Get("http://localhost:8081/not-exist")
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		// 测试关闭
		err = s.Shutdown(context.Background())
		assert.NoError(t, err)
	})
}

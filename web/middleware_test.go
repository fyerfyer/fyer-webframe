package web

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddleware(t *testing.T) {
	s := NewHTTPServer()

	t.Run("middleware execution order", func(t *testing.T) {
		var order []string

		// 注册路由和中间件
		s.Get("/user/123", func(ctx *Context) {
			order = append(order, "handler")
		})

		s.Use("GET", "/user/*", func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				order = append(order, "wildcard before")
				next(ctx)
				order = append(order, "wildcard after")
			}
		})

		s.Use("GET", "/user/:id", func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				order = append(order, "param before")
				next(ctx)
				order = append(order, "param after")
			}
		})

		s.Use("GET", "/user/123", func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				order = append(order, "static before")
				next(ctx)
				order = append(order, "static after")
			}
		})

		// 发送请求
		req := httptest.NewRequest(http.MethodGet, "/user/123", nil)
		recorder := httptest.NewRecorder()

		s.ServeHTTP(recorder, req)

		// 验证执行顺序
		assert.Equal(t, []string{
			"static before",
			"param before",
			"wildcard before",
			"handler",
			"wildcard after",
			"param after",
			"static after",
		}, order)
	})

	t.Run("multiple middleware matching", func(t *testing.T) {
		s := NewHTTPServer()
		var calls []string

		// 注册两个路由
		s.Get("/user/a", func(ctx *Context) {
			calls = append(calls, "handler a")
		})
		s.Get("/user/b", func(ctx *Context) {
			calls = append(calls, "handler b")
		})

		// 注册一个通配符中间件
		s.Use("GET", "/user/*", func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				calls = append(calls, "middleware")
				next(ctx)
			}
		})

		// 测试两个路由
		req1 := httptest.NewRequest(http.MethodGet, "/user/a", nil)
		s.ServeHTTP(httptest.NewRecorder(), req1)

		req2 := httptest.NewRequest(http.MethodGet, "/user/b", nil)
		s.ServeHTTP(httptest.NewRecorder(), req2)

		// 验证中间件对两个路由都生效
		assert.Equal(t, []string{
			"middleware",
			"handler a",
			"middleware",
			"handler b",
		}, calls)
	})

	t.Run("middleware error handling", func(t *testing.T) {
		s := NewHTTPServer()

		// 测试注册到不存在的路由
		assert.Panics(t, func() {
			s.Use("GET", "/not-exist", func(next HandlerFunc) HandlerFunc {
				return next
			})
		})

		// 测试注册到不存在的HTTP方法
		assert.Panics(t, func() {
			s.Use("INVALID", "/", func(next HandlerFunc) HandlerFunc {
				return next
			})
		})
	})

	t.Run("regex route middleware", func(t *testing.T) {
		s := NewHTTPServer()
		var order []string

		// 注册带正则表达式的路由
		s.Get("/user/:id([0-9]+)", func(ctx *Context) {
			order = append(order, "handler")
		})

		// 注册各种类型的中间件
		s.Use("GET", "/user/*", func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				order = append(order, "wildcard before")
				next(ctx)
				order = append(order, "wildcard after")
			}
		})

		s.Use("GET", "/user/:id([0-9]+)", func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				order = append(order, "regex before")
				next(ctx)
				order = append(order, "regex after")
			}
		})

		s.Use("GET", "/user/:id", func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				order = append(order, "param before")
				next(ctx)
				order = append(order, "param after")
			}
		})

		s.Use("GET", "/user/123", func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				order = append(order, "static before")
				next(ctx)
				order = append(order, "static after")
			}
		})

		// 发送请求
		req := httptest.NewRequest(http.MethodGet, "/user/123", nil)
		recorder := httptest.NewRecorder()
		s.ServeHTTP(recorder, req)

		// 验证执行顺序：静态 -> 正则 -> 参数 -> 通配符
		assert.Equal(t, []string{
			"static before",
			"regex before",
			"param before",
			"wildcard before",
			"handler",
			"wildcard after",
			"param after",
			"regex after",
			"static after",
		}, order)
	})

	t.Run("regex route not match", func(t *testing.T) {
		s := NewHTTPServer()
		var order []string

		// 注册带正则表达式的路由
		s.Get("/user/:id([0-9]+)", func(ctx *Context) {
			order = append(order, "handler")
		})

		s.Use("GET", "/user/:id([0-9]+)", func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				order = append(order, "regex middleware")
				next(ctx)
			}
		})

		// 发送不匹配正则的请求
		req := httptest.NewRequest(http.MethodGet, "/user/abc", nil)
		recorder := httptest.NewRecorder()
		s.ServeHTTP(recorder, req)

		// 验证结果：应该返回 404，且中间件不会执行
		assert.Equal(t, http.StatusNotFound, recorder.Code)
		assert.Empty(t, order)
	})
}

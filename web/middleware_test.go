package web

import (
		"encoding/json"
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

		s.Use("GET", "/user/123", func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				order = append(order, "static before")
				next(ctx)
				order = append(order, "static after")
			}
		})

		s.Use("GET", "/user/:id", func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				order = append(order, "param before")
				next(ctx)
				order = append(order, "param after")
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

	t.Run("regex route middleware", func(t *testing.T) {
		s := NewHTTPServer()
		var order []string

		// 注册带正则表达式的路由
		s.Get("/user/:id([0-9]+)", func(ctx *Context) {
			order = append(order, "handler")
		})

		s.Use("GET", "/user/123", func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				order = append(order, "static before")
				next(ctx)
				order = append(order, "static after")
			}
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

func TestComplexMiddlewareOrdering(t *testing.T) {
	s := NewHTTPServer()
	var order []string

	s.Get("/api/users/:id/profile", func(ctx *Context) {
		order = append(order, "handler")
		ctx.String(http.StatusOK, "OK")
	})

	s.Use("GET", "/*", func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			order = append(order, "global-before")
			next(ctx)
			order = append(order, "global-after")
		}
	})

	s.Use("GET", "/api/*", func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			order = append(order, "api-before")
			next(ctx)
			order = append(order, "api-after")
		}
	})

	s.Use("GET", "/api/users/*", func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			order = append(order, "users-before")
			next(ctx)
			order = append(order, "users-after")
		}
	})

	// 这个不应该被匹配
	s.Use("GET", "/api/users/:id", func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			order = append(order, "user-id-before")
			next(ctx)
			order = append(order, "user-id-after")
		}
	})

	s.Use("GET", "/api/users/:id/profile", func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			order = append(order, "profile-before")
			next(ctx)
			order = append(order, "profile-after")
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users/123/profile", nil)
	recorder := httptest.NewRecorder()
	s.ServeHTTP(recorder, req)

	expectedOrder := []string{
		"global-before",
		"profile-before",
		"users-before",
		"api-before",
		"handler",
		"api-after",
		"users-after",
		"profile-after",
		"global-after",
	}
	assert.Equal(t, expectedOrder, order)
}

func TestMiddlewareShortCircuit(t *testing.T) {
	s := NewHTTPServer()
	var order []string

	// 注册路由处理函数
	s.Get("/test/path", func(ctx *Context) {
		order = append(order, "handler")
	})

	// 注册一个会短路的中间件
	s.Use("GET", "/test/*", func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			order = append(order, "middleware1 before")
			ctx.Abort() // 短路后续中间件
			ctx.JSON(http.StatusUnauthorized, map[string]string{
				"error": "unauthorized",
			})
			order = append(order, "middleware1 after")
		}
	})

	// 注册一个不应该被执行的中间件
	s.Use("GET", "/test/*", func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			order = append(order, "middleware2 before")
			next(ctx)
			order = append(order, "middleware2 after")
		}
	})

	// 发送请求
	req := httptest.NewRequest(http.MethodGet, "/test/path", nil)
	recorder := httptest.NewRecorder()
	s.ServeHTTP(recorder, req)

	// 验证执行顺序和结果
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Equal(t, []string{
		"middleware1 before",
		"middleware1 after",
	}, order, "Second middleware and handler should not be executed")

	// 验证响应内容
	var resp map[string]string
	err := json.NewDecoder(recorder.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "unauthorized", resp["error"])
}

func TestMiddlewareNext(t *testing.T) {
	s := NewHTTPServer()
	var order []string

	// 注册路由处理函数
	s.Get("/test/path", func(ctx *Context) {
		order = append(order, "handler")
	})

	// 注册一个使用 Next 的中间件
	s.Use("GET", "/test/*", func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			order = append(order, "middleware1 before")
			ctx.Next(next)
			order = append(order, "middleware1 after")
		}
	})

	// 发送请求
	req := httptest.NewRequest(http.MethodGet, "/test/path", nil)
	recorder := httptest.NewRecorder()
	s.ServeHTTP(recorder, req)

	// 验证执行顺序
	assert.Equal(t, []string{
		"middleware1 before",
		"handler",
		"middleware1 after",
	}, order)
}
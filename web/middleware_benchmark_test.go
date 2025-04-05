package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// BenchmarkNoMiddleware 测试没有中间件的基准性能
func BenchmarkNoMiddleware(b *testing.B) {
	s := NewHTTPServer()
	s.Get("/", func(ctx *Context) {
		ctx.String(http.StatusOK, "Hello World")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
	}
}

// BenchmarkSingleMiddleware 测试单个中间件
func BenchmarkSingleMiddleware(b *testing.B) {
	s := NewHTTPServer()

	s.Use("GET", "/*", func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			next(ctx)
		}
	})

	s.Get("/", func(ctx *Context) {
		ctx.String(http.StatusOK, "Hello World")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
	}
}

// BenchmarkMultipleMiddleware 测试多个中间件
func BenchmarkMultipleMiddleware(b *testing.B) {
	s := NewHTTPServer()

	for i := 0; i < 5; i++ {
		s.Use("GET", "/*", func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				next(ctx)
			}
		})
	}

	s.Get("/", func(ctx *Context) {
		ctx.String(http.StatusOK, "Hello World")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
	}
}

// BenchmarkShortCircuit 测试中间件短路
func BenchmarkShortCircuit(b *testing.B) {
	s := NewHTTPServer()

	s.Use("GET", "/*", func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			ctx.String(http.StatusOK, "Short Circuit")
			// 不调用next，造成短路
		}
	})

	s.Get("/", func(ctx *Context) {
		ctx.String(http.StatusOK, "Hello World")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
	}
}

// BenchmarkStaticPathMiddleware 测试静态路径中间件
func BenchmarkStaticPathMiddleware(b *testing.B) {
	s := NewHTTPServer()

	s.Use("GET", "/users", func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			next(ctx)
		}
	})

	s.Get("/users", func(ctx *Context) {
		ctx.String(http.StatusOK, "Users")
	})

	req := httptest.NewRequest(http.MethodGet, "/users", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
	}
}

// BenchmarkParamPathMiddleware 测试参数路径中间件
func BenchmarkParamPathMiddleware(b *testing.B) {
	s := NewHTTPServer()

	s.Use("GET", "/users/:id", func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			next(ctx)
		}
	})

	s.Get("/users/:id", func(ctx *Context) {
		ctx.String(http.StatusOK, "User: "+ctx.PathParam("id").Value)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
	}
}

// BenchmarkRegexPathMiddleware 测试正则路径中间件
func BenchmarkRegexPathMiddleware(b *testing.B) {
	s := NewHTTPServer()

	s.Use("GET", "/users/([0-9]+)", func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			next(ctx)
		}
	})

	s.Get("/users/([0-9]+)", func(ctx *Context) {
		ctx.String(http.StatusOK, "User ID")
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
	}
}

// BenchmarkWildcardPathMiddleware 测试通配符路径中间件
func BenchmarkWildcardPathMiddleware(b *testing.B) {
	s := NewHTTPServer()

	s.Use("GET", "/users/*", func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			next(ctx)
		}
	})

	s.Get("/users/123", func(ctx *Context) {
		ctx.String(http.StatusOK, "User Path")
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
	}
}

// BenchmarkComplexMiddlewareStack 测试复杂中间件栈
func BenchmarkComplexMiddlewareStack(b *testing.B) {
	s := NewHTTPServer()

	// 全局中间件
	s.Use("GET", "/*", func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			next(ctx)
		}
	})

	// API路径中间件
	s.Use("GET", "/api/*", func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			next(ctx)
		}
	})

	// 用户路径中间件
	s.Use("GET", "/api/users/*", func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			next(ctx)
		}
	})

	// 特定用户路径中间件
	s.Use("GET", "/api/users/:id", func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			next(ctx)
		}
	})

	s.Get("/api/users/123", func(ctx *Context) {
		ctx.String(http.StatusOK, "User Details")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users/123", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
	}
}

// BenchmarkMiddlewareManager 测试中间件管理器
func BenchmarkMiddlewareManager(b *testing.B) {
	s := NewHTTPServer()

	// 全局中间件
	s.Middleware().Global().Add(
		func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				next(ctx)
			}
		},
	)

	// 路径特定中间件
	s.Middleware().For("GET", "/api/users/*").Add(
		func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				next(ctx)
			}
		},
	)

	// 条件中间件
	s.Middleware().When(func(c *Context) bool {
		return true
	}).Add(
		func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				next(ctx)
			}
		},
	)

	s.Get("/api/users/123", func(ctx *Context) {
		ctx.String(http.StatusOK, "User Details")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users/123", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
	}
}

// BenchmarkAbortMiddleware 测试中断中间件
func BenchmarkAbortMiddleware(b *testing.B) {
	s := NewHTTPServer()

	s.Use("GET", "/*", func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			ctx.Abort()
			ctx.String(http.StatusServiceUnavailable, "Service Unavailable")
		}
	})

	s.Get("/", func(ctx *Context) {
		ctx.String(http.StatusOK, "Hello World")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
	}
}

// BenchmarkNestedGroups 测试嵌套路由组和中间件
func BenchmarkNestedGroups(b *testing.B) {
	s := NewHTTPServer()

	// 创建API组
	api := s.Group("/api")

	// 添加API组级中间件
	api.Use(func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			next(ctx)
		}
	})

	// 创建用户子组
	users := api.Group("/users")

	// 添加用户组级中间件
	users.Use(func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			next(ctx)
		}
	})

	// 添加用户路由
	users.Get("/:id", func(ctx *Context) {
		ctx.String(http.StatusOK, "User: "+ctx.PathParam("id").Value)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users/123", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, req)
	}
}
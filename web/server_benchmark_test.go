package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkStaticRoutes(b *testing.B) {
	s := NewHTTPServer()
	s.Get("/", func(ctx *Context) {
		ctx.String(http.StatusOK, "hello")
	})
	s.Get("/users", func(ctx *Context) {
		ctx.String(http.StatusOK, "users")
	})
	s.Get("/users/list", func(ctx *Context) {
		ctx.String(http.StatusOK, "users list")
	})
	s.Get("/products", func(ctx *Context) {
		ctx.String(http.StatusOK, "products")
	})

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.ServeHTTP(w, req)
		w.Body.Reset()
	}
}

func BenchmarkParamRoutes(b *testing.B) {
	s := NewHTTPServer()
	s.Get("/users/:id", func(ctx *Context) {
		ctx.String(http.StatusOK, "user id: %s", ctx.PathParam("id").Value)
	})
	s.Get("/posts/:postid", func(ctx *Context) {
		ctx.String(http.StatusOK, "post id: %s", ctx.PathParam("postid").Value)
	})
	s.Get("/users/:id/posts/:postid", func(ctx *Context) {
		ctx.String(http.StatusOK, "user %s, post %s",
			ctx.PathParam("id").Value, ctx.PathParam("postid").Value)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.ServeHTTP(w, req)
		w.Body.Reset()
	}
}

func BenchmarkWildcardRoutes(b *testing.B) {
	s := NewHTTPServer()
	s.Get("/static/*", func(ctx *Context) {
		ctx.String(http.StatusOK, "static file")
	})
	s.Get("/users/*", func(ctx *Context) {
		ctx.String(http.StatusOK, "user wildcard")
	})

	req := httptest.NewRequest(http.MethodGet, "/static/css/main.css", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.ServeHTTP(w, req)
		w.Body.Reset()
	}
}

func BenchmarkRegexRoutes(b *testing.B) {
	s := NewHTTPServer()
	s.Get("/users/([0-9]+)", func(ctx *Context) {
		ctx.String(http.StatusOK, "user regex")
	})
	s.Get("/posts/([0-9]+)", func(ctx *Context) {
		ctx.String(http.StatusOK, "post regex")
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.ServeHTTP(w, req)
		w.Body.Reset()
	}
}

func BenchmarkRouteGroups(b *testing.B) {
	s := NewHTTPServer()

	api := s.Group("/api")

	// Create users group
	users := api.Group("/users")
	users.Get("", func(ctx *Context) {
		ctx.String(http.StatusOK, "users list")
	})

	users.Get("/:id", func(ctx *Context) {
		ctx.String(http.StatusOK, "user: %s", ctx.PathParam("id").Value)
	})

	// Create posts group
	posts := api.Group("/posts")
	posts.Get("", func(ctx *Context) {
		ctx.String(http.StatusOK, "posts list")
	})

	posts.Get("/:id", func(ctx *Context) {
		ctx.String(http.StatusOK, "post: %s", ctx.PathParam("id").Value)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users/123", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.ServeHTTP(w, req)
		w.Body.Reset()
	}
}

func BenchmarkComplexMiddlewareChain(b *testing.B) {
	s := NewHTTPServer()

	s.Middleware().Global().Add(
		func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				ctx.SetHeader("X-Global", "value")
				next(ctx)
			}
		},
	)

	s.Middleware().For("GET", "/api/*").Add(
		func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				ctx.SetHeader("X-API", "value")
				next(ctx)
			}
		},
	)

	s.Middleware().When(func(c *Context) bool {
		return c.GetHeader("X-Test") != ""
	}).Add(
		func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				ctx.SetHeader("X-Conditional", "true")
				next(ctx)
			}
		},
	)

	api := s.Group("/api").Use(
		func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				ctx.SetHeader("X-API-Group", "true")
				next(ctx)
			}
		},
	)

	api.Get("/users", func(ctx *Context) {
		ctx.String(http.StatusOK, "API users")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.Header.Set("X-Test", "1")
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.ServeHTTP(w, req)
		w.Body.Reset()
	}
}

func BenchmarkFullRequest(b *testing.B) {
	s := NewHTTPServer()

	s.Use("", "/*", func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			next(ctx)
		}
	})

	api := s.Group("/api")

	v1 := api.Group("/v1")

	users := v1.Group("/users")
	users.Get("", func(ctx *Context) {
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"users": []map[string]interface{}{
				{"id": 1, "name": "Alice"},
				{"id": 2, "name": "Bob"},
			},
		})
	})

	users.Get("/:id", func(ctx *Context) {
		id := ctx.PathParam("id").Value
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"id":   id,
			"name": "User " + id,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.ServeHTTP(w, req)
		io.Copy(io.Discard, w.Body)
		w.Body.Reset()
	}
}

func BenchmarkNotFound(b *testing.B) {
	s := NewHTTPServer()

	s.Get("/users", func(ctx *Context) {
		ctx.String(http.StatusOK, "users")
	})

	req := httptest.NewRequest(http.MethodGet, "/not-found", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.ServeHTTP(w, req)
		w.Body.Reset()
	}
}
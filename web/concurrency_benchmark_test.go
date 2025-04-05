package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func BenchmarkConcurrentRequests(b *testing.B) {
	server := NewHTTPServer()

	server.Get("/text", func(ctx *Context) {
		ctx.String(http.StatusOK, "Hello, World!")
	})

	server.Get("/json", func(ctx *Context) {
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"message": "Hello, World!",
			"status":  "success",
		})
	})

	server.Get("/users/:id", func(ctx *Context) {
		id := ctx.PathParam("id").Value
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"id":   id,
			"name": "User " + id,
		})
	})

	logMiddleware := func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			next(ctx)
		}
	}

	server.Get("/with-middleware", func(ctx *Context) {
		ctx.String(http.StatusOK, "Request with middleware")
	}).Middleware(logMiddleware)

	testCases := []struct {
		name       string
		path       string
		concurrent int
	}{
		{"SimpleText", "/text", 10},
		{"SimpleText", "/text", 50},
		{"SimpleText", "/text", 100},
		{"JSON", "/json", 10},
		{"JSON", "/json", 50},
		{"JSON", "/json", 100},
		{"PathParam", "/users/123", 10},
		{"PathParam", "/users/123", 50},
		{"PathParam", "/users/123", 100},
		{"WithMiddleware", "/with-middleware", 10},
		{"WithMiddleware", "/with-middleware", 50},
		{"WithMiddleware", "/with-middleware", 100},
	}

	for _, tc := range testCases {
		b.Run(fmt.Sprintf("%s_Concurrent%d", tc.name, tc.concurrent), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				wg.Add(tc.concurrent)

				for j := 0; j < tc.concurrent; j++ {
					go func() {
						defer wg.Done()
						req := httptest.NewRequest(http.MethodGet, tc.path, nil)
						recorder := httptest.NewRecorder()
						server.ServeHTTP(recorder, req)
					}()
				}

				wg.Wait()
			}
		})
	}
}

func BenchmarkWithRealHTTPServer(b *testing.B) {
	server := NewHTTPServer()

	server.Get("/ping", func(ctx *Context) {
		ctx.String(http.StatusOK, "pong")
	})

	server.Post("/echo", func(ctx *Context) {
		body, err := ctx.ReadBody()
		if err != nil {
			ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		ctx.String(http.StatusOK, string(body))
	})

	go func() {
		err := server.Start(":0")
		if !errors.Is(err, http.ErrServerClosed) {
			b.Fatalf("Failed to start server: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	concurrencyLevels := []int{10, 50, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency%d", concurrency), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				wg.Add(concurrency)

				for j := 0; j < concurrency; j++ {
					go func() {
						defer wg.Done()
						req := httptest.NewRequest(http.MethodGet, "/ping", nil)
						recorder := httptest.NewRecorder()
						server.ServeHTTP(recorder, req)
					}()
				}

				wg.Wait()
			}
		})
	}
}

func BenchmarkRouteGroupConcurrency(b *testing.B) {
	server := NewHTTPServer()

	api := server.Group("/api")

	api.Get("/users", func(ctx *Context) {
		ctx.JSON(http.StatusOK, []map[string]interface{}{
			{"id": 1, "name": "User 1"},
			{"id": 2, "name": "User 2"},
		})
	})

	api.Get("/users/:id", func(ctx *Context) {
		id := ctx.PathParam("id").Value
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"id":   id,
			"name": "User " + id,
		})
	})

	admin := api.Group("/admin")
	admin.Get("/stats", func(ctx *Context) {
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"active_users": 100,
			"total_users":  500,
		})
	})

	testPaths := []string{
		"/api/users",
		"/api/users/42",
		"/api/admin/stats",
	}

	for _, path := range testPaths {
		b.Run(fmt.Sprintf("Path_%s", path), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				wg.Add(50)

				for j := 0; j < 50; j++ {
					go func() {
						defer wg.Done()
						req := httptest.NewRequest(http.MethodGet, path, nil)
						recorder := httptest.NewRecorder()
						server.ServeHTTP(recorder, req)
					}()
				}

				wg.Wait()
			}
		})
	}
}
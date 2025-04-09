package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	objPool "github.com/fyerfyer/fyer-webframe/web/pool"
)

func BenchmarkContextCreation(b *testing.B) {
	b.Run("CreateNewContext", func(b *testing.B) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp := httptest.NewRecorder()

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ctx := &Context{
				Req:         req,
				Resp:        resp,
				Param:       make(map[string]string, 8),
				UserValues:  make(map[string]any, 8),
				Context:     req.Context(),
				unhandled:   true,
			}
			_ = ctx
		}
	})

	b.Run("AcquireFromPool", func(b *testing.B) {
		objPool.DefaultContextPool = nil
		InitContextPool(nil, nil, 8)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp := httptest.NewRecorder()

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ctx := AcquireContext(req, resp)
			ReleaseContext(ctx)
		}
	})
}

func BenchmarkContextReset(b *testing.B) {
	ctx := &Context{
		Req:         httptest.NewRequest(http.MethodGet, "/test", nil),
		Resp:        httptest.NewRecorder(),
		Param:       make(map[string]string, 8),
		UserValues:  make(map[string]any, 8),
		RespData:    []byte("test data"),
		Context:     context.Background(),
		unhandled:   false,
		aborted:     true,
	}

	// Add some data to maps
	ctx.Param["id"] = "123"
	ctx.Param["name"] = "test"
	ctx.UserValues["key1"] = "value1"
	ctx.UserValues["key2"] = 42

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx.Reset()

		// Setup for next iteration
		ctx.Req = httptest.NewRequest(http.MethodGet, "/test", nil)
		ctx.Resp = httptest.NewRecorder()
		ctx.Context = ctx.Req.Context()
		ctx.RespData = []byte("test data")
		ctx.Param["id"] = "123"
		ctx.Param["name"] = "test"
		ctx.UserValues["key1"] = "value1"
		ctx.UserValues["key2"] = 42
		ctx.unhandled = false
		ctx.aborted = true
	}
}

func BenchmarkContextSetup(b *testing.B) {
	b.Run("ManualSetup", func(b *testing.B) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp := httptest.NewRecorder()

		ctx := &Context{
			Param:      make(map[string]string, 8),
			UserValues: make(map[string]any, 8),
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ctx.SetRequest(req)
			ctx.SetResponse(resp)
		}
	})

	b.Run("DirectAssignment", func(b *testing.B) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp := httptest.NewRecorder()

		ctx := &Context{
			Param:      make(map[string]string, 8),
			UserValues: make(map[string]any, 8),
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ctx.Req = req
			ctx.Resp = resp
			ctx.Context = req.Context()
		}
	})
}

func BenchmarkConcurrentContextPool(b *testing.B) {
	objPool.DefaultContextPool = nil
	InitContextPool(nil, nil, 8)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp := httptest.NewRecorder()

			ctx := AcquireContext(req, resp)

			// Simulate some work with the context
			ctx.Param["id"] = "123"
			ctx.UserValues["key"] = "value"
			ctx.RespStatusCode = 200

			ReleaseContext(ctx)
		}
	})
}

func BenchmarkContextPoolVsNew(b *testing.B) {
	objPool.DefaultContextPool = nil
	InitContextPool(nil, nil, 8)

	scenarios := []struct {
		name     string
		poolSize int
	}{
		{"Small", 10},
		{"Medium", 100},
		{"Large", 1000},
	}

	for _, scenario := range scenarios {
		b.Run("Pool"+scenario.name, func(b *testing.B) {
			// Pre-warm the pool
			contexts := make([]*Context, scenario.poolSize)
			for i := 0; i < scenario.poolSize; i++ {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				resp := httptest.NewRecorder()
				contexts[i] = AcquireContext(req, resp)
			}
			for i := 0; i < scenario.poolSize; i++ {
				ReleaseContext(contexts[i])
			}

			b.ReportAllocs()
			b.ResetTimer()

			var wg sync.WaitGroup
			wg.Add(b.N)

			for i := 0; i < b.N; i++ {
				go func() {
					defer wg.Done()
					req := httptest.NewRequest(http.MethodGet, "/test", nil)
					resp := httptest.NewRecorder()

					ctx := AcquireContext(req, resp)
					ctx.Param["id"] = "123"
					ctx.UserValues["key"] = "value"

					ReleaseContext(ctx)
				}()
			}

			wg.Wait()
		})

		b.Run("New"+scenario.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			var wg sync.WaitGroup
			wg.Add(b.N)

			for i := 0; i < b.N; i++ {
				go func() {
					defer wg.Done()
					req := httptest.NewRequest(http.MethodGet, "/test", nil)
					resp := httptest.NewRecorder()

					ctx := &Context{
						Req:         req,
						Resp:        resp,
						Param:       make(map[string]string, 8),
						UserValues:  make(map[string]any, 8),
						Context:     req.Context(),
						unhandled:   true,
					}

					ctx.Param["id"] = "123"
					ctx.UserValues["key"] = "value"

					// No cleanup needed, let GC handle it
					_ = ctx
				}()
			}

			wg.Wait()
		})
	}
}

func BenchmarkContextOperationsWithPool(b *testing.B) {
	objPool.DefaultContextPool = nil
	InitContextPool(nil, nil, 8)

	b.Run("SetGetResetRelease", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp := httptest.NewRecorder()

			ctx := AcquireContext(req, resp)

			// Set operations
			ctx.SetHeader("Content-Type", "application/json")
			ctx.Status(http.StatusOK)

			// Get operations
			_ = ctx.ClientIP()
			_ = ctx.UserAgent()
			_ = ctx.QueryParam("q")

			// Param operations
			ctx.Param["id"] = "123"
			_ = ctx.PathParam("id")

			// Reset and release
			ReleaseContext(ctx)
		}
	})
}

func BenchmarkMultipleContextParameters(b *testing.B) {
	paramSizes := []int{2, 8, 16, 32}

	for _, size := range paramSizes {
		name := "Params" + string(rune(size))

		b.Run("WithPool"+name, func(b *testing.B) {
			objPool.DefaultContextPool = nil
			InitContextPool(nil, nil, size)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				resp := httptest.NewRecorder()

				ctx := AcquireContext(req, resp)

				// Fill with parameters
				for j := 0; j < size; j++ {
					ctx.Param[string(rune('a'+j))] = string(rune('1'+j))
				}

				ReleaseContext(ctx)
			}
		})

		b.Run("WithoutPool"+name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				resp := httptest.NewRecorder()

				ctx := &Context{
					Req:         req,
					Resp:        resp,
					Param:       make(map[string]string, size),
					UserValues:  make(map[string]any, size),
					Context:     req.Context(),
					unhandled:   true,
				}

				// Fill with parameters
				for j := 0; j < size; j++ {
					ctx.Param[string(rune('a'+j))] = string(rune('1'+j))
				}

				// Just let GC collect it
				_ = ctx
			}
		})
	}
}
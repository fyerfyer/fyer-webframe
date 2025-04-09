package web

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	objPool "github.com/fyerfyer/fyer-webframe/web/pool"
)

func BenchmarkServerWithPool(b *testing.B) {
	server := NewHTTPServer(WithObjectPool(10))

	server.Get("/benchmark", func(ctx *Context) {
		ctx.String(http.StatusOK, "Hello World!")
	})

	req := httptest.NewRequest(http.MethodGet, "/benchmark", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			b.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
		}
	}
}

func BenchmarkServerWithoutPool(b *testing.B) {
	server := NewHTTPServer()

	server.Get("/benchmark", func(ctx *Context) {
		ctx.String(http.StatusOK, "Hello World!")
	})

	req := httptest.NewRequest(http.MethodGet, "/benchmark", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			b.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
		}
	}
}

func BenchmarkContextAcquire(b *testing.B) {
	objPool.DefaultContextPool = nil
	InitContextPool(nil, nil, 8)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := AcquireContext(req, resp)
		ReleaseContext(ctx)
	}
}

func BenchmarkContextCreate(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = &Context{
			Req:        req,
			Resp:       resp,
			Param:      make(map[string]string, 8),
			Context:    req.Context(),
			unhandled:  true,
			UserValues: make(map[string]any, 8),
		}
	}
}

type benchmarkData struct {
	Message string `json:"message" xml:"message"`
	Status  string `json:"status" xml:"status"`
	Code    int    `json:"code" xml:"code"`
	Items   []int  `json:"items" xml:"items"`
}

func BenchmarkResponseJSONWithPool(b *testing.B) {
	objPool.DefaultContextPool = nil
	InitContextPool(nil, nil, 8)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp := httptest.NewRecorder()

	data := benchmarkData{
		Message: "Hello World",
		Status:  "success",
		Code:    200,
		Items:   []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := AcquireContext(req, resp)
		ctx.JSON(http.StatusOK, data)
		ReleaseContext(ctx)
	}
}

func BenchmarkResponseJSONWithoutPool(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp := httptest.NewRecorder()

	data := benchmarkData{
		Message: "Hello World",
		Status:  "success",
		Code:    200,
		Items:   []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := &Context{
			Req:        req,
			Resp:       resp,
			Param:      make(map[string]string),
			Context:    req.Context(),
			unhandled:  true,
			UserValues: make(map[string]any),
		}

		resp.Header().Set("Content-Type", ContentTypeJSON)
		ctx.RespStatusCode = http.StatusOK

		var buf bytes.Buffer
		json.NewEncoder(&buf).Encode(data)

		ctx.RespData = make([]byte, buf.Len())
		copy(ctx.RespData, buf.Bytes())
	}
}

func BenchmarkResponseXMLWithPool(b *testing.B) {
	objPool.DefaultContextPool = nil
	InitContextPool(nil, nil, 8)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp := httptest.NewRecorder()

	data := benchmarkData{
		Message: "Hello World",
		Status:  "success",
		Code:    200,
		Items:   []int{1, 2, 3, 4, 5},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := AcquireContext(req, resp)
		ctx.XML(http.StatusOK, data)
		ReleaseContext(ctx)
	}
}

func BenchmarkResponseXMLWithoutPool(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp := httptest.NewRecorder()

	data := benchmarkData{
		Message: "Hello World",
		Status:  "success",
		Code:    200,
		Items:   []int{1, 2, 3, 4, 5},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := &Context{
			Req:        req,
			Resp:       resp,
			Param:      make(map[string]string),
			Context:    req.Context(),
			unhandled:  true,
			UserValues: make(map[string]any),
		}

		resp.Header().Set("Content-Type", ContentTypeXML)
		ctx.RespStatusCode = http.StatusOK

		var buf bytes.Buffer
		buf.WriteString(xml.Header)
		encoder := xml.NewEncoder(&buf)
		encoder.Indent("", "  ")
		encoder.Encode(data)

		ctx.RespData = make([]byte, buf.Len())
		copy(ctx.RespData, buf.Bytes())
	}
}

func BenchmarkResponseStringWithPool(b *testing.B) {
	objPool.DefaultContextPool = nil
	InitContextPool(nil, nil, 8)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := AcquireContext(req, resp)
		ctx.String(http.StatusOK, "Hello %s with iteration %d", "World", i)
		ReleaseContext(ctx)
	}
}

func BenchmarkResponseStringWithoutPool(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := &Context{
			Req:        req,
			Resp:       resp,
			Param:      make(map[string]string),
			Context:    req.Context(),
			unhandled:  true,
			UserValues: make(map[string]any),
		}

		resp.Header().Set("Content-Type", ContentTypePlain)
		ctx.RespStatusCode = http.StatusOK

		var buf bytes.Buffer
		fmt.Fprintf(&buf, "Hello %s with iteration %d", "World", i)

		ctx.RespData = make([]byte, buf.Len())
		copy(ctx.RespData, buf.Bytes())
	}
}

func BenchmarkConcurrentRequestsWithPool(b *testing.B) {
	server := NewHTTPServer(WithObjectPool(10))

	server.Get("/benchmark", func(ctx *Context) {
		id := ctx.QueryParam("id").Value
		ctx.JSON(http.StatusOK, map[string]string{
			"message": "Hello, " + id,
			"status":  "success",
		})
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			counter++
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/benchmark?id=%d", counter), nil)
			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusOK {
				b.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
			}
		}
	})
}

func BenchmarkConcurrentRequestsWithoutPool(b *testing.B) {
	server := NewHTTPServer()

	server.Get("/benchmark", func(ctx *Context) {
		id := ctx.QueryParam("id").Value
		ctx.JSON(http.StatusOK, map[string]string{
			"message": "Hello, " + id,
			"status":  "success",
		})
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			counter++
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/benchmark?id=%d", counter), nil)
			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusOK {
				b.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
			}
		}
	})
}

func BenchmarkMixedOperationsWithPool(b *testing.B) {
	server := NewHTTPServer(WithObjectPool(10))

	server.Get("/json", func(ctx *Context) {
		ctx.JSON(http.StatusOK, benchmarkData{
			Message: "Hello World",
			Status:  "success",
			Code:    200,
			Items:   []int{1, 2, 3, 4, 5},
		})
	})

	server.Get("/xml", func(ctx *Context) {
		ctx.XML(http.StatusOK, benchmarkData{
			Message: "Hello World",
			Status:  "success",
			Code:    200,
			Items:   []int{1, 2, 3, 4, 5},
		})
	})

	server.Get("/string", func(ctx *Context) {
		ctx.String(http.StatusOK, "Hello %s!", "World")
	})

	paths := []string{"/json", "/xml", "/string"}

	b.ResetTimer()

	var wg sync.WaitGroup
	wg.Add(b.N)

	for i := 0; i < b.N; i++ {
		go func(i int) {
			defer wg.Done()
			path := paths[i%len(paths)]
			req := httptest.NewRequest(http.MethodGet, path, nil)
			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, req)
		}(i)
	}

	wg.Wait()
}

func BenchmarkMixedOperationsWithoutPool(b *testing.B) {
	server := NewHTTPServer()

	server.Get("/json", func(ctx *Context) {
		ctx.JSON(http.StatusOK, benchmarkData{
			Message: "Hello World",
			Status:  "success",
			Code:    200,
			Items:   []int{1, 2, 3, 4, 5},
		})
	})

	server.Get("/xml", func(ctx *Context) {
		ctx.XML(http.StatusOK, benchmarkData{
			Message: "Hello World",
			Status:  "success",
			Code:    200,
			Items:   []int{1, 2, 3, 4, 5},
		})
	})

	server.Get("/string", func(ctx *Context) {
		ctx.String(http.StatusOK, "Hello %s!", "World")
	})

	paths := []string{"/json", "/xml", "/string"}

	b.ResetTimer()

	var wg sync.WaitGroup
	wg.Add(b.N)

	for i := 0; i < b.N; i++ {
		go func(i int) {
			defer wg.Done()
			path := paths[i%len(paths)]
			req := httptest.NewRequest(http.MethodGet, path, nil)
			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, req)
		}(i)
	}

	wg.Wait()
}

func BenchmarkBufferPoolOperations(b *testing.B) {
	b.Run("GetReleaseBuffer", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf := objPool.AcquireBuffer()
			buf.Buffer.WriteString("test string")
			objPool.ReleaseBuffer(buf)
		}
	})

	b.Run("CreateReleaseBuffer", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			buf.WriteString("test string")
			// Just let it be GC'd
		}
	})

	b.Run("SizedBuffers", func(b *testing.B) {
		sizes := []int{1024, 4096, 16384, 65536}
		for _, size := range sizes {
			b.Run(fmt.Sprintf("Size-%d", size), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					buf := objPool.AcquireBufferSize(size)
					buf.Buffer.Grow(size)
					buf.Buffer.WriteString("test string with specific size")
					objPool.ReleaseBufferSize(buf, size)
				}
			})
		}
	})
}

func init() {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	objPool.DefaultContextPool = nil
	InitContextPool(nil, nil, 10)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp := httptest.NewRecorder()

	for i := 0; i < 100; i++ {
		ctx := AcquireContext(req, resp)
		ReleaseContext(ctx)

		buf := objPool.AcquireBuffer()
		objPool.ReleaseBuffer(buf)
	}
}

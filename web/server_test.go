package web

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"sync"
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
			if err != nil && err != http.ErrServerClosed {
                t.Errorf("unexpected server error: %v", err)
            }
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

func TestConcurrentRequests(t *testing.T) {
	s := NewHTTPServer()

	// 注册多个路由
	s.Get("/users/:id", func(ctx *Context) {
		id := ctx.PathParam("id").Value
		time.Sleep(10 * time.Millisecond)
		ctx.RespJSON(http.StatusOK, map[string]string{"id": id})
	})

	s.Post("/users", func(ctx *Context) {
		time.Sleep(20 * time.Millisecond)
		ctx.RespJSON(http.StatusCreated, map[string]string{"result": "created"})
	})

	s.Get("/heavy", func(ctx *Context) {
		time.Sleep(100 * time.Millisecond) // 模拟耗时请求
		ctx.RespString(http.StatusOK, "done")
	})

	s.Use("GET", "/*", func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			ctx.SetHeader("X-Request-ID", "test-id")
			next(ctx)
		}
	})

	t.Run("concurrent requests", func(t *testing.T) {
		var wg sync.WaitGroup
		requestCount := 100
		responses := make([]*http.Response, requestCount)

		for i := 0; i < requestCount; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				// 创建不同的混合请求
				var req *http.Request
				switch idx % 3 {
				case 0:
					req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/users/%d", idx), nil)
				case 1:
					req = httptest.NewRequest(http.MethodPost, "/users", nil)
				case 2:
					req = httptest.NewRequest(http.MethodGet, "/heavy", nil)
				}

				w := httptest.NewRecorder()
				s.ServeHTTP(w, req)
				responses[idx] = w.Result()
			}(i)
		}

		wg.Wait()

		for i, resp := range responses {
			if !assert.NotNil(t, resp, "Response %d should not be nil", i) {
				continue
			}

			assert.Less(t, resp.StatusCode, 400, "Request %d should have success status, got %d", i, resp.StatusCode)

			if i%3 != 1 {
				assert.Equal(t, "test-id", resp.Header.Get("X-Request-ID"), "Request %d missing header", i)
			}
		}
	})
}

func TestContextModificationConcurrency(t *testing.T) {
	s := NewHTTPServer()

	s.Get("/shared", func(ctx *Context) {
		if ctx.UserValues == nil {
			ctx.UserValues = make(map[string]any)
		}

		reqID := ctx.Req.Header.Get("X-Request-ID")
		ctx.UserValues["reqID"] = reqID

		// 返回请求ID以确保获得了正确的数据
		ctx.RespJSON(http.StatusOK, map[string]string{
			"id": reqID,
		})
	})

	t.Run("context isolation", func(t *testing.T) {
		var wg sync.WaitGroup
		requestCount := 50

		for i := 0; i < requestCount; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				reqID := fmt.Sprintf("req-%d", idx)
				req := httptest.NewRequest(http.MethodGet, "/shared", nil)
				req.Header.Set("X-Request-ID", reqID)

				w := httptest.NewRecorder()
				s.ServeHTTP(w, req)

				resp := w.Result()
				require.Equal(t, http.StatusOK, resp.StatusCode)

				var body map[string]string
				err := json.NewDecoder(resp.Body).Decode(&body)
				require.NoError(t, err)
				assert.Equal(t, reqID, body["id"], "Context was modified by another request")
			}(i)
		}

		wg.Wait()
	})
}

func TestServerGracefulShutdown(t *testing.T) {
	s := NewHTTPServer()

	s.Get("/long-running", func(ctx *Context) {
		select {
		case <-ctx.Req.Context().Done():
			return
		case <-time.After(500 * time.Millisecond):
			ctx.RespString(http.StatusOK, "completed")
		}
	})

	server := &http.Server{
		Addr:    ":8082",
		Handler: s,
	}
	s.server = server

	go func() {
		err := s.server.ListenAndServe()
		if err != http.ErrServerClosed {
			t.Errorf("unexpected server error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	var clientErr error
	go func() {
		client := &http.Client{
			Timeout: 2 * time.Second,
		}
		_, clientErr = client.Get("http://localhost:8082/long-running")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := s.Shutdown(ctx)
	assert.NoError(t, err, "Graceful shutdown failed")
	assert.NoError(t, clientErr, "Client request should complete despite shutdown")
}

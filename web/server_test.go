package web

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/fyerfyer/fyer-kit/pool"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerBasicFunctionality(t *testing.T) {
	s := NewHTTPServer()

	// 测试基本路由注册和处理
	s.Get("/", func(ctx *Context) {
		ctx.String(http.StatusOK, "hello")
	})

	s.Get("/user", func(ctx *Context) {
		ctx.String(http.StatusOK, "user")
	})

	// 测试HTTP请求
	t.Run("basic routes", func(t *testing.T) {
		// 测试根路径
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		resp := httptest.NewRecorder()
		s.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "hello", resp.Body.String())

		// 测试用户路径
		req = httptest.NewRequest(http.MethodGet, "/user", nil)
		resp = httptest.NewRecorder()
		s.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "user", resp.Body.String())

		// 测试404
		req = httptest.NewRequest(http.MethodGet, "/not-exist", nil)
		resp = httptest.NewRecorder()
		s.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestRouteRegistrationChaining(t *testing.T) {
	s := NewHTTPServer()

	// 记录中间件调用
	var order []string

	// 测试链式路由注册API
	t.Run("route registration chaining", func(t *testing.T) {
		s.Get("/chained", func(ctx *Context) {
			order = append(order, "handler")
			ctx.String(http.StatusOK, "chained handler")
		}).Middleware(
			func(next HandlerFunc) HandlerFunc {
				return func(ctx *Context) {
					order = append(order, "middleware1 before")
					next(ctx)
					order = append(order, "middleware1 after")
				}
			},
			func(next HandlerFunc) HandlerFunc {
				return func(ctx *Context) {
					order = append(order, "middleware2 before")
					next(ctx)
					order = append(order, "middleware2 after")
				}
			},
		)

		req := httptest.NewRequest(http.MethodGet, "/chained", nil)
		resp := httptest.NewRecorder()
		s.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "chained handler", resp.Body.String())

		// 验证中间件执行顺序
		expectedOrder := []string{
			"middleware1 before",
			"middleware2 before",
			"handler",
			"middleware2 after",
			"middleware1 after",
		}
		assert.Equal(t, expectedOrder, order)
	})
}

func TestRouteGroups(t *testing.T) {
	s := NewHTTPServer()

	// 设置接收到的请求计数
	userCount := 0
	adminCount := 0
	publicCount := 0

	// 创建API组
	api := s.Group("/api")

	// 注册API组路由
	api.Get("/public", func(ctx *Context) {
		publicCount++
		ctx.JSON(http.StatusOK, map[string]string{"area": "public"})
	})

	// 创建用户子组
	users := api.Group("/users")
	users.Get("", func(ctx *Context) { // 对应 /api/users
		userCount++
		ctx.JSON(http.StatusOK, map[string]string{"area": "users list"})
	})

	users.Get("/:id", func(ctx *Context) { // 对应 /api/users/:id
		userCount++
		id := ctx.PathParam("id").Value
		ctx.JSON(http.StatusOK, map[string]string{"user_id": id})
	})

	// 创建带中间件的管理员子组
	admin := api.Group("/admin").Use(
		func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				// 简单的授权检查
				authHeader := ctx.Req.Header.Get("Authorization")
				if authHeader != "AdminSecretKey" {
					ctx.Unauthorized("admin access required")
					return
				}
				next(ctx)
			}
		},
	)

	admin.Get("/dashboard", func(ctx *Context) {
		adminCount++
		ctx.JSON(http.StatusOK, map[string]string{"area": "admin"})
	})

	t.Run("route groups", func(t *testing.T) {
		// 测试公共API
		req := httptest.NewRequest(http.MethodGet, "/api/public", nil)
		resp := httptest.NewRecorder()
		s.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assertJSONResponse(t, resp, map[string]string{"area": "public"})
		assert.Equal(t, 1, publicCount)

		// 测试用户列表
		req = httptest.NewRequest(http.MethodGet, "/api/users", nil)
		resp = httptest.NewRecorder()
		s.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assertJSONResponse(t, resp, map[string]string{"area": "users list"})
		assert.Equal(t, 1, userCount)

		// 测试单个用户
		req = httptest.NewRequest(http.MethodGet, "/api/users/123", nil)
		resp = httptest.NewRecorder()
		s.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assertJSONResponse(t, resp, map[string]string{"user_id": "123"})
		assert.Equal(t, 2, userCount)

		// 测试未授权访问管理员
		req = httptest.NewRequest(http.MethodGet, "/api/admin/dashboard", nil)
		resp = httptest.NewRecorder()
		s.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
		assert.Equal(t, 0, adminCount, "Admin handler should not be called")

		// 测试授权访问管理员
		req = httptest.NewRequest(http.MethodGet, "/api/admin/dashboard", nil)
		req.Header.Set("Authorization", "AdminSecretKey")
		resp = httptest.NewRecorder()
		s.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assertJSONResponse(t, resp, map[string]string{"area": "admin"})
		assert.Equal(t, 1, adminCount)
	})
}

func TestMiddlewareManager(t *testing.T) {
	s := NewHTTPServer()
	var calls []string

	// 1. 全局中间件
	s.Middleware().Global().Add(
		func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				calls = append(calls, "global before")
				next(ctx)
				calls = append(calls, "global after")
			}
		},
	)

	// 2. 路径特定中间件
	s.Middleware().For("GET", "/api/users/*").Add(
		func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				calls = append(calls, "users before")
				next(ctx)
				calls = append(calls, "users after")
			}
		},
	)

	// 3. 条件中间件
	s.Middleware().When(func(c *Context) bool {
		// 只在请求头中有特定标志时应用
		return c.Req.Header.Get("X-Feature") == "enabled"
	}).Add(
		func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				calls = append(calls, "conditional before")
				next(ctx)
				calls = append(calls, "conditional after")
			}
		},
	)

	// 注册一些路由
	s.Get("/api/users/list", func(ctx *Context) {
		calls = append(calls, "handler users list")
		ctx.String(http.StatusOK, "users list")
	})

	s.Get("/api/admin", func(ctx *Context) {
		calls = append(calls, "handler admin")
		ctx.String(http.StatusOK, "admin page")
	})

	t.Run("middleware manager with path specific middleware", func(t *testing.T) {
		calls = []string{}

		req := httptest.NewRequest(http.MethodGet, "/api/users/list", nil)
		resp := httptest.NewRecorder()
		s.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "users list", resp.Body.String())

		expectedCalls := []string{
			"global before",
			"users before",
			"handler users list",
			"users after",
			"global after",
		}
		assert.Equal(t, expectedCalls, calls)
	})

	t.Run("middleware manager with conditional middleware", func(t *testing.T) {
		calls = []string{}

		req := httptest.NewRequest(http.MethodGet, "/api/admin", nil)
		req.Header.Set("X-Feature", "enabled")
		resp := httptest.NewRecorder()
		s.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "admin page", resp.Body.String())

		expectedCalls := []string{
			"global before",
			"conditional before",
			"handler admin",
			"conditional after",
			"global after",
		}
		assert.Equal(t, expectedCalls, calls)
	})

	t.Run("middleware manager without conditional middleware", func(t *testing.T) {
		calls = []string{}

		req := httptest.NewRequest(http.MethodGet, "/api/admin", nil)
		// 不设置 X-Feature 头，条件中间件不会触发
		resp := httptest.NewRecorder()
		s.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "admin page", resp.Body.String())

		expectedCalls := []string{
			"global before",
			"handler admin",
			"global after",
		}
		assert.Equal(t, expectedCalls, calls)
	})
}

func TestServerOptions(t *testing.T) {
	// 测试各种服务器选项
	customHandler := func(ctx *Context) {
		ctx.String(http.StatusNotFound, "custom 404 page")
	}

	s := NewHTTPServer(
		WithNotFoundHandler(customHandler),
		WithBasePath("/app"),
		WithReadTimeout(5*time.Second),
		WithWriteTimeout(10*time.Second),
	)

	// 注册一个应用路径下的路由
	s.Get("/hello", func(ctx *Context) {
		ctx.String(http.StatusOK, "hello from app")
	})

	t.Run("server with base path", func(t *testing.T) {
		// 测试基础路径下的路由
		req := httptest.NewRequest(http.MethodGet, "/app/hello", nil)
		resp := httptest.NewRecorder()
		s.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "hello from app", resp.Body.String())
	})

	t.Run("custom not found handler", func(t *testing.T) {
		// 测试自定义404处理器
		req := httptest.NewRequest(http.MethodGet, "/app/notexist", nil)
		resp := httptest.NewRecorder()
		s.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
		assert.Equal(t, "custom 404 page", resp.Body.String())
	})
}

func TestCombinedFeatures(t *testing.T) {
	s := NewHTTPServer()
	logs := []string{}

	// 定义一个日志中间件
	logMiddleware := func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			path := ctx.Req.URL.Path
			method := ctx.Req.Method
			logs = append(logs, fmt.Sprintf("%s %s started", method, path))

			start := time.Now()
			next(ctx)

			duration := time.Since(start)
			logs = append(logs, fmt.Sprintf("%s %s completed in %v", method, path, duration))
		}
	}

	// 设置全局中间件
	s.Middleware().Global().Add(logMiddleware)

	// 创建一个 API 组
	api := s.Group("/api").Use(func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			ctx.SetHeader("X-API-Version", "v1.0")
			next(ctx)
		}
	})

	// 添加用户路由组
	users := api.Group("/users")

	// 设置路由处理器
	users.Get("", func(ctx *Context) {
		ctx.JSON(http.StatusOK, []map[string]string{
			{"id": "1", "name": "Alice"},
			{"id": "2", "name": "Bob"},
		})
	})

	users.Get("/:id", func(ctx *Context) {
		id := ctx.PathParam("id").Value
		ctx.JSON(http.StatusOK, map[string]string{
			"id":   id,
			"name": "User " + id,
		})
	}).Middleware(func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			id := ctx.PathParam("id").Value
			logs = append(logs, fmt.Sprintf("Fetching user %s", id))
			next(ctx)
		}
	})

	t.Run("combined features test", func(t *testing.T) {
		logs = []string{}

		// 测试用户列表
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		resp := httptest.NewRecorder()
		s.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "v1.0", resp.Header().Get("X-API-Version"))

		// 解析响应体
		var users []map[string]string
		err := json.NewDecoder(resp.Body).Decode(&users)
		require.NoError(t, err)
		assert.Len(t, users, 2)

		// 验证日志
		assert.Contains(t, logs, "GET /api/users started")
		assert.Contains(t, logs[1], "GET /api/users completed in")

		// 测试单个用户（带路由参数和特定中间件）
		logs = []string{}
		req = httptest.NewRequest(http.MethodGet, "/api/users/42", nil)
		resp = httptest.NewRecorder()
		s.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "v1.0", resp.Header().Get("X-API-Version"))

		// 验证响应体
		var user map[string]string
		err = json.NewDecoder(resp.Body).Decode(&user)
		require.NoError(t, err)
		assert.Equal(t, "42", user["id"])
		assert.Equal(t, "User 42", user["name"])

		// 验证日志，应包含特定路由中间件的日志
		assert.Contains(t, logs, "GET /api/users/42 started")
		assert.Contains(t, logs, "Fetching user 42")
		assert.Contains(t, logs[2], "GET /api/users/42 completed in")
	})
}

func TestServerWithTemplate(t *testing.T) {
	// 创建临时模板
	tpl := NewGoTemplate()

	// 创建并设置服务器
	s := NewHTTPServer(WithTemplate(tpl))

	// 测试链式 UseTemplate 设置
	s2 := NewHTTPServer()
	s2.UseTemplate(tpl)

	// 注册路由
	s.Get("/template", func(ctx *Context) {
		// 在实际测试中，这里会渲染模板
		ctx.String(http.StatusOK, "template would be rendered")
	})

	t.Run("server with template", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/template", nil)
		resp := httptest.NewRecorder()
		s.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "template would be rendered", resp.Body.String())

		// 验证模板引擎被正确设置
		assert.Equal(t, tpl, s.tplEngine)
		assert.Equal(t, tpl, s2.tplEngine)
	})
}

func TestConcurrentRequests(t *testing.T) {
	s := NewHTTPServer()

	// 注册多个路由
	s.Get("/users/:id", func(ctx *Context) {
		id := ctx.PathParam("id").Value
		time.Sleep(10 * time.Millisecond)
		ctx.JSON(http.StatusOK, map[string]string{"id": id})
	})

	s.Post("/users", func(ctx *Context) {
		time.Sleep(20 * time.Millisecond)
		ctx.JSON(http.StatusCreated, map[string]string{"result": "created"})
	})

	s.Get("/heavy", func(ctx *Context) {
		time.Sleep(100 * time.Millisecond) // 模拟耗时请求
		ctx.String(http.StatusOK, "done")
	})

	s.Middleware().Global().Add(func(next HandlerFunc) HandlerFunc {
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

		// 验证所有响应
		for i, resp := range responses {
			if !assert.NotNil(t, resp, "Response %d should not be nil", i) {
				continue
			}

			assert.Less(t, resp.StatusCode, 400, "Request %d should have success status, got %d", i, resp.StatusCode)
			assert.Equal(t, "test-id", resp.Header.Get("X-Request-ID"), "Request %d missing header", i)

			// 关闭响应Body
			resp.Body.Close()
		}
	})
}

func TestServerGracefulShutdown(t *testing.T) {
	s := NewHTTPServer()

	// 注册一个耗时请求的处理器
	s.Get("/long-running", func(ctx *Context) {
		select {
		case <-ctx.Req.Context().Done():
			// 请求被取消
			return
		case <-time.After(500 * time.Millisecond):
			ctx.String(http.StatusOK, "completed")
		}
	})

	// 启动一个临时服务器用于测试
	addr := ":18082" // 使用不常用端口
	server := &http.Server{
		Addr:    addr,
		Handler: s,
	}
	s.server = server

	// 在goroutine中启动服务器
	go func() {
		err := s.Start(addr)
		if err != nil && err != http.ErrServerClosed {
			t.Logf("unexpected server error: %v", err)
		}
	}()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 发起长时间请求
	clientDone := make(chan struct{})
	var clientErr error
	go func() {
		defer close(clientDone)
		client := &http.Client{
			Timeout: 2 * time.Second,
		}
		resp, err := client.Get("http://localhost" + addr + "/long-running")
		if err != nil {
			clientErr = err
			return
		}
		defer resp.Body.Close()

		// 读取响应
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			clientErr = err
			return
		}

		if resp.StatusCode != http.StatusOK || string(body) != "completed" {
			clientErr = fmt.Errorf("unexpected response: %d - %s", resp.StatusCode, string(body))
		}
	}()

	// 等待一小段时间，确保请求已经开始
	time.Sleep(100 * time.Millisecond)

	// 优雅关闭
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := s.Shutdown(shutdownCtx)
	assert.NoError(t, err, "Graceful shutdown should not error")

	// 等待客户端完成
	<-clientDone
	assert.NoError(t, clientErr, "Client request should complete successfully despite shutdown")
}

func assertJSONResponse(t *testing.T, resp *httptest.ResponseRecorder, expected interface{}) {
	t.Helper()
	var actual interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &actual)
	require.NoError(t, err, "Response should be valid JSON")

	// 将预期值转换为JSON，再解析回来，以确保相同的类型
	expectedJSON, err := json.Marshal(expected)
	require.NoError(t, err)

	var expectedParsed interface{}
	err = json.Unmarshal(expectedJSON, &expectedParsed)
	require.NoError(t, err)

	assert.Equal(t, expectedParsed, actual, "JSON response should match expected value")
}

func TestServerPoolManagement(t *testing.T) {
	// 测试连接池的初始化和注册
	t.Run("pool initialization and registration", func(t *testing.T) {
		mockManager := NewMockPoolManager()

		// 给server创建PoolManger
		s := NewHTTPServer(WithPoolManager(mockManager))

		// 创建一些连接池
		pool1 := NewMockPool(mockManager)
		pool2 := NewMockPool(mockManager)

		require.NoError(t, mockManager.Register("db1", pool1))
		require.NoError(t, mockManager.Register("db2", pool2))

		// 验证池可被访问
		assert.Equal(t, mockManager, s.PoolManager())

		p1, err := s.PoolManager().Get("db1")
		require.NoError(t, err)
		assert.Equal(t, pool1, p1)

		p2, err := s.PoolManager().Get("db2")
		require.NoError(t, err)
		assert.Equal(t, pool2, p2)
	})

	// 测试服务器关闭时池的优雅关闭
	t.Run("graceful shutdown of pools", func(t *testing.T) {
		mockManager := NewMockPoolManager()

		pool1 := &MockPool{
			manager:     mockManager,
			connections: make([]pool.Connection, 0),
		}
		pool2 := &MockPool{
			manager:     mockManager,
			connections: make([]pool.Connection, 0),
		}

		require.NoError(t, mockManager.Register("db1", pool1))
		require.NoError(t, mockManager.Register("db2", pool2))

		s := NewHTTPServer(WithPoolManager(mockManager))

		s.Get("/use-pool", func(ctx *Context) {
			conn, err := ctx.GetConnection("db1")
			if err != nil {
				ctx.String(http.StatusInternalServerError, "Failed to get connection: %v", err)
				return
			}
			defer conn.Close()
			ctx.String(http.StatusOK, "Used pool connection: %v", conn.Raw())
		})

		ctx := context.Background()
		require.NoError(t, s.Shutdown(ctx))

		assert.Contains(t, mockManager.connLifecycle, "shutdown pool")
	})

	// 测试多个请求并发访问池
	t.Run("concurrent pool access", func(t *testing.T) {
		mockManager := NewMockPoolManager()

		sharedPool := &MockPool{
			manager:     mockManager,
			connections: make([]pool.Connection, 0),
		}
		require.NoError(t, mockManager.Register("db", sharedPool))

		s := NewHTTPServer(WithPoolManager(mockManager))
		s.Get("/concurrent-pool-access", func(ctx *Context) {
			conn, err := ctx.GetConnection("db")
			if err != nil {
				ctx.String(http.StatusInternalServerError, "Failed to get connection: %v", err)
				return
			}
			defer conn.Close()

			// 模拟连接池执行一些工作
			time.Sleep(10 * time.Millisecond)

			ctx.String(http.StatusOK, "Used connection: %v", conn.Raw())
		})

		ts := httptest.NewServer(s)
		defer ts.Close()

		var wg sync.WaitGroup
		concurrentRequests := 10
		results := make([]bool, concurrentRequests)

		for i := 0; i < concurrentRequests; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				resp, err := http.Get(ts.URL + "/concurrent-pool-access")
				if err != nil {
					t.Logf("Request %d failed: %v", index, err)
					results[index] = false
					return
				}
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Logf("Failed to read response body for request %d: %v", index, err)
					results[index] = false
					return
				}

				results[index] = resp.StatusCode == http.StatusOK &&
					strings.Contains(string(body), "Used connection:")
			}(i)
		}

		wg.Wait()

		successCount := 0
		for _, success := range results {
			if success {
				successCount++
			}
		}

		assert.Equal(t, concurrentRequests, successCount, "All concurrent requests should succeed")

		assert.GreaterOrEqual(t, sharedPool.currentConnID, 1, "At least one connection should have been created")

		assert.LessOrEqual(t, sharedPool.currentConnID, concurrentRequests,
			"Number of connections should be less than or equal to number of requests")
	})
}

// MockPoolManager 的这些代码已从 context_test.go 中提取，此处是为了参考

// func NewMockPoolManager() *MockPoolManager {
//     return &MockPoolManager{
//         pools:         make(map[string]pool.Pool),
//         connLifecycle: make([]string, 0),
//     }
// }

// func NewMockPool(manager *MockPoolManager) *MockPool {
//     return &MockPool{
//         manager:     manager,
//         connections: make([]pool.Connection, 0),
//     }
// }
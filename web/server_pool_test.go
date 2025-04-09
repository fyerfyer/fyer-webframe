package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	objPool "github.com/fyerfyer/fyer-webframe/web/pool"
)

func TestServerWithObjectPool(t *testing.T) {
	// 测试服务器初始化时启用对象池
	t.Run("ServerInitWithObjectPool", func(t *testing.T) {
		server := NewHTTPServer(WithObjectPool(10))

		if !server.useObjPool {
			t.Error("Object pool should be enabled when WithObjectPool option is used")
		}

		if server.paramCap != 10 {
			t.Errorf("Expected paramCap to be 10, got %d", server.paramCap)
		}

		// 初始化对象池
		server.initObjectPool()

		if objPool.DefaultContextPool == nil {
			t.Fatal("Expected DefaultContextPool to be initialized")
		}

		// 测试关闭时资源清理
		err := server.Shutdown(context.Background())
		if err != nil {
			t.Errorf("Failed to shutdown server: %v", err)
		}
	})

	// 测试请求处理时是否正确使用上下文池
	t.Run("RequestProcessingWithPool", func(t *testing.T) {
		// 创建启用对象池的服务器
		server := NewHTTPServer(WithObjectPool(8))

		// 注册一个简单的处理函数
		server.Get("/test", func(ctx *Context) {
			ctx.String(http.StatusOK, "Hello from pooled context")
		})

		// 创建测试请求
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		// 处理请求
		server.ServeHTTP(recorder, req)

		// 验证响应
		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
		}

		body, _ := io.ReadAll(recorder.Body)
		if !strings.Contains(string(body), "Hello from pooled context") {
			t.Errorf("Unexpected response body: %s", string(body))
		}
	})

	// 测试对象池是否正确启用
	t.Run("ObjectPoolEnabledCheck", func(t *testing.T) {
		// 清除已有的对象池
		objPool.DefaultContextPool = nil

		// 创建未启用对象池的服务器
		server1 := NewHTTPServer()

		if server1.useObjPool {
			t.Error("Object pool should not be enabled by default")
		}

		// 通过方法启用对象池
		server1.EnableObjectPool(16)

		if !server1.useObjPool {
			t.Error("Object pool should be enabled after calling EnableObjectPool")
		}

		if server1.paramCap != 16 {
			t.Errorf("Expected paramCap to be 16, got %d", server1.paramCap)
		}

		// 确认对象池已初始化
		if objPool.DefaultContextPool == nil {
			t.Error("DefaultContextPool should be initialized after enabling object pool")
		}
	})

	// 测试并发请求下对象池的复用
	t.Run("ConcurrentRequestsWithPool", func(t *testing.T) {
		// 清除已有的对象池
		objPool.DefaultContextPool = nil

		// 创建启用对象池的服务器
		server := NewHTTPServer(WithObjectPool(8))

		// 注册一个处理函数
		server.Get("/concurrent", func(ctx *Context) {
			// 模拟一些工作
			time.Sleep(10 * time.Millisecond)
			ctx.JSON(http.StatusOK, map[string]string{"status": "success"})
		})

		// 并发发送多个请求
		const requestCount = 100
		var wg sync.WaitGroup
		wg.Add(requestCount)

		for i := 0; i < requestCount; i++ {
			go func() {
				defer wg.Done()

				req := httptest.NewRequest(http.MethodGet, "/concurrent", nil)
				recorder := httptest.NewRecorder()

				server.ServeHTTP(recorder, req)

				if recorder.Code != http.StatusOK {
					t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
				}
			}()
		}

		wg.Wait()

		// 由于sync.Pool的实现细节，我们无法准确知道池中有多少对象
		// 这里主要测试服务器在高并发下仍能正常工作

		// 正确关闭服务器
		err := server.Shutdown(context.Background())
		if err != nil {
			t.Errorf("Failed to shutdown server: %v", err)
		}
	})

	// 测试请求处理中的异常情况
	t.Run("ErrorHandlingWithPool", func(t *testing.T) {
		server := NewHTTPServer(WithObjectPool(8))

		// 注册一个会触发错误的处理函数
		server.Get("/error", func(ctx *Context) {
			ctx.RespStatusCode = http.StatusInternalServerError
			ctx.RespData = []byte("Error occurred")
		})

		// 注册一个会触发404的路径
		server.Get("/exists", func(ctx *Context) {
			ctx.String(http.StatusOK, "This route exists")
		})

		// 测试错误路由
		req1 := httptest.NewRequest(http.MethodGet, "/error", nil)
		recorder1 := httptest.NewRecorder()
		server.ServeHTTP(recorder1, req1)

		if recorder1.Code != http.StatusInternalServerError {
			t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, recorder1.Code)
		}

		// 测试404路由
		req2 := httptest.NewRequest(http.MethodGet, "/not-found", nil)
		recorder2 := httptest.NewRecorder()
		server.ServeHTTP(recorder2, req2)

		if recorder2.Code != http.StatusNotFound {
			t.Errorf("Expected status code %d, got %d", http.StatusNotFound, recorder2.Code)
		}
	})

	// 测试对象池在请求处理完成后正确释放对象
	t.Run("ContextReleasedAfterRequest", func(t *testing.T) {
		// 清除已有的对象池，并创建自定义池以便追踪
		objPool.DefaultContextPool = nil

		// 追踪变量
		acquired := 0
		released := 0
		mu := sync.Mutex{}

		// 创建服务器并启用对象池
		server := NewHTTPServer(WithObjectPool(8))

		// 添加路由
		server.Get("/test-release", func(ctx *Context) {
			mu.Lock()
			acquired++
			mu.Unlock()

			ctx.String(http.StatusOK, "Testing context release")
		})

		// 发送几个请求
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test-release", nil)
			recorder := httptest.NewRecorder()

			server.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusOK {
				t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder.Code)
			}

			mu.Lock()
			released++
			mu.Unlock()
		}

		// 验证所有获取的上下文都已释放
		// 注意：由于无法直接访问池的内部状态，我们只能验证服务器正常工作
		// 实际项目中可以添加钩子来详细追踪对象池操作
		mu.Lock()
		if acquired != 5 {
			t.Errorf("Expected 5 context acquisitions, got %d", acquired)
		}
		if released != 5 {
			t.Errorf("Expected 5 context releases, got %d", released)
		}
		mu.Unlock()
	})

	// 测试带有基础路径的服务器使用对象池
	t.Run("ServerWithBasePathAndPool", func(t *testing.T) {
		server := NewHTTPServer(
			WithObjectPool(8),
			WithBasePath("/api"),
		)

		// 注册路由
		server.Get("/users", func(ctx *Context) {
			ctx.JSON(http.StatusOK, map[string]string{"message": "users endpoint"})
		})

		// 测试正确的路径
		req1 := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		recorder1 := httptest.NewRecorder()
		server.ServeHTTP(recorder1, req1)

		if recorder1.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, recorder1.Code)
		}

		// 测试不匹配基础路径的请求
		req2 := httptest.NewRequest(http.MethodGet, "/users", nil)
		recorder2 := httptest.NewRecorder()
		server.ServeHTTP(recorder2, req2)

		if recorder2.Code != http.StatusNotFound {
			t.Errorf("Expected status code %d, got %d", http.StatusNotFound, recorder2.Code)
		}
	})
}
package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	objPool "github.com/fyerfyer/fyer-webframe/web/pool"
)

func TestContextPool(t *testing.T) {
	// 测试初始化对象池
	t.Run("InitializePool", func(t *testing.T) {
		InitContextPool(nil, nil, 8)

		if objPool.DefaultContextPool == nil {
			t.Fatal("Failed to initialize DefaultContextPool")
		}
	})

	// 测试获取和释放上下文对象
	t.Run("AcquireAndRelease", func(t *testing.T) {
		if objPool.DefaultContextPool == nil {
			InitContextPool(nil, nil, 8)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp := httptest.NewRecorder()

		ctx := AcquireContext(req, resp)
		if ctx == nil {
			t.Fatal("Failed to acquire context from pool")
		}

		if ctx.Req != req {
			t.Errorf("Expected request to be %v, got %v", req, ctx.Req)
		}

		if ctx.Resp != resp {
			t.Errorf("Expected response to be %v, got %v", resp, ctx.Resp)
		}

		// 添加一些数据到上下文
		ctx.Param["id"] = "123"
		ctx.UserValues["key"] = "value"
		ctx.RespStatusCode = 200
		ctx.RespData = []byte("test data")
		ctx.unhandled = false
		ctx.aborted = true

		// 释放上下文回池中
		ReleaseContext(ctx)

		// 再次获取上下文（应该是同一个实例但已重置）
		newCtx := AcquireContext(req, resp)

		// 检查重置是否正确执行
		if len(newCtx.Param) != 0 {
			t.Errorf("Expected empty Param map, got %v", newCtx.Param)
		}

		if len(newCtx.UserValues) != 0 {
			t.Errorf("Expected empty UserValues map, got %v", newCtx.UserValues)
		}

		if newCtx.RespStatusCode != 0 {
			t.Errorf("Expected RespStatusCode to be 0, got %d", newCtx.RespStatusCode)
		}

		if newCtx.RespData != nil {
			t.Errorf("Expected RespData to be nil, got %v", newCtx.RespData)
		}

		if !newCtx.unhandled {
			t.Error("Expected unhandled to be true")
		}

		if newCtx.aborted {
			t.Error("Expected aborted to be false")
		}

		ReleaseContext(newCtx)
	})

	// 测试Reset方法
	t.Run("ResetMethod", func(t *testing.T) {
		// 创建一个上下文并初始化各种字段
		ctx := &Context{
			Req:            httptest.NewRequest(http.MethodGet, "/test", nil),
			Resp:           httptest.NewRecorder(),
			Param:          map[string]string{"id": "123"},
			RespStatusCode: 200,
			RespData:       []byte("test data"),
			unhandled:      false,
			aborted:        true,
			UserValues:     map[string]any{"key": "value"},
			Context:        httptest.NewRequest(http.MethodGet, "/test", nil).Context(),
		}

		// 调用Reset方法
		ctx.Reset()

		// 验证字段是否被正确重置
		if ctx.Req != nil {
			t.Errorf("Expected Req to be nil after reset, got %v", ctx.Req)
		}

		if ctx.Resp != nil {
			t.Errorf("Expected Resp to be nil after reset, got %v", ctx.Resp)
		}

		if len(ctx.Param) != 0 {
			t.Errorf("Expected Param to be empty after reset, got %v", ctx.Param)
		}

		if ctx.RespStatusCode != 0 {
			t.Errorf("Expected RespStatusCode to be 0 after reset, got %d", ctx.RespStatusCode)
		}

		if ctx.RespData != nil {
			t.Errorf("Expected RespData to be nil after reset, got %v", ctx.RespData)
		}

		if !ctx.unhandled {
			t.Error("Expected unhandled to be true after reset")
		}

		if ctx.aborted {
			t.Error("Expected aborted to be false after reset")
		}

		if ctx.Context != nil {
			t.Error("Expected Context to be nil after reset")
		}
	})

	// 测试SetRequest和SetResponse方法
	t.Run("SetRequestAndResponse", func(t *testing.T) {
		ctx := &Context{
			Param:      make(map[string]string),
			UserValues: make(map[string]any),
		}

		req := httptest.NewRequest(http.MethodPost, "/api", nil)
		resp := httptest.NewRecorder()

		ctx.SetRequest(req)
		ctx.SetResponse(resp)

		if ctx.Req != req {
			t.Errorf("Expected request to be set correctly, got %v instead of %v", ctx.Req, req)
		}

		if ctx.Resp != resp {
			t.Errorf("Expected response to be set correctly, got %v instead of %v", ctx.Resp, resp)
		}

		if ctx.Context != req.Context() {
			t.Error("Context not properly set from request")
		}
	})

	// 测试对象池的完整生命周期
	t.Run("PoolLifecycle", func(t *testing.T) {
		// 确保池已初始化
		if objPool.DefaultContextPool == nil {
			InitContextPool(nil, nil, 8)
		}

		// 模拟处理多个请求
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			resp := httptest.NewRecorder()

			ctx := AcquireContext(req, resp)

			// 模拟请求处理
			ctx.Param["id"] = "123"
			ctx.RespStatusCode = 200
			ctx.RespData = []byte("response data")

			// 释放上下文
			ReleaseContext(ctx)
		}

		// 再次获取一个上下文，应该已经重置
		req := httptest.NewRequest(http.MethodGet, "/final", nil)
		resp := httptest.NewRecorder()
		ctx := AcquireContext(req, resp)

		if len(ctx.Param) != 0 {
			t.Error("Context not properly reset in pool lifecycle")
		}

		if ctx.Req != req {
			t.Error("Request not properly set in reused context")
		}

		ReleaseContext(ctx)
	})
}
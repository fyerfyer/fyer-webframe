package web

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"

	objPool "github.com/fyerfyer/fyer-webframe/web/pool"
)

func TestResponseBufferPool(t *testing.T) {
	// 测试基本的缓冲区创建和获取
	t.Run("BasicBufferAcquireAndRelease", func(t *testing.T) {
		// 获取默认缓冲区
		buf := objPool.AcquireBuffer()
		if buf == nil || buf.Buffer == nil {
			t.Fatal("Failed to acquire buffer from pool")
		}

		// 写入一些数据
		data := "test data for buffer pool"
		buf.Buffer.WriteString(data)

		if buf.Buffer.String() != data {
			t.Errorf("Expected buffer to contain '%s', got '%s'", data, buf.Buffer.String())
		}

		// 释放缓冲区
		objPool.ReleaseBuffer(buf)

		// 再次获取，应该是重置后的缓冲区
		newBuf := objPool.AcquireBuffer()
		if newBuf.Buffer.Len() != 0 {
			t.Error("Buffer should be empty after being reset and reacquired")
		}

		objPool.ReleaseBuffer(newBuf)
	})

	// 测试不同大小缓冲区的分配
	t.Run("DifferentSizeBuffers", func(t *testing.T) {
		// 小缓冲区
		smallBuf := objPool.AcquireBufferSize(1024)
		if smallBuf.Buffer.Cap() < 1024 {
			t.Errorf("Small buffer capacity should be at least 1024, got %d", smallBuf.Buffer.Cap())
		}
		objPool.ReleaseBufferSize(smallBuf, 1024)

		// 中等缓冲区
		mediumBuf := objPool.AcquireBufferSize(5*1024)
		if mediumBuf.Buffer.Cap() < 5*1024 {
			t.Errorf("Medium buffer capacity should be at least 5120, got %d", mediumBuf.Buffer.Cap())
		}
		objPool.ReleaseBufferSize(mediumBuf, 5*1024)

		// 大缓冲区
		largeBuf := objPool.AcquireBufferSize(20*1024)
		if largeBuf.Buffer.Cap() < 20*1024 {
			t.Errorf("Large buffer capacity should be at least 20480, got %d", largeBuf.Buffer.Cap())
		}
		objPool.ReleaseBufferSize(largeBuf, 20*1024)
	})

	// 测试缓冲区重置
	t.Run("BufferReset", func(t *testing.T) {
		buf := objPool.AcquireBuffer()
		buf.Buffer.WriteString("data that should be cleared")

		if buf.Buffer.Len() == 0 {
			t.Fatal("Buffer should contain data before reset")
		}

		buf.Reset()

		if buf.Buffer.Len() != 0 {
			t.Errorf("Buffer should be empty after reset, got length %d", buf.Buffer.Len())
		}

		objPool.ReleaseBuffer(buf)
	})

	// 测试释放nil缓冲区
	t.Run("NilBufferRelease", func(t *testing.T) {
		// 不应该发生panic
		objPool.ReleaseBuffer(nil)
		objPool.ReleaseBufferSize(nil, 1024)
	})

	// 测试自定义缓冲区池
	t.Run("CustomBufferPool", func(t *testing.T) {
		customSize := 16 * 1024
		customPool := objPool.NewResponseBufferPool(customSize)

		buf := customPool.Get()
		if buf.Buffer.Cap() < customSize {
			t.Errorf("Custom buffer capacity should be at least %d, got %d", customSize, buf.Buffer.Cap())
		}

		customPool.Put(buf)
	})
}

func TestContextResponseMethods(t *testing.T) {
	// 测试Context响应方法中使用的缓冲区池

	// 测试JSON响应
	t.Run("JSONResponseWithBufferPool", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp := httptest.NewRecorder()
		ctx := &Context{
			Req:         req,
			Resp:        resp,
			Param:       make(map[string]string),
			UserValues:  make(map[string]any),
			unhandled:   true,
		}

		testData := map[string]interface{}{
			"message": "hello world",
			"status":  "success",
			"code":    200,
		}

		err := ctx.JSON(http.StatusOK, testData)
		if err != nil {
			t.Fatalf("Failed to create JSON response: %v", err)
		}

		// 验证响应
		if ctx.RespStatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, ctx.RespStatusCode)
		}

		if resp.Header().Get("Content-Type") != ContentTypeJSON {
			t.Errorf("Expected Content-Type %s, got %s", ContentTypeJSON, resp.Header().Get("Content-Type"))
		}

		// 解析响应体
		var result map[string]interface{}
		if err := json.Unmarshal(ctx.RespData, &result); err != nil {
			t.Fatalf("Failed to unmarshal response data: %v", err)
		}

		// 验证数据
		if result["message"] != "hello world" {
			t.Errorf("Expected message 'hello world', got '%v'", result["message"])
		}

		if result["status"] != "success" {
			t.Errorf("Expected status 'success', got '%v'", result["status"])
		}
	})

	// 测试XML响应
	t.Run("XMLResponseWithBufferPool", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp := httptest.NewRecorder()
		ctx := &Context{
			Req:         req,
			Resp:        resp,
			Param:       make(map[string]string),
			UserValues:  make(map[string]any),
			unhandled:   true,
		}

		type TestData struct {
			Message string `xml:"message"`
			Status  string `xml:"status"`
			Code    int    `xml:"code"`
		}

		testData := TestData{
			Message: "hello world",
			Status:  "success",
			Code:    200,
		}

		err := ctx.XML(http.StatusOK, testData)
		if err != nil {
			t.Fatalf("Failed to create XML response: %v", err)
		}

		// 验证响应
		if ctx.RespStatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, ctx.RespStatusCode)
		}

		if resp.Header().Get("Content-Type") != ContentTypeXML {
			t.Errorf("Expected Content-Type %s, got %s", ContentTypeXML, resp.Header().Get("Content-Type"))
		}

		// 解析响应体
		var result TestData
		if err := xml.Unmarshal(ctx.RespData, &result); err != nil {
			t.Fatalf("Failed to unmarshal response data: %v", err)
		}

		// 验证数据
		if result.Message != "hello world" {
			t.Errorf("Expected message 'hello world', got '%s'", result.Message)
		}

		if result.Status != "success" {
			t.Errorf("Expected status 'success', got '%s'", result.Status)
		}
	})

	// 测试String响应
	t.Run("StringResponseWithBufferPool", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp := httptest.NewRecorder()
		ctx := &Context{
			Req:         req,
			Resp:        resp,
			Param:       make(map[string]string),
			UserValues:  make(map[string]any),
			unhandled:   true,
		}

		err := ctx.String(http.StatusOK, "Hello %s!", "World")
		if err != nil {
			t.Fatalf("Failed to create String response: %v", err)
		}

		// 验证响应
		if ctx.RespStatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, ctx.RespStatusCode)
		}

		if resp.Header().Get("Content-Type") != ContentTypePlain {
			t.Errorf("Expected Content-Type %s, got %s", ContentTypePlain, resp.Header().Get("Content-Type"))
		}

		// 验证响应体
		expected := "Hello World!"
		actual := string(ctx.RespData)
		if actual != expected {
			t.Errorf("Expected response body '%s', got '%s'", expected, actual)
		}
	})

	// 测试多个响应方法连续调用
	t.Run("MultipleResponseCalls", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		resp := httptest.NewRecorder()
		ctx := &Context{
			Req:         req,
			Resp:        resp,
			Param:       make(map[string]string),
			UserValues:  make(map[string]any),
			unhandled:   true,
		}

		// 连续调用几个响应方法，模拟高频使用场景
		for i := 0; i < 10; i++ {
			err := ctx.JSON(http.StatusOK, map[string]int{"index": i})
			if err != nil {
				t.Fatalf("Failed to create JSON response in iteration %d: %v", i, err)
			}
		}

		// 最后一次调用应该仍然正常工作
		var result map[string]int
		if err := json.Unmarshal(ctx.RespData, &result); err != nil {
			t.Fatalf("Failed to unmarshal final response data: %v", err)
		}

		if result["index"] != 9 {
			t.Errorf("Expected final index 9, got %d", result["index"])
		}
	})
}
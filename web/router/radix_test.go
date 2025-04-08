package router

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRadixTree_Add(t *testing.T) {
	tree := NewRadixTree()
	handler := func() {}

	// 测试基本路由添加
	tree.Add(http.MethodGet, "/", handler)
	tree.Add(http.MethodGet, "/users", handler)
	tree.Add(http.MethodGet, "/users/list", handler)
	tree.Add(http.MethodPost, "/users", handler)

	// 验证路由添加成功
	assert.Equal(t, 4, tree.Routes(), "Total routes should be 4")
	assert.Contains(t, tree.PrintTree(), "/users", "Route tree should contain /users path")

	// 测试重复路由应该覆盖旧路由
	newHandler := func() {}
	tree.Add(http.MethodGet, "/users", newHandler)

	// 提取路由并验证是新的处理器
	params := make(map[string]string)
	h, ok := tree.Find(http.MethodGet, "/users", params)
	assert.True(t, ok, "Should find /users route")
	assert.Equal(t, newHandler, h, "Handler should be overridden with new one")
}

func TestRadixTree_Find_Static(t *testing.T) {
	tree := NewRadixTree()
	handler1 := func() { /* 处理器1 */ }
	handler2 := func() { /* 处理器2 */ }

	// 添加静态路由
	tree.Add(http.MethodGet, "/", handler1)
	tree.Add(http.MethodGet, "/users", handler1)
	tree.Add(http.MethodGet, "/users/details", handler2)
	tree.Add(http.MethodPost, "/users", handler2)

	testCases := []struct {
		method      string
		path        string
		shouldFind  bool
		description string
	}{
		{http.MethodGet, "/", true, "Root path should match"},
		{http.MethodGet, "/users", true, "User path should match"},
		{http.MethodGet, "/users/details", true, "User details path should match"},
		{http.MethodGet, "/notfound", false, "Non-existent path should not match"},
		{http.MethodPost, "/users", true, "POST user path should match"},
		{http.MethodPost, "/", false, "POST root path should not match"},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			params := make(map[string]string)
			handler, found := tree.Find(tc.method, tc.path, params)
			assert.Equal(t, tc.shouldFind, found, tc.description)
			if found {
				assert.NotNil(t, handler, "Found route should have a handler")
			}
		})
	}
}

func TestRadixTree_Find_Param(t *testing.T) {
	tree := NewRadixTree()
	handler := func() {}

	// 添加参数路由
	tree.Add(http.MethodGet, "/users/:id", handler)
	tree.Add(http.MethodGet, "/posts/:postId/comments/:commentId", handler)

	testCases := []struct {
		path        string
		shouldFind  bool
		params      map[string]string
		description string
	}{
		{
			"/users/123",
			true,
			map[string]string{"id": "123"},
			"Single parameter path should match and extract correct parameter",
		},
		{
			"/posts/456/comments/789",
			true,
			map[string]string{"postId": "456", "commentId": "789"},
			"Multi-parameter path should match and extract all parameters",
		},
		{
			"/users",
			false,
			nil,
			"Path missing parameter part should not match",
		},
		{
			"/posts/123/comments",
			false,
			nil,
			"Path missing subsequent parameter should not match",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			params := make(map[string]string)
			_, found := tree.Find(http.MethodGet, tc.path, params)
			assert.Equal(t, tc.shouldFind, found, tc.description)

			if tc.shouldFind {
				for key, expectedValue := range tc.params {
					value, exists := params[key]
					assert.True(t, exists, "Parameter %s should exist", key)
					assert.Equal(t, expectedValue, value, "Parameter %s should have value %s", key, expectedValue)
				}
			}
		})
	}
}

func TestRadixTree_Find_Wildcard(t *testing.T) {
	tree := NewRadixTree()
	handler := func() {}

	// 添加通配符路由
	tree.Add(http.MethodGet, "/static/*", handler)
	tree.Add(http.MethodGet, "/files/documents/*", handler)

	testCases := []struct {
		path        string
		shouldFind  bool
		wildcard    string
		description string
	}{
		{
			"/static/css/style.css",
			true,
			"css/style.css",
			"Static wildcard should match any sub-path",
		},
		{
			"/static/",
			true,
			"",
			"Empty wildcard should also match",
		},
		{
			"/files/documents/report.pdf",
			true,
			"report.pdf",
			"Document wildcard should match single file",
		},
		{
			"/files/documents/2023/Q4/report.pdf",
			true,
			"2023/Q4/report.pdf",
			"Document wildcard should match multiple level path",
		},
		{
			"/images/logo.png",
			false,
			"",
			"Resource not under wildcard path should not match",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			params := make(map[string]string)
			_, found := tree.Find(http.MethodGet, tc.path, params)
			assert.Equal(t, tc.shouldFind, found, tc.description)

			if tc.shouldFind {
				wildcard, exists := params["*"]
				assert.True(t, exists, "Wildcard parameter should exist")
				assert.Equal(t, tc.wildcard, wildcard, "Wildcard value should be correct")
			}
		})
	}
}

func TestRadixTree_Find_Regex(t *testing.T) {
	tree := NewRadixTree()
	handler := func() {}

	// 添加正则路由
	tree.Add(http.MethodGet, "/users/:id([0-9]+)", handler)
	tree.Add(http.MethodGet, "/posts/:slug([a-z0-9-]+)", handler)
	tree.Add(http.MethodGet, "/files/:filename([^/]+\\.pdf)", handler)

	testCases := []struct {
		path        string
		shouldFind  bool
		param       map[string]string
		description string
	}{
		{
			"/users/123",
			true,
			map[string]string{"id": "123"},
			"Numeric ID should match regex",
		},
		{
			"/users/abc",
			false,
			nil,
			"Non-numeric ID should not match",
		},
		{
			"/posts/my-awesome-post-123",
			true,
			map[string]string{"slug": "my-awesome-post-123"},
			"Valid slug should match",
		},
		{
			"/posts/Invalid_Slug",
			false,
			nil,
			"Slug with uppercase and underscore should not match",
		},
		{
			"/files/document.pdf",
			true,
			map[string]string{"filename": "document.pdf"},
			"PDF file should match",
		},
		{
			"/files/document.txt",
			false,
			nil,
			"Non-PDF file should not match",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			params := make(map[string]string)
			_, found := tree.Find(http.MethodGet, tc.path, params)
			assert.Equal(t, tc.shouldFind, found, tc.description)

			if tc.shouldFind {
				for key, expectedValue := range tc.param {
					value, exists := params[key]
					assert.True(t, exists, "Parameter %s should exist", key)
					assert.Equal(t, expectedValue, value, "Parameter %s should have value %s", key, expectedValue)
				}
			}
		})
	}
}

func TestRadixTree_FindPriority(t *testing.T) {
	tree := NewRadixTree()

	// 定义不同的处理器来验证优先级
	staticHandler := func() { /* 静态处理器 */ }
	regexHandler := func() { /* 正则处理器 */ }
	paramHandler := func() { /* 参数处理器 */ }
	wildcardHandler := func() { /* 通配符处理器 */ }

	// 添加各种类型的路由，测试优先级：静态 > 正则 > 参数 > 通配符
	tree.Add(http.MethodGet, "/users/list", staticHandler)
	tree.Add(http.MethodGet, "/users/:id([0-9]+)", regexHandler)
	tree.Add(http.MethodGet, "/users/:name", paramHandler)
	tree.Add(http.MethodGet, "/users/*", wildcardHandler)

	// 测试优先级匹配
	tests := []struct {
		path           string
		expectedHandler interface{}
		description     string
	}{
		{"/users/list", staticHandler, "Static route should take precedence over parameter route"},
		{"/users/123", regexHandler, "Regex route should take precedence over plain parameter route"},
		{"/users/john", paramHandler, "Parameter route should take precedence over wildcard route"},
		{"/users/profile/edit", wildcardHandler, "Wildcard should match paths that don't match anything else"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			params := make(map[string]string)
			handler, found := tree.Find(http.MethodGet, tt.path, params)
			require.True(t, found, "Should find route: %s", tt.path)
			assert.Equal(t, tt.expectedHandler, handler, tt.description)
		})
	}
}

func TestRadixTree_HTTP_Methods(t *testing.T) {
	tree := NewRadixTree()
	handler := func() {}

	// 测试所有HTTP方法
	tree.GET("/users", handler)
	tree.POST("/users", handler)
	tree.PUT("/users/:id", handler)
	tree.DELETE("/users/:id", handler)
	tree.PATCH("/users/:id", handler)
	tree.OPTIONS("/users", handler)
	tree.HEAD("/users", handler)

	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
		http.MethodOptions,
		http.MethodHead,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			path := "/users"
			if method == http.MethodPut || method == http.MethodDelete || method == http.MethodPatch {
				path = "/users/1"
			}

			params := make(map[string]string)
			h, found := tree.Find(method, path, params)
			assert.True(t, found, "%s method should be correctly registered", method)
			assert.Equal(t, handler, h, "%s method should return correct handler", method)
		})
	}
}

func TestRadixTree_MixedRoutes(t *testing.T) {
	tree := NewRadixTree()
	handler := func() {}

	// 添加混合类型的路由
	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/"},
		{http.MethodGet, "/users"},
		{http.MethodGet, "/users/:id"},
		{http.MethodGet, "/users/:id/profile"},
		{http.MethodGet, "/posts/:id([0-9]+)"},
		{http.MethodGet, "/posts/:slug([a-z-]+)"},
		{http.MethodGet, "/files/*"},
		{http.MethodGet, "/static/*"},
		{http.MethodPost, "/users"},
	}

	for _, route := range routes {
		tree.Add(route.method, route.path, handler)
	}

	// 验证所有路由均可找到
	assert.Equal(t, len(routes), tree.Routes(), "Should correctly count total routes")

	// 随机测试几条路由
	testPaths := []struct {
		method      string
		path        string
		shouldFind  bool
		params      map[string]string
		description string
	}{
		{http.MethodGet, "/", true, map[string]string{}, "Root path should match"},
		{http.MethodGet, "/users/123", true, map[string]string{"id": "123"}, "User ID parameter should match"},
		{http.MethodGet, "/posts/123", true, map[string]string{"id": "123"}, "Numeric post ID should match"},
		{http.MethodGet, "/posts/hello-world", true, map[string]string{"slug": "hello-world"}, "Alphabetic post slug should match"},
		{http.MethodGet, "/files/document.pdf", true, map[string]string{"*": "document.pdf"}, "File wildcard should match"},
		{http.MethodGet, "/unknown", false, nil, "Unknown path should not match"},
	}

	for _, tc := range testPaths {
		t.Run(tc.path, func(t *testing.T) {
			params := make(map[string]string)
			_, found := tree.Find(tc.method, tc.path, params)
			assert.Equal(t, tc.shouldFind, found, tc.description)

			if tc.shouldFind {
				for k, v := range tc.params {
					param, exists := params[k]
					assert.True(t, exists, "Parameter %s should exist", k)
					assert.Equal(t, v, param, "Parameter %s value should be correct", k)
				}
			}
		})
	}
}

func TestRadixTree_EdgeCases(t *testing.T) {
	tree := NewRadixTree()
	handler := func() {}

	// 测试特殊情况
	tree.Add(http.MethodGet, "/", handler)
	tree.Add(http.MethodGet, "/:param", handler)
	tree.Add(http.MethodGet, "/users/:id/:action", handler)
	tree.Add(http.MethodGet, "/very/long/path/with/many/segments", handler)
	tree.Add(http.MethodGet, "/api/v1/:version/users/:userId/posts/:postId/comments/:commentId", handler)

	testCases := []struct {
		path       string
		shouldFind bool
		paramCount int
		description string
	}{
		{"/", true, 0, "Root path should match"},
		{"/value", true, 1, "Single parameter path should match"},
		{"/users/123/edit", true, 2, "Multi-parameter path should match"},
		{"/very/long/path/with/many/segments", true, 0, "Long path should match"},
		{"/api/v1/2.0/users/123/posts/456/comments/789", true, 4, "Complex multi-parameter path should match"},
		{"/api/v1/2.0/users/123/posts/456", false, 0, "Incomplete complex path should not match"},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			params := make(map[string]string)
			_, found := tree.Find(http.MethodGet, tc.path, params)
			assert.Equal(t, tc.shouldFind, found, tc.description)

			if tc.shouldFind && tc.paramCount > 0 {
				assert.Equal(t, tc.paramCount, len(params), "Should extract correct number of parameters")
			}
		})
	}
}
package router

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegexRoute(t *testing.T) {
	tree := NewRadixTree()
	handler := func() {}

	// 测试基本的正则路由
	tree.Add(http.MethodGet, "/users/:id([0-9]+)", handler)
	tree.Add(http.MethodGet, "/posts/:slug([a-z0-9-]+)", handler)
	tree.Add(http.MethodGet, "/files/:filename([^/]+\\.pdf)", handler)

	// 验证路由数量
	assert.Equal(t, 3, tree.Routes(), "Total routes should be 3")

	t.Run("valid regex path", func(t *testing.T) {
		testCases := []struct {
			path   string
			params map[string]string
		}{
			{
				"/users/123",
				map[string]string{"id": "123"},
			},
			{
				"/posts/my-first-post",
				map[string]string{"slug": "my-first-post"},
			},
			{
				"/files/document.pdf",
				map[string]string{"filename": "document.pdf"},
			},
		}

		for _, tc := range testCases {
			params := make(map[string]string)
			handler, found := tree.Find(http.MethodGet, tc.path, params)

			assert.True(t, found, "Route should be found: %s", tc.path)
			assert.NotNil(t, handler, "Handler should not be nil for path: %s", tc.path)

			for key, expectedValue := range tc.params {
				value, exists := params[key]
				assert.True(t, exists, "Parameter %s should exist", key)
				assert.Equal(t, expectedValue, value, "Parameter %s value should be %s", key, expectedValue)
			}
		}
	})

	t.Run("invalid regex path", func(t *testing.T) {
		testCases := []struct {
			path string
		}{
			{"/users/abc"},           // 不是数字
			{"/posts/INVALID_SLUG"},  // 包含大写字母
			{"/files/document.txt"},  // 不是pdf文件
		}

		for _, tc := range testCases {
			params := make(map[string]string)
			_, found := tree.Find(http.MethodGet, tc.path, params)
			assert.False(t, found, "Route should not be found: %s", tc.path)
		}
	})

	t.Run("正则表达式语法错误", func(t *testing.T) {
		assert.Panics(t, func() {
			tree.Add(http.MethodGet, "/invalid/:id([abc)", handler)
		}, "Should panic with invalid regex")
	})
}

func TestComplexRegexPatterns(t *testing.T) {
	tree := NewRadixTree()
	handler := func() {}

	// 添加一些复杂的正则表达式路由
	tree.Add(http.MethodGet, "/api/:version(v[0-9]+)/users", handler)
	tree.Add(http.MethodGet, "/date/:year([0-9]{4})/:month([0-9]{2})/:day([0-9]{2})", handler)
	tree.Add(http.MethodGet, "/products/:category([a-z]+)/:id([0-9]{3,10})", handler)
	tree.Add(http.MethodGet, "/download/:filename([^/]+\\.(zip|tar\\.gz|rar))", handler)

	t.Run("complex regex path", func(t *testing.T) {
		testCases := []struct {
			path   string
			params map[string]string
		}{
			{
				"/api/v1/users",
				map[string]string{"version": "v1"},
			},
			{
				"/date/2023/05/23",
				map[string]string{"year": "2023", "month": "05", "day": "23"},
			},
			{
				"/products/electronics/12345",
				map[string]string{"category": "electronics", "id": "12345"},
			},
			{
				"/download/archive.zip",
				map[string]string{"filename": "archive.zip"},
			},
			{
				"/download/data.tar.gz",
				map[string]string{"filename": "data.tar.gz"},
			},
		}

		for _, tc := range testCases {
			params := make(map[string]string)
			handler, found := tree.Find(http.MethodGet, tc.path, params)

			assert.True(t, found, "Route should be found: %s", tc.path)
			assert.NotNil(t, handler, "Handler should not be nil for path: %s", tc.path)

			for key, expectedValue := range tc.params {
				value, exists := params[key]
				assert.True(t, exists, "Parameter %s should exist", key)
				assert.Equal(t, expectedValue, value, "Parameter %s value should be %s", key, expectedValue)
			}
		}
	})

	t.Run("invalid complex regex path", func(t *testing.T) {
		invalidPaths := []string{
			"/api/version1/users",     // 不符合v数字格式
			"/date/23/05/23",          // 年份需要4位
			"/date/2023/5/23",         // 月份需要2位
			"/products/123/456",       // 类别应该是字母
			"/download/archive.exe",   // 不支持的扩展名
		}

		for _, path := range invalidPaths {
			params := make(map[string]string)
			_, found := tree.Find(http.MethodGet, path, params)
			assert.False(t, found, "Route should not be found: %s", path)
		}
	})
}

func TestNestedRegexParameters(t *testing.T) {
	tree := NewRadixTree()
	handler := func() {}

	// 嵌套的带正则表达式的路由
	tree.Add(http.MethodGet, "/api/:version(v[0-9]+)/users/:id([0-9]+)", handler)
	tree.Add(http.MethodGet, "/blog/:year([0-9]{4})/:month([0-9]{2})/:slug([a-z0-9-]+)", handler)

	t.Run("match regex param", func(t *testing.T) {
		params := make(map[string]string)
		handler, found := tree.Find(http.MethodGet, "/api/v1/users/123", params)

		assert.True(t, found, "Nested regex route should be found")
		assert.NotNil(t, handler, "Handler should not be nil")
		assert.Equal(t, "v1", params["version"], "Version parameter should match")
		assert.Equal(t, "123", params["id"], "ID parameter should match")

		// 重置参数映射
		params = make(map[string]string)
		handler, found = tree.Find(http.MethodGet, "/blog/2023/05/my-first-post", params)

		assert.True(t, found, "Nested regex blog route should be found")
		assert.NotNil(t, handler, "Handler should not be nil")
		assert.Equal(t, "2023", params["year"], "Year parameter should match")
		assert.Equal(t, "05", params["month"], "Month parameter should match")
		assert.Equal(t, "my-first-post", params["slug"], "Slug parameter should match")
	})

	t.Run("invalid regex param", func(t *testing.T) {
		invalidPaths := []string{
			"/api/ver1/users/123",        // 版本格式不正确
			"/api/v1/users/abc",          // ID不是数字
			"/blog/23/05/my-post",        // 年份需要4位
			"/blog/2023/5/my-post",       // 月份需要2位
			"/blog/2023/05/MY-POST",      // slug不能有大写字母
		}

		for _, path := range invalidPaths {
			params := make(map[string]string)
			_, found := tree.Find(http.MethodGet, path, params)
			assert.False(t, found, "Route should not be found: %s", path)
		}
	})
}

func TestRegexCacheUsage(t *testing.T) {
	// 创建一个自定义的正则缓存进行测试
	cache := NewRegexCache()

	// 测试缓存的基本功能
	pattern := "[0-9]+"
	regex1, err1 := cache.Get(pattern)
	require.NoError(t, err1, "First regex compilation should succeed")

	regex2, err2 := cache.Get(pattern)
	require.NoError(t, err2, "Second regex compilation should succeed")

	// 检查两次获取的是同一个正则对象
	assert.Same(t, regex1, regex2, "Cached regex objects should be identical")

	// 测试缓存大小
	assert.Equal(t, 1, cache.Size(), "Cache should contain 1 item")

	// 测试错误处理
	_, err := cache.Get("(invalid")
	assert.Error(t, err, "Invalid regex should return error")

	// 测试MustGet
	assert.Panics(t, func() {
		cache.MustGet("(invalid")
	}, "MustGet should panic with invalid regex")

	// 测试缓存清理
	cache.Clear()
	assert.Equal(t, 0, cache.Size(), "Cache should be empty after clearing")
}

func TestRegexVsParamPriority(t *testing.T) {
	tree := NewRadixTree()

	// 添加正则和普通参数路由
	regexHandler := func() {}
	paramHandler := func() {}

	tree.Add(http.MethodGet, "/users/:id([0-9]+)/profile", regexHandler)
	tree.Add(http.MethodGet, "/users/:name/settings", paramHandler)

	t.Run("regex match priority", func(t *testing.T) {
		// 应该匹配第一个路由，因为id符合正则表达式
		params := make(map[string]string)
		handler, found := tree.Find(http.MethodGet, "/users/123/profile", params)

		assert.True(t, found, "Regex route should be found")
		assert.Equal(t, regexHandler, handler, "Should match regex handler")
		assert.Equal(t, "123", params["id"], "ID parameter should be extracted")
	})

	t.Run("param route", func(t *testing.T) {
		// 匹配参数路由
		params := make(map[string]string)
		handler, found := tree.Find(http.MethodGet, "/users/john/settings", params)

		assert.True(t, found, "Parameter route should be found")
		assert.Equal(t, paramHandler, handler, "Should match parameter handler")
		assert.Equal(t, "john", params["name"], "Name parameter should be extracted")
	})
}

func TestEdgeCasesForRegex(t *testing.T) {
	tree := NewRadixTree()
	handler := func() {}

	// 测试边缘情况
	tree.Add(http.MethodGet, "/empty/:param()", handler) // 空的正则表达式
	tree.Add(http.MethodGet, "/optional/:param([0-9]*)", handler) // 可选的数字
	tree.Add(http.MethodGet, "/special/:param([\\w\\-\\.]+)", handler) // 特殊字符

	t.Run("edge cases", func(t *testing.T) {
		testCases := []struct {
			path       string
			shouldFind bool
			paramValue string
		}{
			{"/empty/", true, ""}, // 空参数
			{"/optional/", true, ""}, // 可选数字，空值
			{"/optional/123", true, "123"}, // 可选数字
			{"/special/file-name.txt", true, "file-name.txt"}, // 特殊字符
			{"/special/user_name-123.jpg", true, "user_name-123.jpg"}, // 更多特殊字符
		}

		for _, tc := range testCases {
			params := make(map[string]string)
			_, found := tree.Find(http.MethodGet, tc.path, params)

			if tc.shouldFind {
				assert.True(t, found, "Route should be found: %s", tc.path)
				if tc.paramValue != "" {
					assert.Equal(t, tc.paramValue, params["param"], "Parameter value should match for %s", tc.path)
				}
			} else {
				assert.False(t, found, "Route should not be found: %s", tc.path)
			}
		}
	})
}

func TestOverwritingRegexRoutes(t *testing.T) {
	tree := NewRadixTree()

	// 首先添加一个路由
	handler1 := func() {}
	tree.Add(http.MethodGet, "/api/:version(v[0-9]+)", handler1)

	// 然后用另一个处理器覆盖它
	handler2 := func() { }
	tree.Add(http.MethodGet, "/api/:version(v[0-9]+)", handler2)

	// 验证被覆盖
	params := make(map[string]string)
	handler, found := tree.Find(http.MethodGet, "/api/v1", params)

	assert.True(t, found, "Route should be found")
	assert.Equal(t, handler2, handler, "Handler should be the overwritten one")
	assert.Equal(t, "v1", params["version"], "Version parameter should be extracted")

	// 验证总路由数仍为1
	assert.Equal(t, 1, tree.Routes(), "There should still be only 1 route")
}
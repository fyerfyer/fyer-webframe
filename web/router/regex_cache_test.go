package router

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegexCache(t *testing.T) {
	t.Run("basic caching", func(t *testing.T) {
		cache := NewRegexCache()

		// 第一次获取编译并缓存
		pattern1, err := cache.Get("[0-9]+")
		require.NoError(t, err, "Should compile valid regex")
		assert.Equal(t, 1, cache.Size(), "Cache should have 1 item")

		// 第二次获取应从缓存中获取
		pattern2, err := cache.Get("[0-9]+")
		require.NoError(t, err, "Should retrieve from cache")
		assert.Same(t, pattern1, pattern2, "Should return same compiled instance")

		// 获取另一个不同的正则
		pattern3, err := cache.Get("[a-z]+")
		require.NoError(t, err, "Should compile second regex")
		assert.Equal(t, 2, cache.Size(), "Cache should have 2 items")
		assert.NotSame(t, pattern1, pattern3, "Different patterns should be different instances")
	})

	t.Run("error handling", func(t *testing.T) {
		cache := NewRegexCache()

		// 测试无效正则表达式
		_, err := cache.Get("[unclosed")
		assert.Error(t, err, "Should return error for invalid regex")

		// 测试MustGet的panic行为
		assert.Panics(t, func() {
			cache.MustGet("[unclosed")
		}, "MustGet should panic on invalid regex")
	})

	t.Run("cache operations", func(t *testing.T) {
		cache := NewRegexCache()

		// 填充缓存
		cache.MustGet("[0-9]+")
		cache.MustGet("[a-z]+")
		assert.Equal(t, 2, cache.Size(), "Cache should have 2 items before clear")

		// 清空缓存
		cache.Clear()
		assert.Equal(t, 0, cache.Size(), "Cache should be empty after clear")
	})
}

func TestRegexRouting(t *testing.T) {
	t.Run("numeric parameters", func(t *testing.T) {
		tree := NewRadixTree()
		handler := func() {}

		// 注册带数字ID参数的路由
		tree.Add(http.MethodGet, "/users/:id([0-9]+)", handler)

		// 有效匹配
		params := make(map[string]string)
		_, found := tree.Find(http.MethodGet, "/users/123", params)
		assert.True(t, found, "Should find route with numeric ID")
		assert.Equal(t, "123", params["id"], "Should extract correct ID parameter")

		// 无效匹配
		params = make(map[string]string)
		_, found = tree.Find(http.MethodGet, "/users/abc", params)
		assert.False(t, found, "Should not find route with non-numeric ID")
	})

	t.Run("alphanumeric parameters", func(t *testing.T) {
		tree := NewRadixTree()
		handler := func() {}

		// 注册带字母数字参数的路由
		tree.Add(http.MethodGet, "/products/:code([a-z0-9]+)", handler)

		// 有效匹配
		params := make(map[string]string)
		_, found := tree.Find(http.MethodGet, "/products/product123", params)
		assert.True(t, found, "Should find route with alphanumeric code")
		assert.Equal(t, "product123", params["code"], "Should extract correct code parameter")

		// 无效匹配
		params = make(map[string]string)
		_, found = tree.Find(http.MethodGet, "/products/PRODUCT123", params)
		assert.False(t, found, "Should not find route with uppercase letters")
	})

	t.Run("date format parameters", func(t *testing.T) {
		tree := NewRadixTree()
		handler := func() {}

		// 注册带日期格式参数的路由
		tree.Add(http.MethodGet, "/events/:date([0-9]{4}-[0-9]{2}-[0-9]{2})", handler)

		// 有效匹配
		params := make(map[string]string)
		_, found := tree.Find(http.MethodGet, "/events/2023-12-31", params)
		assert.True(t, found, "Should find route with valid date format")
		assert.Equal(t, "2023-12-31", params["date"], "Should extract correct date parameter")

		// 无效匹配
		params = make(map[string]string)
		_, found = tree.Find(http.MethodGet, "/events/2023/12/31", params)
		assert.False(t, found, "Should not find route with invalid date format")
	})

	t.Run("slugs and file extensions", func(t *testing.T) {
		tree := NewRadixTree()
		handler := func() {}

		tree.Add(http.MethodGet, "/posts/:slug([a-z0-9-]+)", handler)

		// 测试slug
		params := make(map[string]string)
		_, found := tree.Find(http.MethodGet, "/posts/my-awesome-post-123", params)
		assert.True(t, found, "Should find route with valid slug")
		assert.Equal(t, "my-awesome-post-123", params["slug"], "Should extract correct slug parameter")
	})

	t.Run("route priority", func(t *testing.T) {
		tree := NewRadixTree()

		// 定义不同的处理器
		type Handler struct { name string }
		staticHandler := &Handler{name: "static"}
		regexHandler := &Handler{name: "regex"}
		paramHandler := &Handler{name: "param"}
		wildcardHandler := &Handler{name: "wildcard"}

		// 注册不同类型的路由
		tree.Add(http.MethodGet, "/api/users/list", staticHandler)
		tree.Add(http.MethodGet, "/api/users/:id([0-9]+)", regexHandler)
		tree.Add(http.MethodGet, "/api/users/:name", paramHandler)
		tree.Add(http.MethodGet, "/api/users/*", wildcardHandler)

		// 测试优先级匹配
		params := make(map[string]string)

		// 静态路由应该优先
		handler, found := tree.Find(http.MethodGet, "/api/users/list", params)
		assert.True(t, found, "Static route should be found")
		assert.Equal(t, staticHandler, handler, "Static route should have highest priority")

		// 正则路由其次
		params = make(map[string]string)
		handler, found = tree.Find(http.MethodGet, "/api/users/123", params)
		assert.True(t, found, "Regex route should be found")
		assert.Equal(t, regexHandler, handler, "Regex route should have second highest priority")
		assert.Equal(t, "123", params["id"], "Regex parameter should be extracted")

		// 普通参数路由再次
		params = make(map[string]string)
		handler, found = tree.Find(http.MethodGet, "/api/users/john", params)
		assert.True(t, found, "Parameter route should be found")
		assert.Equal(t, paramHandler, handler, "Parameter route should have third highest priority")
		assert.Equal(t, "john", params["name"], "Parameter should be extracted")

		// 通配符路由最后
		params = make(map[string]string)
		handler, found = tree.Find(http.MethodGet, "/api/users/something/else", params)
		assert.True(t, found, "Wildcard route should be found")
		assert.Equal(t, wildcardHandler, handler, "Wildcard route should have lowest priority")
		assert.Equal(t, "something/else", params["*"], "Wildcard parameter should be extracted")
	})

	t.Run("multiple regex parameters", func(t *testing.T) {
		tree := NewRadixTree()
		handler := func() {}

		// 注册带多个正则参数的路由
		tree.Add(http.MethodGet, "/api/:version(v[0-9])/users/:id([0-9]+)", handler)

		// 测试有效匹配
		params := make(map[string]string)
		_, found := tree.Find(http.MethodGet, "/api/v1/users/123", params)
		assert.True(t, found, "Should find route with multiple regex parameters")
		assert.Equal(t, "v1", params["version"], "Should extract correct version parameter")
		assert.Equal(t, "123", params["id"], "Should extract correct id parameter")

		// 测试部分无效匹配
		params = make(map[string]string)
		_, found = tree.Find(http.MethodGet, "/api/v12/users/123", params)
		assert.False(t, found, "Should not find route when first parameter doesn't match")

		params = make(map[string]string)
		_, found = tree.Find(http.MethodGet, "/api/v1/users/abc", params)
		assert.False(t, found, "Should not find route when second parameter doesn't match")
	})

	t.Run("duplicate regex routes", func(t *testing.T) {
		tree := NewRadixTree()
		handler := func() {}

		// 添加一个正则路由
		tree.Add(http.MethodGet, "/users/:id([0-9]+)", handler)

		// 再次添加同样的路由应该会panic
		assert.Panics(t, func() {
			tree.Add(http.MethodGet, "/users/:id([0-9]+)", handler)
		}, "Adding duplicate regex route should panic")

		// 添加同名参数但不同正则也应该会panic
		assert.Panics(t, func() {
			tree.Add(http.MethodGet, "/users/:id([a-z]+)", handler)
		}, "Adding different regex with same parameter name should panic")
	})

	t.Run("invalid regex", func(t *testing.T) {
		tree := NewRadixTree()
		handler := func() {}

		// 添加无效正则表达式应该会panic
		assert.Panics(t, func() {
			tree.Add(http.MethodGet, "/users/:id([0-9", handler)
		}, "Adding invalid regex should panic")
	})
}

// 测试正则表达式加上静态路径的组合
func TestRegexWithStaticSuffix(t *testing.T) {
	tree := NewRadixTree()
	handler := func() {}

	// 注册带正则参数后接静态路径的路由
	tree.Add(http.MethodGet, "/users/:id([0-9]+)/profile", handler)

	// 测试有效匹配
	params := make(map[string]string)
	_, found := tree.Find(http.MethodGet, "/users/123/profile", params)
	assert.True(t, found, "Should find route with regex parameter and static suffix")
	assert.Equal(t, "123", params["id"], "Should extract correct id parameter")

	// 测试无效匹配 - 参数不匹配
	params = make(map[string]string)
	_, found = tree.Find(http.MethodGet, "/users/abc/profile", params)
	assert.False(t, found, "Should not find route when regex parameter doesn't match")

	// 测试无效匹配 - 后缀不匹配
	params = make(map[string]string)
	_, found = tree.Find(http.MethodGet, "/users/123/settings", params)
	assert.False(t, found, "Should not find route when static suffix doesn't match")
}

// 测试常用的HTTP API路由模式
func TestCommonAPIPatterns(t *testing.T) {
	tree := NewRadixTree()
	handler := func() {}

	// 注册一些常见的API路由模式
	tree.Add(http.MethodGet, "/api/v1/users/:id([0-9]+)", handler)
	tree.Add(http.MethodGet, "/api/v1/posts/:slug([a-z0-9-]+)", handler)
	tree.Add(http.MethodGet, "/api/v1/products/:sku([A-Z0-9]{6})", handler)

	// 测试用户ID路由
	params := make(map[string]string)
	_, found := tree.Find(http.MethodGet, "/api/v1/users/42", params)
	assert.True(t, found, "Should find user route")
	assert.Equal(t, "42", params["id"], "Should extract correct user id")

	// 测试文章slug路由
	params = make(map[string]string)
	_, found = tree.Find(http.MethodGet, "/api/v1/posts/my-first-post", params)
	assert.True(t, found, "Should find post route")
	assert.Equal(t, "my-first-post", params["slug"], "Should extract correct post slug")

	// 测试产品SKU路由
	params = make(map[string]string)
	_, found = tree.Find(http.MethodGet, "/api/v1/products/ABC123", params)
	assert.True(t, found, "Should find product route")
	assert.Equal(t, "ABC123", params["sku"], "Should extract correct product SKU")
}
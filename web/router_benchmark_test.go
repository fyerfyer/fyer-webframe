package web

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func BenchmarkRouter_StaticRoutes(b *testing.B) {
	r := NewRouter()
	handler := func(ctx *Context) {}

	// 注册静态路由
	r.Get("/", handler)
	r.Get("/user", handler)
	r.Get("/user/profile", handler)
	r.Get("/admin", handler)
	r.Get("/admin/settings", handler)
	r.Get("/products", handler)
	r.Get("/products/list", handler)
	r.Get("/api/v1/users", handler)
	r.Get("/api/v1/products", handler)

	// 创建测试上下文
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	ctx := &Context{Req: req, Param: make(map[string]string)}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.findHandler(http.MethodGet, "/api/v1/users", ctx)
	}
}

func BenchmarkRouter_ParamRoutes(b *testing.B) {
	r := NewRouter()
	handler := func(ctx *Context) {}

	// 注册参数路由
	r.Get("/user/:id", handler)
	r.Get("/user/:id/profile", handler)
	r.Get("/products/:category", handler)
	r.Get("/products/:category/:id", handler)
	r.Get("/api/v1/users/:id", handler)

	// 创建测试上下文
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/123", nil)
	ctx := &Context{Req: req, Param: make(map[string]string)}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.findHandler(http.MethodGet, "/api/v1/users/123", ctx)
	}
}

func BenchmarkRouter_WildcardRoutes(b *testing.B) {
	r := NewRouter()
	handler := func(ctx *Context) {}

	// 注册通配符路由
	r.Get("/static/*", handler)
	r.Get("/images/*", handler)
	r.Get("/api/v1/files/*", handler)
	r.Get("/download/*", handler)

	// 创建测试上下文
	req := httptest.NewRequest(http.MethodGet, "/api/v1/files/images/avatar.jpg", nil)
	ctx := &Context{Req: req, Param: make(map[string]string)}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.findHandler(http.MethodGet, "/api/v1/files/images/avatar.jpg", ctx)
	}
}

func BenchmarkRouter_RegexRoutes(b *testing.B) {
	r := NewRouter()
	handler := func(ctx *Context) {}

	// 注册正则路由
	r.Get("/user/:id([0-9]+)", handler)
	r.Get("/products/:category([a-z]+)", handler)
	r.Get("/api/v1/users/:id([0-9]+)", handler)
	r.Get("/api/v1/products/:id([a-f0-9]+)", handler)

	// 创建测试上下文
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/123", nil)
	ctx := &Context{Req: req, Param: make(map[string]string)}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.findHandler(http.MethodGet, "/api/v1/users/123", ctx)
	}
}

func BenchmarkRouter_MixedRoutes(b *testing.B) {
	r := NewRouter()
	handler := func(ctx *Context) {}

	// 注册混合类型的路由
	r.Get("/", handler)
	r.Get("/user", handler)
	r.Get("/user/:id", handler)
	r.Get("/user/:id/profile", handler)
	r.Get("/admin", handler)
	r.Get("/admin/settings", handler)
	r.Get("/products", handler)
	r.Get("/products/:category", handler)
	r.Get("/products/:category/:id([0-9]+)", handler)
	r.Get("/api/v1/users", handler)
	r.Get("/api/v1/users/:id", handler)
	r.Get("/api/v1/files/*", handler)

	// 创建测试上下文
	req := httptest.NewRequest(http.MethodGet, "/products/electronics/123", nil)
	ctx := &Context{Req: req, Param: make(map[string]string)}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.findHandler(http.MethodGet, "/products/electronics/123", ctx)
	}
}

func BenchmarkRouter_MultipleMethods(b *testing.B) {
	r := NewRouter()
	handler := func(ctx *Context) {}

	// 注册多种HTTP方法的路由
	r.Get("/api/users", handler)
	r.Post("/api/users", handler)
	r.Put("/api/users/:id", handler)
	r.Delete("/api/users/:id", handler)
	r.Get("/api/products", handler)
	r.Post("/api/products", handler)
	r.Put("/api/products/:id", handler)
	r.Delete("/api/products/:id", handler)

	// 创建测试上下文
	req := httptest.NewRequest(http.MethodPut, "/api/users/123", nil)
	ctx := &Context{Req: req, Param: make(map[string]string)}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.findHandler(http.MethodPut, "/api/users/123", ctx)
	}
}

func BenchmarkRouter_LongPath(b *testing.B) {
	r := NewRouter()
	handler := func(ctx *Context) {}

	// 注册长路径
	r.Get("/api/v1/organizations/:orgId/departments/:deptId/employees/:empId/profile", handler)

	// 创建测试上下文
	req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/123/departments/456/employees/789/profile", nil)
	ctx := &Context{Req: req, Param: make(map[string]string)}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.findHandler(http.MethodGet, "/api/v1/organizations/123/departments/456/employees/789/profile", ctx)
	}
}

func BenchmarkRouter_Middleware(b *testing.B) {
	r := NewRouter()
	handler := func(ctx *Context) {}
	middleware := func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			next(ctx)
		}
	}

	// 注册中间件和路由
	r.Get("/api/v1/users", handler)
	r.Use("GET", "/api/v1/*", middleware)
	r.Use("GET", "/api/*", middleware)
	r.Use("GET", "/*", middleware)

	// 创建测试上下文
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	ctx := &Context{Req: req, Param: make(map[string]string)}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		n, ok := r.findHandler(http.MethodGet, "/api/v1/users", ctx)
		if ok && n != nil {
			middlewares := collectMatchingMiddlewares(r.middlewares["GET"], "/api/v1/users")
			BuildChain(n.handler, "/api/v1/users", middlewares)(ctx)
		}
	}
}

func BenchmarkRouter_LargeNumberOfRoutes(b *testing.B) {
	r := NewRouter()
	handler := func(ctx *Context) {}

	// 注册大量路由
	for i := 0; i < 500; i++ {
		r.Get("/api/route"+strconv.Itoa(i), handler)
		r.Get("/api/users/"+strconv.Itoa(i), handler)
		r.Get("/api/products/"+strconv.Itoa(i), handler)
	}

	// 创建测试上下文
	req := httptest.NewRequest(http.MethodGet, "/api/products/499", nil)
	ctx := &Context{Req: req, Param: make(map[string]string)}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.findHandler(http.MethodGet, "/api/products/499", ctx)
	}
}

func BenchmarkRouter_WorstCaseScenario(b *testing.B) {
	r := NewRouter()
	handler := func(ctx *Context) {}

	// 注册多种路由和中间件
	for i := 0; i < 50; i++ {
		path := "/api/v1/users/" + strconv.Itoa(i)
		r.Get(path, handler)
		r.Use("GET", path, func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				next(ctx)
			}
		})
	}

	r.Get("/api/v1/users/:id([0-9]+)/profile", handler)

	// 创建测试上下文
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/9999/profile", nil)
	ctx := &Context{Req: req, Param: make(map[string]string)}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		n, ok := r.findHandler(http.MethodGet, "/api/v1/users/9999/profile", ctx)
		if ok && n != nil {
			middlewares := collectMatchingMiddlewares(r.middlewares["GET"], "/api/v1/users/9999/profile")
			BuildChain(n.handler, "/api/v1/users/9999/profile", middlewares)(ctx)
		}
	}
}
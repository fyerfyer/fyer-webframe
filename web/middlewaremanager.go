package web

// MiddlewareManager 中间件管理器接口
type MiddlewareManager interface {
	// 全局中间件
	Global() MiddlewareRegister

	// 针对特定路径的中间件
	For(method string, path string) MiddlewareRegister

	// 条件中间件
	When(condition func(c *Context) bool) MiddlewareRegister
}

// MiddlewareRegister 中间件注册器
type MiddlewareRegister interface {
	Add(middleware ...Middleware) MiddlewareRegister
}

// middlewareManager 实现中间件管理器接口
type middlewareManager struct {
	server *HTTPServer
}

// newMiddlewareManager 创建一个新的中间件管理器
func newMiddlewareManager(server *HTTPServer) *middlewareManager {
	return &middlewareManager{
		server: server,
	}
}

// Global 注册全局中间件
func (m *middlewareManager) Global() MiddlewareRegister {
	return &middlewareRegister{
		server: m.server,
		method: "",
		path:   "/*",
	}
}

// For 注册针对特定路径的中间件
func (m *middlewareManager) For(method string, path string) MiddlewareRegister {
	// 如果没有提供方法，则应用到所有方法
	if method == "" {
		return &middlewareRegister{
			server:    m.server,
			method:    "",
			path:      path,
			allMethod: true,
		}
	}
	return &middlewareRegister{
		server: m.server,
		method: method,
		path:   path,
	}
}

// When 注册条件中间件
func (m *middlewareManager) When(condition func(c *Context) bool) MiddlewareRegister {
	return &conditionalRegister{
		server:    m.server,
		condition: condition,
	}
}

// middlewareRegister 实现中间件注册接口
type middlewareRegister struct {
	server    *HTTPServer
	method    string
	path      string
	allMethod bool
}

// Add 添加中间件
func (r *middlewareRegister) Add(middleware ...Middleware) MiddlewareRegister {
	// Handle all HTTP methods case
	if r.allMethod {
		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}
		for _, method := range methods {
			for _, mw := range middleware {
				r.server.Use(method, r.path, mw)
			}
		}
		return r
	}

	// Handle single HTTP method case
	for _, mw := range middleware {
		r.server.Use(r.method, r.path, mw)
	}
	return r
}


// conditionalRegister 实现条件中间件注册器
type conditionalRegister struct {
	server    *HTTPServer
	condition func(c *Context) bool
}

// Add 添加条件中间件
func (r *conditionalRegister) Add(middlewares ...Middleware) MiddlewareRegister {
	for _, mw := range middlewares {
		// Create a wrapped middleware that only executes when condition is true
		wrapped := func(next HandlerFunc) HandlerFunc {
			return func(ctx *Context) {
				if r.condition(ctx) {
					mw(next)(ctx)
				} else {
					next(ctx)
				}
			}
		}

		// Mark this as a conditional middleware internally by wrapping it
		conditionalMw := func(next HandlerFunc) HandlerFunc {
			handler := wrapped(next)
			// Store the source information in the context or elsewhere if needed
			return func(ctx *Context) {
				handler(ctx)
			}
		}

		// Apply to all HTTP methods with global wildcard path
		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}
		for _, method := range methods {
			r.server.Use(method, "/*", conditionalMw)
		}
	}
	return r
}

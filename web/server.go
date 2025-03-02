package web

import (
	"context"
	"net"
	"net/http"
	"time"
)

// Server 接口定义
type Server interface {
	http.Handler
	// Start 启动服务器
	Start(addr string) error
	// Shutdown 优雅关闭
	Shutdown(ctx context.Context) error

	// Get ...路由注册简化方法
	Get(path string, handler HandlerFunc) RouteRegister
	Post(path string, handler HandlerFunc) RouteRegister
	Put(path string, handler HandlerFunc) RouteRegister
	Delete(path string, handler HandlerFunc) RouteRegister
	Patch(path string, handler HandlerFunc) RouteRegister
	Options(path string, handler HandlerFunc) RouteRegister

	// 路由组和中间件
	Group(prefix string) RouteGroup
	Middleware() MiddlewareManager

	// 模板引擎
	UseTemplate(tpl Template) Server
}

// RouteRegister 路由链式注册接口
type RouteRegister interface {
	// Middleware 为特定路由添加中间件
	Middleware(middleware ...Middleware) RouteRegister
}

// HTTPServer 结构体
type HTTPServer struct {
	*Router   // 继承Router
	start     bool
	noRouter  HandlerFunc  // 404处理器
	server    *http.Server // 底层的http server
	baseRoute string       // 基础路由前缀
	tplEngine Template     // 模板引擎
}

// ServerOption 定义服务器选项
type ServerOption func(*HTTPServer)

// WithReadTimeout 设置读取超时
func WithReadTimeout(timeout time.Duration) ServerOption {
	return func(server *HTTPServer) {
		server.server.ReadTimeout = timeout
	}
}

// WithWriteTimeout 设置写入超时
func WithWriteTimeout(timeout time.Duration) ServerOption {
	return func(server *HTTPServer) {
		server.server.WriteTimeout = timeout
	}
}

// WithTemplate 设置模板引擎
func WithTemplate(tpl Template) ServerOption {
	return func(server *HTTPServer) {
		server.tplEngine = tpl
	}
}

// WithNotFoundHandler 设置404处理器
func WithNotFoundHandler(handler HandlerFunc) ServerOption {
	return func(server *HTTPServer) {
		server.noRouter = handler
	}
}

// WithBasePath 设置基础路径前缀
func WithBasePath(basePath string) ServerOption {
	return func(server *HTTPServer) {
		server.baseRoute = basePath
	}
}

// NewHTTPServer 创建HTTP服务器实例
func NewHTTPServer(opts ...ServerOption) *HTTPServer {
	server := &HTTPServer{
		Router: NewRouter(),
		server: &http.Server{},
		noRouter: func(ctx *Context) {
			ctx.Resp.WriteHeader(http.StatusNotFound)
			ctx.Resp.Write([]byte("404 Not Found"))
		},
	}

	// 应用所有选项
	for _, opt := range opts {
		opt(server)
	}

	// 设置 http.Server 的处理器为当前实例
	server.server.Handler = server
	return server
}

// ServeHTTP HTTPServer的核心处理函数
func (s *HTTPServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	ctx := &Context{
		Req:        req,
		Resp:       res,
		Param:      make(map[string]string),
		tplEngine:  s.tplEngine,
		Context:    req.Context(),
		unhandled:  true,
		UserValues: make(map[string]any),
	}

	// 如果设置了基础路径，需要处理路径前缀
	originalPath := req.URL.Path
	path := originalPath

	// Check if the request path starts with the base route
	if s.baseRoute != "" {
		if len(path) >= len(s.baseRoute) && path[:len(s.baseRoute)] == s.baseRoute {
			// Strip base path for routing
			path = path[len(s.baseRoute):]
			if path == "" {
				path = "/"
			}
		} else {
			// If base route is set but the path doesn't start with it, return 404
			s.noRouter(ctx)
			s.handleResponse(ctx)
			return
		}
	}

	// 查找路由
	node, ok := s.findHandler(req.Method, path, ctx)
	if !ok {
		s.noRouter(ctx)
		s.handleResponse(ctx)
		return
	}

	// 构建并执行处理链
	handler := BuildChain(node.handler, path, s.Router.middlewares[req.Method])
	handler(ctx)

	// 处理响应
	s.handleResponse(ctx)
}

// handleResponse 统一处理响应
func (s *HTTPServer) handleResponse(ctx *Context) {
	// 如果已经直接操作了ResponseWriter，就不再进行处理
	if !ctx.unhandled {
		return
	}

	// 设置默认的状态码，如果没有设置
	if ctx.RespStatusCode <= 0 {
		ctx.RespStatusCode = http.StatusOK
	}

	// 设置状态码
	ctx.Resp.WriteHeader(ctx.RespStatusCode)

	// 写入响应数据（如果有）
	if len(ctx.RespData) > 0 {
		_, err := ctx.Resp.Write(ctx.RespData)
		if err != nil {
			// 尝试写入一个错误响应（如果我们还没有开始写入）
			if ctx.RespStatusCode < 400 {
				http.Error(ctx.Resp, "Internal Server Error", http.StatusInternalServerError)
			}
		}
	}
}

// Start 启动服务器
func (s *HTTPServer) Start(addr string) error {
	listen, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s.start = true
	s.server.Addr = addr
	return s.server.Serve(listen)
}

// Shutdown 优雅关闭
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	s.start = false
	return s.server.Shutdown(ctx)
}

// Get 注册GET路由
func (s *HTTPServer) Get(path string, handler HandlerFunc) RouteRegister {
	s.Router.Get(path, handler)
	return newRouteRegister(s, "GET", path)
}

// Post 注册POST路由
func (s *HTTPServer) Post(path string, handler HandlerFunc) RouteRegister {
	s.Router.Post(path, handler)
	return newRouteRegister(s, "POST", path)
}

// Put 注册PUT路由
func (s *HTTPServer) Put(path string, handler HandlerFunc) RouteRegister {
	s.Router.Put(path, handler)
	return newRouteRegister(s, "PUT", path)
}

// Delete 注册DELETE路由
func (s *HTTPServer) Delete(path string, handler HandlerFunc) RouteRegister {
	s.Router.Delete(path, handler)
	return newRouteRegister(s, "DELETE", path)
}

// Patch 注册PATCH路由
func (s *HTTPServer) Patch(path string, handler HandlerFunc) RouteRegister {
	s.Router.Patch(path, handler)
	return newRouteRegister(s, "PATCH", path)
}

// Options 注册OPTIONS路由
func (s *HTTPServer) Options(path string, handler HandlerFunc) RouteRegister {
	s.Router.Options(path, handler)
	return newRouteRegister(s, "OPTIONS", path)
}

// Group 创建路由组
func (s *HTTPServer) Group(prefix string) RouteGroup {
	return newRouteGroup(s, prefix)
}

// Middleware 返回中间件管理器
func (s *HTTPServer) Middleware() MiddlewareManager {
	return newMiddlewareManager(s)
}

// UseTemplate 设置模板引擎
func (s *HTTPServer) UseTemplate(tpl Template) Server {
	s.tplEngine = tpl
	return s
}

// routeRegister 实现RouteRegister接口
type routeRegister struct {
	server *HTTPServer
	method string
	path   string
}

func newRouteRegister(server *HTTPServer, method, path string) *routeRegister {
	return &routeRegister{
		server: server,
		method: method,
		path:   path,
	}
}

// Middleware 为特定路由添加中间件
func (r *routeRegister) Middleware(middleware ...Middleware) RouteRegister {
	for _, m := range middleware {
		r.server.Use(r.method, r.path, m)
	}
	return r
}

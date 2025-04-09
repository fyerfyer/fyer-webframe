package web

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/fyerfyer/fyer-kit/pool"
	objPool "github.com/fyerfyer/fyer-webframe/web/pool"
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
	GetTemplateEngine() Template
}

// RouteRegister 路由链式注册接口
type RouteRegister interface {
	// Middleware 为特定路由添加中间件
	Middleware(middleware ...Middleware) RouteRegister
}

// HTTPServer 结构体
type HTTPServer struct {
	*Router     // 继承Router
	start       bool
	noRouter    HandlerFunc      // 404处理器
	server      *http.Server     // 底层的http server
	baseRoute   string           // 基础路由前缀
	tplEngine   Template         // 模板引擎
	poolManager pool.PoolManager // 连接池管理器
	useObjPool  bool             // 是否使用对象池
	paramCap    int              // 参数映射的初始容量
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

// WithPoolManager 设置连接池管理器
func WithPoolManager(manager pool.PoolManager) ServerOption {
	return func(server *HTTPServer) {
		server.poolManager = manager
	}
}

// WithObjectPool 启用对象池以减少GC压力
func WithObjectPool(paramCap int) ServerOption {
	return func(server *HTTPServer) {
		server.useObjPool = true
		if paramCap > 0 {
			server.paramCap = paramCap
		}
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
		paramCap: 8, // 默认参数容量
	}

	// 应用所有选项
	for _, opt := range opts {
		opt(server)
	}

	// 设置 http.Server 的处理器为当前实例
	server.server.Handler = server
	return server
}

// initObjectPool 初始化对象池
func (s *HTTPServer) initObjectPool() {
	if s.useObjPool && objPool.DefaultContextPool == nil {
		InitContextPool(s.tplEngine, s.poolManager, s.paramCap)
	}
}

// ServeHTTP HTTPServer的核心处理函数
func (s *HTTPServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// 确保对象池已初始化
	s.initObjectPool()

	var ctx *Context
	// 使用对象池创建上下文
	if s.useObjPool && objPool.DefaultContextPool != nil {
		ctx = AcquireContext(req, res)
	} else {
		// 不使用对象池时，直接创建
		ctx = &Context{
			Req:         req,
			Resp:        res,
			Param:       make(map[string]string, s.paramCap),
			tplEngine:   s.tplEngine,
			Context:     req.Context(),
			unhandled:   true,
			UserValues:  make(map[string]any, s.paramCap),
			poolManager: s.poolManager,
		}
	}

	// 在函数返回时释放对象（如果使用了对象池）
	if s.useObjPool && objPool.DefaultContextPool != nil {
		defer ReleaseContext(ctx)
	}

	// 如果设置了基础路径，需要处理路径前缀
	originalPath := req.URL.Path
	path := originalPath

	// 检查请求路径是否以基础路由开头
	if s.baseRoute != "" {
		if len(path) >= len(s.baseRoute) && path[:len(s.baseRoute)] == s.baseRoute {
			// 剥离基础路径
			path = path[len(s.baseRoute):]
			if path == "" {
				path = "/"
			}
		} else {
			// 如果设置了基础路由但路径不匹配，返回404
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
	// 确保对象池已初始化
	s.initObjectPool()

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

	// 关闭连接池管理器
	if s.poolManager != nil {
		if err := s.poolManager.Shutdown(ctx); err != nil {
			return err
		}
	}

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

// GetTemplateEngine 返回服务器使用的模板引擎
func (s *HTTPServer) GetTemplateEngine() Template {
	return s.tplEngine
}

// PoolManager 返回连接池管理器
func (s *HTTPServer) PoolManager() pool.PoolManager {
	return s.poolManager
}

// SetPoolManager 设置连接池管理器
func (s *HTTPServer) SetPoolManager(manager pool.PoolManager) {
	s.poolManager = manager
}

// EnableObjectPool 启用对象池功能
func (s *HTTPServer) EnableObjectPool(paramCap int) {
	s.useObjPool = true
	if paramCap > 0 {
		s.paramCap = paramCap
	}
	s.initObjectPool()
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
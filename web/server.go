package web

import (
	"context"
	"net"
	"net/http"
	"time"
)

type HandlerFunc func(ctx *Context)

// Server 接口定义
type Server interface {
	http.Handler
	// Start 启动服务器
	Start(addr string) error
	// Shutdown 优雅关闭
	Shutdown(ctx context.Context) error
	// AddRoute 注册路由
	addRoute(method string, path string, handler HandlerFunc)
	// Use 注册中间件
	Use(method string, path string, middleware ...Middleware)
}

// HTTPServer 结构体
type HTTPServer struct {
	*Router   // 继承Router
	start     bool
	noRouter  HandlerFunc  // 404处理器
	server    *http.Server // 底层的http server
	baseRoute string       // 基础路由前缀
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
		Req:   req,
		Resp:  res,
		Param: make(map[string]string),
	}

	// 如果设置了基础路径，需要处理路径前缀
	path := req.URL.Path
	if s.baseRoute != "" {
		if len(path) >= len(s.baseRoute) && path[:len(s.baseRoute)] == s.baseRoute {
			path = path[len(s.baseRoute):]
			if path == "" {
				path = "/"
			}
		}
	}

	// 查找路由
	node, ok := s.findHandler(req.Method, path, ctx)
	if !ok {
		s.noRouter(ctx)
		return
	}

	// 构建并执行处理链
	handler := BuildChain(node, node.handler)
	handler(ctx)
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

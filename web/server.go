package web

import (
	"context"
	"net"
	"net/http"
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
	*Router  // 继承Router
	start    bool
	noRouter HandlerFunc // 404处理器
}

// NewHTTPServer 创建HTTP服务器实例
func NewHTTPServer() *HTTPServer {
	return &HTTPServer{
		Router: NewRouter(),
		noRouter: func(ctx *Context) {
			ctx.Resp.WriteHeader(http.StatusNotFound)
		},
	}
}

// ServeHTTP HTTPServer的核心处理函数
func (s *HTTPServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	ctx := &Context{
		Req:   req,
		Resp:  res,
		Param: make(map[string]string),
	}

	// 查找路由
	node, ok := s.findHandler(req.Method, req.URL.Path, ctx)
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
	return http.Serve(listen, s)
}

// Shutdown 优雅关闭
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	s.start = false
	return nil
}

package web

import (
	"context"
	"net"
	"net/http"
)

// 确保 HTTPServer 实现了 Server 接口
var _ Server = &HTTPServer{}

type Server interface {
	http.Handler
	Start(addr string) error
	Shutdown(ctx context.Context) error
}

type HTTPServer struct {
	srv *http.Server
}

func (s *HTTPServer) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	res.Write([]byte("Hello, World!"))
}

func (s *HTTPServer) Start(addr string) error {
	listener, err := net.Listen("tcp", addr) // 监听端口
	if err != nil {
		return err
	}

	s.srv = &http.Server{Handler: s}

	// 启动 HTTP 服务器
	return s.srv.Serve(listener)
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	if s.srv != nil {
		return s.srv.Shutdown(ctx)
	}
	return nil
}

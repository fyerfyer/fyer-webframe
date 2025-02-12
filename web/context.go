package web

import "testing"

func TestServerConn(t *testing.T) {
	srv := &HTTPServer{}
	srv.Start(":8080")
}

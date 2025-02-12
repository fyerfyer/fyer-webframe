package web

import (
	"context"
	"github.com/stretchr/testify/assert"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	srv := &HTTPServer{}

	listener, err := net.Listen("tcp", "127.0.0.1:8080")
	assert.NoError(t, err, "should be able to get a free port")
	addr := listener.Addr().String()
	listener.Close()

	go func() {
		_ = srv.Start(addr)
	}()

	time.Sleep(200 * time.Millisecond)

	resp, err := http.Get("http://" + addr)
	assert.NoError(t, err, "should be able to send request")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "server should respond with 200")

	err = srv.Shutdown(context.Background())
	assert.NoError(t, err, "server should shutdown without error")
}

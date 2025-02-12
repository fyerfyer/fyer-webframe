package web

import "net/http"

type Context struct {
	Request   *http.Request
	ResWriter http.ResponseWriter
}

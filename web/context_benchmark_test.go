package web

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type benchUser struct {
	ID   int    `json:"id" xml:"id"`
	Name string `json:"name" xml:"name"`
}

func BenchmarkContextJSON(b *testing.B) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := &Context{
		Req:  req,
		Resp: w,
	}

	user := &benchUser{ID: 123, Name: "tester"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		ctx.JSON(200, user)
	}
}

func BenchmarkContextXML(b *testing.B) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := &Context{
		Req:  req,
		Resp: w,
	}

	user := &benchUser{ID: 123, Name: "tester"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		ctx.XML(200, user)
	}
}

func BenchmarkContextString(b *testing.B) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := &Context{
		Req:  req,
		Resp: w,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		ctx.String(200, "Hello, %s!", "World")
	}
}

func BenchmarkContextBindJSON(b *testing.B) {
	user := &benchUser{ID: 123, Name: "tester"}
	data, _ := json.Marshal(user)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/json")
		ctx := &Context{Req: req}

		result := &benchUser{}
		ctx.BindJSON(result)
	}
}

func BenchmarkContextBindXML(b *testing.B) {
	user := &benchUser{ID: 123, Name: "tester"}
	data, _ := xml.Marshal(user)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/xml")
		ctx := &Context{Req: req}

		result := &benchUser{}
		ctx.BindXML(result)
	}
}

func BenchmarkContextPathParam(b *testing.B) {
	ctx := &Context{
		Param: map[string]string{
			"id": "123",
			"name": "test",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.PathParam("id")
	}
}

func BenchmarkContextQueryParam(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/?id=123&name=test", nil)
	ctx := &Context{Req: req}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.QueryParam("id")
	}
}

func BenchmarkContextFormValue(b *testing.B) {
	formData := "id=123&name=test"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := &Context{Req: req}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.FormValue("id")
	}
}

func BenchmarkMiddlewareChain(b *testing.B) {
	handler := func(ctx *Context) {
		ctx.String(200, "OK")
	}

	middlewares := []MiddlewareWithPath{
		{
			Middleware: func(next HandlerFunc) HandlerFunc {
				return func(ctx *Context) {
					next(ctx)
				}
			},
			Path:  "/benchmark",
			Type:  StaticMiddleware,
			Order: 1,
		},
		{
			Middleware: func(next HandlerFunc) HandlerFunc {
				return func(ctx *Context) {
					next(ctx)
				}
			},
			Path:  "/benchmark/*",
			Type:  WildcardMiddleware,
			Order: 2,
		},
	}

	chainedHandler := BuildChain(handler, "/benchmark/test", middlewares)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/benchmark/test", nil)
	ctx := &Context{Req: req, Resp: w}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		chainedHandler(ctx)
	}
}

func BenchmarkRouter_Static(b *testing.B) {
	router := NewRouter()
	router.Get("/users", func(ctx *Context) {
		ctx.String(200, "Users List")
	})

	server := HTTPServer{Router: router}
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		server.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_Param(b *testing.B) {
	router := NewRouter()
	router.Get("/users/:id", func(ctx *Context) {
		ctx.String(200, "User ID: %s", ctx.PathParam("id").Value)
	})

	server := HTTPServer{Router: router}
	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		server.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_Regex(b *testing.B) {
	router := NewRouter()
	router.Get("/users/:id([0-9]+)", func(ctx *Context) {
		ctx.String(200, "User ID: %s", ctx.PathParam("id").Value)
	})

	server := HTTPServer{Router: router}
	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		server.ServeHTTP(w, req)
	}
}

func BenchmarkRouter_Wildcard(b *testing.B) {
	router := NewRouter()
	router.Get("/static/*", func(ctx *Context) {
		ctx.String(200, "Static file")
	})

	server := HTTPServer{Router: router}
	req := httptest.NewRequest(http.MethodGet, "/static/css/style.css", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		server.ServeHTTP(w, req)
	}
}

func BenchmarkContextAbort(b *testing.B) {
	ctx := &Context{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.aborted = false
		ctx.Abort()
	}
}

func BenchmarkContextNext(b *testing.B) {
	ctx := &Context{
		Resp: httptest.NewRecorder(), // 初始化Resp字段
	}
	handler := func(ctx *Context) {
		ctx.String(200, "OK")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.aborted = false
		ctx.Next(handler)
	}
}

func BenchmarkContext_ReadBody(b *testing.B) {
	body := []byte(`{"id": 123, "name": "tester"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(body))
		ctx := &Context{Req: req}
		ctx.ReadBody()
	}
}

func BenchmarkHTTPServer_ServeHTTP(b *testing.B) {
	server := NewHTTPServer()
	server.Get("/benchmark", func(ctx *Context) {
		ctx.String(200, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/benchmark", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		server.ServeHTTP(w, req)
	}
}
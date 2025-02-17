package middleware

import "github.com/fyerfyer/fyer-webframe/web"

type MiddlewareBuilder struct {
	logger func(content string)
}

type logInfo struct {
	Host       string
	Route      string
	Path       string
	HttpMethod string
}

func (m *MiddlewareBuilder) SetLogger(logger func(content string)) *MiddlewareBuilder {
	m.logger = logger
	return m
}

func NewMiddlewareBuilder() *MiddlewareBuilder {
	return &MiddlewareBuilder{
		logger: func(content string) {
			println(content)
		},
	}
}

func (m *MiddlewareBuilder) Build() web.Middleware {
	return func(handler web.HandlerFunc) web.HandlerFunc {
		return func(ctx *web.Context) {
			info := logInfo{
				Host: ctx.Req.Host,
				Route: 
			}
		}
	}
}

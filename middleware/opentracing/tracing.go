package opentracing

import (
	"github.com/fyerfyer/fyer-webframe/web"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type MiddlewareBuilder struct {
	tracer trace.Tracer
}

var defaultInstrumentationName = "fyer-webframe"

func (m *MiddlewareBuilder) Build() web.Middleware {
	if m.tracer == nil {
		m.tracer = otel.GetTracerProvider().Tracer(defaultInstrumentationName)
	}

	return func(handlerFunc web.HandlerFunc) web.HandlerFunc {
		return func(ctx *web.Context) {
			reqCtx := ctx.Req.Context()
			reqCtx = otel.GetTextMapPropagator().Extract(reqCtx, propagation.HeaderCarrier(ctx.Req.Header))
			reqCtx, span := m.tracer.Start(reqCtx, "unknown")
			defer span.End()

			span.SetAttributes(attribute.String("http.method", ctx.Req.Method))
			span.SetAttributes(attribute.String("http.host", ctx.Req.Host))
			span.SetAttributes(attribute.String("http.url", ctx.Req.URL.String()))
			span.SetAttributes(attribute.String("http.scheme", ctx.Req.URL.Scheme))
			span.SetAttributes(attribute.String("span.kind", "server"))
			span.SetAttributes(attribute.String("component", "web"))
			span.SetAttributes(attribute.String("http.proto", ctx.Req.Proto))
		}
	}
}

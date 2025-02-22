package prometheus

import (
	"github.com/fyerfyer/fyer-webframe/web"
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
	"time"
)

type MiddlewareBuilder struct {
	NameSpace string
	Name      string
	SubSystem string
	Help      string
}

func (m *MiddlewareBuilder) Build() web.Middleware {
	vec := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name:      m.Name,
		Help:      m.Help,
		Namespace: m.NameSpace,
		Subsystem: m.SubSystem,
		Objectives: map[float64]float64{
			0.5:   0.05,
			0.9:   0.01,
			0.99:  0.001,
			0.999: 0.0001,
		},
	}, []string{"method", "path", "status"})

	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(ctx *web.Context) {
			startTime := time.Now()
			defer func() {
				duration := time.Now().Sub(startTime).Microseconds()
				vec.WithLabelValues(ctx.Req.Method,
					ctx.RouteURL,
					strconv.Itoa(ctx.RespStatusCode)).
					Observe(float64(duration))
			}()

			next(ctx)
		}
	}
}

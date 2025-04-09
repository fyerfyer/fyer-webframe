package accesslog

import (
	"github.com/fyerfyer/fyer-webframe/web"
	"github.com/fyerfyer/fyer-webframe/web/logger"
	"time"
)

// Config 访问日志中间件配置
type Config struct {
	// 跳过日志记录的路径
	SkipPaths []string
	// 慢请求阈值（毫秒）
	SlowThreshold time.Duration
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		SkipPaths:     make([]string, 0),
		SlowThreshold: 500 * time.Millisecond,
	}
}

// New 创建一个默认配置的访问日志中间件
func New() web.Middleware {
	return NewWithConfig(DefaultConfig())
}

// NewWithConfig 使用自定义配置创建访问日志中间件
func NewWithConfig(config *Config) web.Middleware {
	// 创建跳过路径的映射以便快速查找
	skipMap := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipMap[path] = true
	}

	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(ctx *web.Context) {
			// 如果路径在跳过列表中，不记录日志
			if skipMap[ctx.Req.URL.Path] {
				next(ctx)
				return
			}

			// 记录开始时间
			start := time.Now()

			// 准备请求字段
			reqFields := []logger.Field{
				logger.String("method", ctx.Req.Method),
				logger.String("path", ctx.Req.URL.Path),
				logger.String("client_ip", ctx.ClientIP()),
				logger.String("user_agent", ctx.UserAgent()),
			}

			// 记录请求开始
			ctx.Logger().Info("Request started", reqFields...)

			// 执行下一个处理器
			next(ctx)

			// 计算处理时间
			duration := time.Since(start)

			// 准备响应字段
			respFields := append([]logger.Field{
				logger.Int("status", ctx.RespStatusCode),
				logger.Int64("duration_ms", duration.Milliseconds()),
				logger.Int("resp_size", len(ctx.RespData)),
			}, reqFields...)

			// 根据状态码和响应时间选择日志级别
			if ctx.RespStatusCode >= 500 {
				ctx.Logger().Error("Request failed with server error", respFields...)
			} else if ctx.RespStatusCode >= 400 {
				ctx.Logger().Warn("Request failed with client error", respFields...)
			} else if duration > config.SlowThreshold {
				ctx.Logger().Warn("Slow request completed", respFields...)
			} else {
				ctx.Logger().Info("Request completed", respFields...)
			}
		}
	}
}
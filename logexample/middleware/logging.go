package middleware

import (
	"time"

	"github.com/fyerfyer/fyer-webframe/web"
	"github.com/fyerfyer/fyer-webframe/web/logger"
	"github.com/google/uuid"
)

// RequestLogger 请求日志中间件
func RequestLogger(next web.HandlerFunc) web.HandlerFunc {
	return func(ctx *web.Context) {
		// 记录请求开始时间
		start := time.Now()

		// 添加请求信息
		requestLog := ctx.Logger().WithFields(
			logger.String("method", ctx.Req.Method),
			logger.String("path", ctx.Req.URL.Path),
			logger.String("client_ip", ctx.ClientIP()),
			logger.String("user_agent", ctx.UserAgent()),
		)

		// 记录请求开始
		requestLog.Info("Request started")

		// 处理请求
		next(ctx)

		// 计算处理时间
		duration := time.Since(start)

		// 根据状态码确定日志级别
		statusCode := ctx.RespStatusCode
		switch {
		case statusCode >= 500:
			requestLog.Error("Request completed with server error",
				logger.Int("status", statusCode),
				logger.Int64("duration_ms", duration.Milliseconds()),
			)
		case statusCode >= 400:
			requestLog.Warn("Request completed with client error",
				logger.Int("status", statusCode),
				logger.Int64("duration_ms", duration.Milliseconds()),
			)
		default:
			requestLog.Info("Request completed successfully",
				logger.Int("status", statusCode),
				logger.Int64("duration_ms", duration.Milliseconds()),
			)
		}
	}
}

// RequestID 添加请求ID中间件
func RequestID(next web.HandlerFunc) web.HandlerFunc {
	return func(ctx *web.Context) {
		// 检查是否已有请求ID
		requestID := ctx.GetHeader("X-Request-ID")
		if requestID == "" {
			// 生成新的请求ID
			requestID = uuid.New().String()
			ctx.SetHeader("X-Request-ID", requestID)
		}

		// 更新日志记录器，加入请求ID
		ctx.SetLogger(ctx.Logger().WithField("request_id", requestID))

		// 记录请求ID
		ctx.Logger().Debug("Request ID assigned", logger.String("request_id", requestID))

		// 处理请求
		next(ctx)
	}
}

// AdminLogger 管理页面特定的日志中间件
func AdminLogger(next web.HandlerFunc) web.HandlerFunc {
	return func(ctx *web.Context) {
		// 为管理页面添加特殊的日志字段
		adminLogger := ctx.Logger().WithFields(
			logger.String("access_type", "admin"),
			logger.String("security_level", "restricted"),
		)

		// 替换上下文的日志记录器
		ctx.SetLogger(adminLogger)

		// 记录管理页面访问
		ctx.Logger().Warn("Admin page access attempt",
			logger.String("client_ip", ctx.ClientIP()),
			logger.String("user_agent", ctx.UserAgent()),
		)

		// 处理请求
		next(ctx)
	}
}
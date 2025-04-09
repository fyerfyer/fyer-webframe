package recovery

import (
    "fmt"
    "github.com/fyerfyer/fyer-webframe/web"
    "github.com/fyerfyer/fyer-webframe/web/logger"
    "runtime"
    "strings"
    "time"
)

// Recovery 返回一个恢复panic并将其转换为HTTP 500错误的中间件
func Recovery() web.Middleware {
    return func(next web.HandlerFunc) web.HandlerFunc {
        return func(ctx *web.Context) {
            defer func() {
                if err := recover(); err != nil {
                    // 获取堆栈跟踪信息
                    stackTrace := getStackTrace(3) // 跳过前3个堆栈帧，获取更相关的信息

                    // 准备结构化日志字段
                    fields := []logger.Field{
                        logger.FieldError(fmt.Errorf("%v", err)),
                        logger.String("stack_trace", stackTrace),
                        logger.String("method", ctx.Req.Method),
                        logger.String("path", ctx.Req.URL.Path),
                        logger.String("client_ip", ctx.ClientIP()),
                    }

                    // 记录错误日志
                    ctx.Logger().Error("Panic recovered", fields...)

                    // 返回500错误给客户端
                    // 可以生成一个唯一ID，方便用户报告问题时关联日志
                    errorID := fmt.Sprintf("%d", time.Now().UnixNano())
                    ctx.InternalServerError(fmt.Sprintf("Internal server error occurred. Error ID: %s", errorID))
                }
            }()

            // 执行下一个处理器
            next(ctx)
        }
    }
}

// getStackTrace 生成格式化的堆栈跟踪信息
func getStackTrace(skip int) string {
    // 分配缓冲区获取堆栈信息
    buf := make([]byte, 4096)
    n := runtime.Stack(buf, false)
    stackInfo := string(buf[:n])

    // 分割堆栈信息，丢弃前面的运行时帧
    lines := strings.Split(stackInfo, "\n")
    if len(lines) <= skip*2 {
        return stackInfo // 如果堆栈太短就返回完整信息
    }

    // 保留关键堆栈帧
    relevantLines := lines[skip*2:]
    // 限制堆栈大小，避免日志过长
    if len(relevantLines) > 20 {
        relevantLines = relevantLines[:20]
        relevantLines = append(relevantLines, "...stack trace truncated...")
    }

    return strings.Join(relevantLines, "\n")
}
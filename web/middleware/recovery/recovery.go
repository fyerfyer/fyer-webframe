package recovery

import (
    "fmt"
    "github.com/fyerfyer/fyer-webframe/web"
    "log"
    "runtime"
    "strings"
)

// Recovery 返回一个恢复 panic 并将其转换为 HTTP 500 错误的中间件
func Recovery() web.Middleware {
    return func(next web.HandlerFunc) web.HandlerFunc {
        return func(ctx *web.Context) {
            defer func() {
                if err := recover(); err != nil {
                    // 获取调用栈
                    buf := make([]byte, 4096)
                    n := runtime.Stack(buf, false)
                    stackTrace := string(buf[:n])

                    errMsg := fmt.Sprintf("PANIC: %v\n%s", err, stackTrace)
                    fmt.Printf("%s\n", errMsg)

                    lines := strings.Split(stackTrace, "\n")
                    userStackTrace := strings.Join(lines[0:6], "\n")
                    // todo: 可让用户自定义日志
                    log.Println(userStackTrace)

                    // 返回 500 错误
                    ctx.InternalServerError(fmt.Sprintf("Internal server error occurred. Error ID: %v", err))
                }
            }()
            
            next(ctx)
        }
    }
}
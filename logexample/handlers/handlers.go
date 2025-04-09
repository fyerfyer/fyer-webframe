package handlers

import (
	"errors"
	"math/rand"
	"os"
	"time"

	"github.com/fyerfyer/fyer-webframe/web"
	"github.com/fyerfyer/fyer-webframe/web/logger"
)

// HomePage 显示首页
func HomePage(ctx *web.Context) {
	// 使用上下文日志记录器记录日志
	ctx.Logger().Info("Homepage requested",
		logger.String("user_agent", ctx.UserAgent()),
		logger.String("client_ip", ctx.ClientIP()),
	)

	// 添加一些模拟的处理延迟
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)

	// 渲染模板
	err := ctx.Template("index.html", map[string]interface{}{
		"title":   "Web框架日志示例",
		"message": "这是一个简单的示例，展示日志系统的功能",
		"time":    time.Now().Format(time.RFC3339),
	})

	if err != nil {
		ctx.LogError("Failed to render template", err)
		ctx.InternalServerError("无法加载页面")
	}
}

// GetInfo 返回API信息
func GetInfo(ctx *web.Context) {
	ctx.Logger().Info("API info requested")

	// 记录字段信息
	ctx.Logger().Debug("Processing API info request",
		logger.String("details", "Fetching system information"),
		logger.Int("process_id", os.Getpid()),
	)

	// 返回JSON响应
	ctx.JSON(200, map[string]interface{}{
		"name":    "Web框架日志示例",
		"version": "1.0.0",
		"status":  "running",
		"time":    time.Now().Format(time.RFC3339),
	})
}

// CreateUser 创建用户（演示不同日志级别）
func CreateUser(ctx *web.Context) {
	// 记录开始处理请求
	ctx.Logger().Info("Creating new user")

	// 解析用户数据
	var userData struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}

	if err := ctx.BindJSON(&userData); err != nil {
		ctx.LogError("Failed to parse user data", err)
		ctx.BadRequest("无效的用户数据")
		return
	}

	// 记录业务信息
	ctx.Logger().Debug("User data parsed successfully",
		logger.String("username", userData.Username),
		logger.String("email", userData.Email),
	)

	// 业务逻辑验证
	if userData.Username == "" || userData.Email == "" {
		ctx.Logger().Warn("Validation failed - missing required fields",
			logger.String("username", userData.Username),
			logger.String("email", userData.Email),
		)
		ctx.BadRequest("用户名和邮箱不能为空")
		return
	}

	// 假设创建用户成功
	ctx.Logger().Info("User created successfully",
		logger.String("username", userData.Username),
	)

	ctx.JSON(201, map[string]interface{}{
		"id":       rand.Intn(1000) + 1,
		"username": userData.Username,
		"email":    userData.Email,
		"created":  time.Now().Format(time.RFC3339),
	})
}

// SimulateError 模拟错误并记录
func SimulateError(ctx *web.Context) {
	// 创建错误
	err := errors.New("这是一个模拟的错误")

	// 记录错误级别日志
	ctx.Logger().Error("An error occurred during processing",
		logger.FieldError(err),
		logger.String("operation", "error_simulation"),
		logger.Int("random_value", rand.Intn(100)),
	)

	// 返回错误响应
	ctx.InternalServerError("处理请求时出错")
}

// SimulatePanic 模拟panic记录
func SimulatePanic(ctx *web.Context) {
	// 使用Fatal记录致命错误日志
	ctx.Logger().Fatal("Fatal error occurred - application will be terminated",
		logger.String("operation", "panic_simulation"),
		logger.String("severity", "critical"),
	)

	// 返回错误响应（在实际的panic情况下不会执行到这里，但我们不希望真正触发panic）
	ctx.InternalServerError("发生致命错误")
}

// DebugPage 显示调试信息页面
func DebugPage(ctx *web.Context) {
	// 记录大量调试信息
	ctx.Logger().Debug("Debug page requested",
		logger.String("client_ip", ctx.ClientIP()),
		logger.String("user_agent", ctx.UserAgent()),
		logger.String("referer", ctx.Referer()),
	)

	// 记录所有请求头
	headers := make(map[string]string)
	for name, values := range ctx.Req.Header {
		if len(values) > 0 {
			headers[name] = values[0]
		}
	}

	ctx.Logger().Debug("Request headers", logger.Interface("headers", headers))

	// 记录查询参数
	queries := ctx.QueryAll()
	ctx.Logger().Debug("Query parameters", logger.Interface("queries", queries))

	// 返回调试信息
	ctx.JSON(200, map[string]interface{}{
		"headers":     headers,
		"queries":     queries,
		"client_ip":   ctx.ClientIP(),
		"user_agent":  ctx.UserAgent(),
		"request_url": ctx.Req.URL.String(),
		"time":        time.Now().Format(time.RFC3339),
	})
}

// AdminPage 管理员页面（使用特定的中间件）
func AdminPage(ctx *web.Context) {
	ctx.Logger().Info("Admin page accessed")

	// 假设进行权限检查
	ctx.Logger().Debug("Performing admin authorization check")

	// 返回管理页面
	ctx.HTML(200, `
        <html>
            <head>
                <title>管理面板</title>
                <link rel="stylesheet" href="/static/style.css">
            </head>
            <body>
                <h1>管理面板</h1>
                <p>这是管理页面，访问时会生成特定的日志记录</p>
            </body>
        </html>
    `)
}
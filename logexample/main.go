package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fyerfyer/fyer-webframe/logexample/handlers"
	"github.com/fyerfyer/fyer-webframe/logexample/middleware"
	"github.com/fyerfyer/fyer-webframe/web"
	"github.com/fyerfyer/fyer-webframe/web/logger"
)

func main() {
	// 配置日志
	setupLogger()

	// 创建服务器
	server := setupServer()

	// 启动服务器
	addr := ":8080"
	fmt.Printf("Server starting on %s\n", addr)
	err := server.Start(addr)
	if err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
		os.Exit(1)
	}
}

func setupLogger() {
	// 创建控制台日志
	consoleLogger := logger.NewLogger(
		logger.WithLevel(logger.DebugLevel),  // 设置日志级别为Debug
		logger.WithOutput(os.Stdout),         // 输出到标准输出
	)

	// 创建文件日志（同时保留控制台输出）
	logFile, err := os.OpenFile("example.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		// 创建多输出日志，同时写入控制台和文件
		multiWriter := io.MultiWriter(os.Stdout, logFile)
		fileLogger := logger.NewLogger(
			logger.WithLevel(logger.InfoLevel),
			logger.WithOutput(multiWriter),
		)
		// 设置为默认日志记录器
		logger.SetDefaultLogger(fileLogger)
	} else {
		// 如果无法创建文件，仅使用控制台日志并输出警告
		logger.SetDefaultLogger(consoleLogger)
		logger.Warn("Unable to create log file", logger.FieldError(err))
	}

	// 记录应用启动日志
	logger.Info("Application starting",
		logger.String("app", "example"),
		logger.String("version", "1.0.0"),
		logger.String("time", time.Now().Format(time.RFC3339)),
	)
}

func setupServer() web.Server {
	// 创建HTTP服务器实例，使用默认日志记录器
	server := web.NewHTTPServer()

	// 设置模板引擎
	tplEngine := web.NewGoTemplate(
		web.WithPattern("./templates/*.html"),
		web.WithAutoReload(true),
	)
	server.UseTemplate(tplEngine)

	// 注册全局中间件
	server.Middleware().Global().Add(middleware.RequestLogger)

	// 注册静态资源路由 - 改进版本
	server.Get("/static/*", func(ctx *web.Context) {
		// 获取文件路径
		filePath := ctx.PathParam("*").Value
		if filePath == "" {
			ctx.LogError("Empty static file path", errors.New("empty path"))
			ctx.BadRequest("Invalid static file path")
			return
		}

		// 构建完整的物理路径
		fullPath := filepath.Join("static", filePath)
		ctx.Logger().Debug("Serving static file",
			logger.String("path", fullPath),
			logger.String("requested_path", filePath))

		// 检查文件是否存在
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			ctx.LogError("Static file not found", err)
			ctx.NotFound("Static file not found")
			return
		}

		// 设置适当的内容类型
		if strings.HasSuffix(filePath, ".css") {
			ctx.Resp.Header().Set("Content-Type", "text/css")
		} else if strings.HasSuffix(filePath, ".js") {
			ctx.Resp.Header().Set("Content-Type", "application/javascript")
		}

		// 提供文件
		err := ctx.File(fullPath)
		if err != nil {
			ctx.LogError("Failed to serve static file", err)
			ctx.InternalServerError("Failed to serve file")
		}
	})

	// 注册API路由组，添加请求ID中间件
	apiGroup := server.Group("/api")
	apiGroup.Use(middleware.RequestID)

	// 注册API路由
	apiGroup.Get("/info", handlers.GetInfo)
	apiGroup.Post("/users", handlers.CreateUser)
	apiGroup.Get("/error", handlers.SimulateError)
	apiGroup.Get("/panic", handlers.SimulatePanic)

	// 注册Web页面路由
	server.Get("/", handlers.HomePage)
	server.Get("/debug", handlers.DebugPage)

	// 添加路由级别的日志中间件示例
	server.Get("/admin", handlers.AdminPage).Middleware(middleware.AdminLogger)

	return server
}
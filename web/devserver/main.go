package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fyerfyer/fyer-webframe/web"
	"github.com/fyerfyer/fyer-webframe/web/hotreload"
)

// DevServer 开发服务器结构体
type DevServer struct {
	config     *hotreload.Config  // 热重载配置
	manager    *hotreload.Manager // 热重载管理器
	flags      *Flags             // 命令行参数
	httpServer web.Server         // Web 服务器实例
}

// Run 启动开发服务器
func Run() error {
	// 解析命令行参数
	flags, err := ParseFlags()
	if err != nil {
		return fmt.Errorf("failed to parse command line flags: %w", err)
	}

	// 创建开发服务器
	server := &DevServer{
		flags: flags,
	}

	// 初始化服务器
	if err := server.init(); err != nil {
		return fmt.Errorf("failed to initialize dev server: %w", err)
	}

	// 启动服务器
	return server.run()
}

// init 初始化开发服务器
func (s *DevServer) init() error {
	// 创建热重载配置
	s.config = hotreload.NewConfig(
		hotreload.WithWatchPaths(s.flags.WatchPaths...),
		hotreload.WithIgnorePaths(s.flags.IgnorePaths...),
		hotreload.WithRestartDelay(time.Duration(s.flags.RestartDelay)*time.Millisecond),
		hotreload.WithEntryPoint(s.flags.EntryPoint),
		hotreload.WithAppArgs(s.flags.AppArgs...),
		hotreload.WithTempDir(s.flags.TempDir),
		hotreload.WithShowBanner(!s.flags.NoBanner),
		hotreload.WithShowBuildOutput(!s.flags.QuietBuild),
		hotreload.WithShowAppOutput(!s.flags.QuietApp),
		hotreload.WithTemplatePattern(s.flags.TemplatePattern),
	)

	// 设置构建命令和参数
	if s.flags.BuildCmd != "" {
		hotreload.WithBuildCommand(s.flags.BuildCmd, s.flags.BuildArgs...)(s.config)
	}

	// 设置环境变量
	if len(s.flags.Env) > 0 {
		hotreload.WithEnvironment(s.flags.Env)(s.config)
	}

	// 创建热重载管理器
	manager, err := hotreload.NewManager(s.config)
	if err != nil {
		return fmt.Errorf("failed to create hot reload manager: %w", err)
	}
	s.manager = manager

	// 如果指定了模板目录，设置模板监控
	if s.flags.TemplatePattern != "" {
		// 创建模板引擎
		tpl := web.NewGoTemplate(
			web.WithPattern(s.flags.TemplatePattern),
		)

		// 将模板引擎添加到热重载管理器中
		s.manager.AddTemplateMonitor(tpl, s.flags.TemplatePattern)

		// 创建 HTTP 服务器并使用模板引擎
		s.httpServer = web.NewHTTPServer(
			web.WithTemplate(tpl),
		)
	}

	// 初始化工作目录
	if s.flags.WorkDir != "" {
		if err := os.Chdir(s.flags.WorkDir); err != nil {
			return fmt.Errorf("failed to change working directory: %w", err)
		}
		log.Printf("Changed working directory to: %s", s.flags.WorkDir)
	}

	// 确保临时目录存在
	if s.flags.TempDir != "" {
		if err := os.MkdirAll(s.flags.TempDir, 0755); err != nil {
			return fmt.Errorf("failed to create temporary directory: %w", err)
		}
	}

	return nil
}

// run 运行开发服务器
func (s *DevServer) run() error {
	// 打印启动信息
	if !s.flags.NoBanner {
		printBanner(s.flags)
	}

	// 打印配置信息
	if s.flags.Verbose {
		printConfig(s.config)
	}

	// 启动热重载管理器
	if err := s.manager.Start(); err != nil {
		return fmt.Errorf("failed to start hot reload manager: %w", err)
	}

	// 设置退出信号处理
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 等待退出信号
	<-quit
	log.Println("Received shutdown signal, closing server...")

	// 创建一个带超时的上下文用于优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 关闭热重载管理器
	if err := s.manager.Stop(); err != nil {
		log.Printf("Error closing hot reload manager: %v", err)
	}

	// 如果有 HTTP 服务器实例，也关闭它
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Printf("Error closing HTTP server: %v", err)
		}
	}

	log.Println("Server shutdown completed")
	return nil
}

// printBanner 打印欢迎信息
func printBanner(flags *Flags) {
	fmt.Println(`
    ______               __             __    
   / ____/_  _____  ____/ /__  _   ____/ /___ 
  / __/ / / / / _ \/ __  / _ \| | / / _/ / _ \
 / /   / /_/ /  __/ /_/ /  __/| |/ / /_/ /  __/
/_/    \__, /\___/\__,_/\___(_)___/\__/_/\___/ 
      /____/                                   
                                               
     Hot Reload Development Server
`)
	fmt.Printf("Entry Point: %s\n", flags.EntryPoint)
	fmt.Printf("Watching: %v\n", flags.WatchPaths)
	fmt.Printf("Ignoring: %v\n", flags.IgnorePaths)
	fmt.Println("Press Ctrl+C to exit")
	fmt.Println("")
}

// printConfig 打印配置信息
func printConfig(config *hotreload.Config) {
	fmt.Println("--- Hot Reload Configuration ---")
	fmt.Printf("Temp Directory: %s\n", config.TempDir)
	fmt.Printf("Restart Delay: %v\n", config.RestartDelay)

	// 如果有模板匹配模式
	tplPattern := "None"
	if config.GetTemplatePattern() != "" {
		matches, err := filepath.Glob(config.GetTemplatePattern())
		if err == nil {
			tplPattern = fmt.Sprintf("%s (%d files)", config.GetTemplatePattern(), len(matches))
		} else {
			tplPattern = fmt.Sprintf("%s (error: %v)", config.GetTemplatePattern(), err)
		}
	}
	fmt.Printf("Template Pattern: %s\n", tplPattern)
	fmt.Println("------------------------------")
}

// 主函数
func main() {
	if err := Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
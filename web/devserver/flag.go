package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

// Flags 命令行参数
type Flags struct {
	// 项目相关参数
	EntryPoint      string   // 应用入口点
	WorkDir         string   // 工作目录
	TempDir         string   // 临时文件目录
	TemplatePattern string   // 模板文件匹配模式

	// 监控相关参数
	WatchPaths   []string        // 要监控的路径
	IgnorePaths  []string        // 要忽略的路径
	RestartDelay int             // 重启延迟 (ms)
	LiveReload   bool            // 是否启用浏览器自动刷新
	PollInterval time.Duration   // 轮询间隔(如果不支持fsnotify)

	// 构建相关参数
	BuildCmd   string            // 构建命令
	BuildArgs  []string          // 构建参数
	AppArgs    []string          // 应用参数
	Env        map[string]string // 环境变量

	// 显示相关参数
	Verbose    bool              // 显示详细信息
	NoBanner   bool              // 不显示欢迎信息
	QuietBuild bool              // 不显示构建输出
	QuietApp   bool              // 不显示应用输出
	Port       int               // 应用启动端口
}

// ParseFlags 解析命令行参数
func ParseFlags() (*Flags, error) {
	flags := &Flags{
		Env: make(map[string]string),
	}

	// 显示帮助信息和版本信息
	help := flag.Bool("help", false, "Show help information")
	version := flag.Bool("version", false, "Show version information")

	// 项目相关参数
	flag.StringVar(&flags.EntryPoint, "entry", ".", "Application entry point (main package directory)")
	flag.StringVar(&flags.WorkDir, "work-dir", "", "Working directory")
	flag.StringVar(&flags.TempDir, "temp-dir", "tmp", "Temporary directory")
	flag.StringVar(&flags.TemplatePattern, "template", "", "Template file pattern (e.g. './templates/*.html')")

	// 监控相关参数
	watchPaths := flag.String("watch", "./", "Paths to monitor (comma separated)")
	ignorePaths := flag.String("ignore", ".git,node_modules,tmp,vendor", "Paths to ignore (comma separated)")
	flag.IntVar(&flags.RestartDelay, "delay", 500, "Restart delay (milliseconds)")
	flag.BoolVar(&flags.LiveReload, "livereload", false, "Enable browser auto-refresh on template changes")
	pollIntervalSec := flag.Int("poll", 0, "Poll interval in seconds (0 = use filesystem events)")

	// 构建相关参数
	flag.StringVar(&flags.BuildCmd, "build-cmd", "go", "Build command")
	buildArgs := flag.String("build-args", "build -o ./tmp/app", "Build arguments (space separated)")
	appArgs := flag.String("app-args", "", "Application arguments (space separated)")
	env := flag.String("env", "", "Environment variables (format: KEY=VALUE,KEY2=VALUE2)")

	// 端口设置
	flag.IntVar(&flags.Port, "port", 8080, "Port to run the application (will be passed as -port flag)")

	// 显示相关参数
	flag.BoolVar(&flags.Verbose, "verbose", false, "Show verbose information")
	flag.BoolVar(&flags.NoBanner, "no-banner", false, "Don't show welcome banner")
	flag.BoolVar(&flags.QuietBuild, "quiet-build", false, "Don't show build output")
	flag.BoolVar(&flags.QuietApp, "quiet-app", false, "Don't show application output")

	// 解析命令行参数
	flag.Parse()

	// 显示版本信息
	if *version {
		fmt.Fprintln(os.Stdout, "WebFrame DevServer v1.0.0")
		os.Exit(0)
	}

	// 显示帮助信息并退出
	if *help {
		printUsageGuide()
		os.Exit(0)
	}

	// 处理复杂参数
	flags.WatchPaths = splitAndTrim(*watchPaths)
	flags.IgnorePaths = splitAndTrim(*ignorePaths)
	flags.BuildArgs = splitBySpace(*buildArgs)
	flags.AppArgs = splitBySpace(*appArgs)

	// 处理轮询间隔
	flags.PollInterval = time.Duration(*pollIntervalSec) * time.Second

	// 添加端口参数（如果没有在app-args中明确指定）
	portSpecified := false
	for _, arg := range flags.AppArgs {
		if arg == "-port" || strings.HasPrefix(arg, "-port=") {
			portSpecified = true
			break
		}
	}

	if !portSpecified && flags.Port > 0 {
		flags.AppArgs = append(flags.AppArgs, "-port", fmt.Sprintf("%d", flags.Port))
	}

	// 处理环境变量
	if *env != "" {
		envPairs := splitAndTrim(*env)
		for _, pair := range envPairs {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				flags.Env[parts[0]] = parts[1]
			}
		}
	}

	return flags, nil
}

// splitAndTrim 分割字符串并移除空白
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// splitBySpace 按空格分割字符串
func splitBySpace(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Fields(s)
}

// printUsageGuide 打印详细的使用指南
func printUsageGuide() {
	fmt.Println(`WebFrame DevServer - Hot Reload Development Tool

Usage: devserver [options]

Basic Examples:
  devserver                                    # Run with default settings
  devserver -entry ./cmd/main.go              # Specify entry point
  devserver -template ./templates/*.html       # Enable template hot reload
  devserver -port 3000                         # Run on port 3000

Common Options:`)

	flag.PrintDefaults()

	fmt.Println(`
Advanced Examples:
  devserver -entry ./cmd/main.go -watch ./cmd,./pkg -ignore .git,tmp
  devserver -build-cmd "go" -build-args "build -race -o ./tmp/app"
  devserver -env "DEBUG=true,ENV=development" -verbose
  
For more information, visit: https://github.com/fyerfyer/fyer-webframe`)
}
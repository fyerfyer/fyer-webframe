package hotreload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// HotReloadServer 负责应用程序的热重载功能
type HotReloadServer struct {
	config       *Config        // 热重载配置
	watcher      *Watcher       // 文件系统监视器
	cmd          *exec.Cmd      // 当前运行的应用程序进程
	manager      *Manager       // 关联的热重载管理器
	stdout       io.Writer      // 标准输出
	stderr       io.Writer      // 错误输出
	buildMutex   sync.Mutex     // 构建和重启互斥锁
	running      bool           // 服务器是否正在运行
	stopping     bool           // 服务器是否正在停止
	lastBuild    time.Time      // 最后一次构建时间
	lastStart    time.Time      // 最后一次启动时间
	processMutex sync.Mutex     // 进程操作互斥锁
	buildError   error          // 最后一次构建错误
	startError   error          // 最后一次启动错误
	done         chan struct{}  // 服务器停止信号
	restarting   bool           // 是否正在重启中
	tempBinPath  string         // 临时可执行文件路径

	testBuildAndRunFunc func() error // 仅测试使用
}

// ServerOption 定义服务器配置选项
type ServerOption func(*HotReloadServer)



// NewHotReloadServer 创建一个新的热重载服务器实例
func NewHotReloadServer(config *Config, manager *Manager, opts ...ServerOption) (*HotReloadServer, error) {
	watcher, err := NewWatcher(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	// 确保临时目录存在
	if config.TempDir == "" {
		config.TempDir = "tmp"
	}
	if err := os.MkdirAll(config.TempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temporary dictionary: %w", err)
	}

	// 构建临时二进制文件路径
	tempBinName := "app"
	if runtime.GOOS == "windows" {
		tempBinName += ".exe"
	}
	tempBinPath := filepath.Join(config.TempDir, tempBinName)

	// 初始化服务器
	server := &HotReloadServer{
		config:      config,
		watcher:     watcher,
		manager:     manager,
		stdout:      os.Stdout,
		stderr:      os.Stderr,
		done:        make(chan struct{}),
		tempBinPath: tempBinPath,
	}

	// 应用选项
	for _, opt := range opts {
		opt(server)
	}

	// 设置文件变更回调
	watcher.AddEventCallback(server.handleFileChange)

	return server, nil
}

// Start 启动热重载服务器
func (s *HotReloadServer) Start() error {
	s.buildMutex.Lock()
	defer s.buildMutex.Unlock()

	if s.running {
		return errors.New("server is under running")
	}

	s.running = true
	s.stopping = false

	// 显示欢迎信息
	if s.config.ShowBanner {
		fmt.Fprintln(s.stdout, "🔄 starting hotreload server...")
		fmt.Fprintf(s.stdout, "📁 monitoring path: %s\n", strings.Join(s.config.WatchPaths, ", "))
		fmt.Fprintf(s.stdout, "⏱  restart delay: %v\n", s.config.RestartDelay)
		fmt.Fprintln(s.stdout, "👀 monitoring file changes...")
	}

	// 启动文件监视器
	if err := s.watcher.Start(); err != nil {
		s.running = false
		return fmt.Errorf("failed to start file watcher: %w", err)
	}

	// 执行初始构建和启动
	if err := s.buildAndRun(); err != nil {
		fmt.Fprintf(s.stderr, "❌  initial build failed: %v\n", err)
		// 即使初始构建失败，我们也继续运行以等待文件修复
	}

	return nil
}

// Stop 停止热重载服务器
func (s *HotReloadServer) Stop() error {
	s.buildMutex.Lock()
	defer s.buildMutex.Unlock()

	if !s.running {
		return nil // 已经停止
	}

	s.stopping = true
	defer func() {
		s.stopping = false
	}()

	// 调用退出前钩子
	if s.config.BeforeExitHook != nil {
		if err := s.config.BeforeExitHook(); err != nil {
			fmt.Fprintf(s.stderr, "failed to exit hook: %v\n", err)
		}
	}

	// 停止应用程序
	if err := s.stopApp(); err != nil {
		fmt.Fprintf(s.stderr, "failed to stop application: %v\n", err)
	}

	// 停止文件监视器
	if err := s.watcher.Stop(); err != nil {
		fmt.Fprintf(s.stderr, "failed to stop file watcher: %v\n", err)
	}

	s.running = false

	// 通知已停止
	close(s.done)
	return nil
}

// Restart 重启应用程序
func (s *HotReloadServer) Restart() error {
	s.buildMutex.Lock()
	defer s.buildMutex.Unlock()

	if !s.running {
		return errors.New("server is not running")
	}

	if s.restarting {
		return nil // 已经在重启中
	}

	s.restarting = true
	defer func() {
		s.restarting = false
	}()

	fmt.Fprintln(s.stdout, "🔄 restarting application...")
	return s.buildAndRun()
}

// buildAndRun 构建并运行应用程序
func (s *HotReloadServer) buildAndRun() error {
	if s.testBuildAndRunFunc != nil {
		return s.testBuildAndRunFunc()
	}

	// 首先停止现有应用
	if err := s.stopApp(); err != nil {
		fmt.Fprintf(s.stderr, "failed to stop application: %v\n", err)
		// 继续执行，尝试强制重启
	}

	// 构建应用
	if err := s.buildApp(); err != nil {
		s.buildError = err
		return fmt.Errorf("failed to build application: %w", err)
	}

	// 构建成功后运行
	if err := s.runApp(); err != nil {
		s.startError = err
		return fmt.Errorf("failed to start application: %w", err)
	}

	return nil
}

// buildApp 构建应用程序
func (s *HotReloadServer) buildApp() error {
	fmt.Fprintln(s.stdout, "🔨 start building...")
	startTime := time.Now()

	// 执行构建前钩子
	if s.config.BeforeBuildHook != nil {
		if err := s.config.BeforeBuildHook(); err != nil {
			return fmt.Errorf("before build hook error: %w", err)
		}
	}

	// 准备构建命令
	buildCmd := s.config.BuildCommand
	buildArgs := make([]string, len(s.config.BuildArgs))
	copy(buildArgs, s.config.BuildArgs)

	// 如果没有设置输出文件参数，添加默认输出参数
	hasOutputFlag := false
	for i, arg := range buildArgs {
		if arg == "-o" && i+1 < len(buildArgs) {
			hasOutputFlag = true
			// 确保输出路径存在
			outputDir := filepath.Dir(buildArgs[i+1])
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return fmt.Errorf("failed to create dictionary %s: %w", outputDir, err)
			}
			break
		}
	}

	if !hasOutputFlag && buildCmd == "go" {
		// 添加默认输出路径
		buildArgs = append(buildArgs, "-o", s.tempBinPath)
	}

	// 添加入口点
	if s.config.EntryPoint != "" && buildCmd == "go" {
		found := false
		for _, arg := range buildArgs {
			if !strings.HasPrefix(arg, "-") {
				found = true
				break
			}
		}
		if !found {
			buildArgs = append(buildArgs, s.config.EntryPoint)
		}
	}

	// 创建构建命令
	cmd := exec.Command(buildCmd, buildArgs...)

	// 设置环境变量
	env := os.Environ()
	for k, v := range s.config.Env {
		env = append(env, k+"="+v)
	}
	cmd.Env = env

	// 设置工作目录
	cmd.Dir = filepath.Dir(s.config.EntryPoint)

	// 配置输出
	var buildOutput io.Writer
	if s.config.ShowBuildOutput {
		buildOutput = s.stdout
	} else {
		buildOutput = io.Discard
	}
	cmd.Stdout = buildOutput
	cmd.Stderr = buildOutput

	// 执行构建
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute build command: %w", err)
	}

	// 执行构建后钩子
	if s.config.AfterBuildHook != nil {
		if err := s.config.AfterBuildHook(); err != nil {
			return fmt.Errorf("after build hook error: %w", err)
		}
	}

	s.lastBuild = time.Now()
	duration := time.Since(startTime)
	fmt.Fprintf(s.stdout, "✅  build successfully in: %v\n", duration)

	return nil
}

// runApp 运行应用程序
func (s *HotReloadServer) runApp() error {
	s.processMutex.Lock()
	defer s.processMutex.Unlock()

	// 检查是否已有应用在运行
	if s.cmd != nil && s.cmd.Process != nil {
		return fmt.Errorf("application is already running")
	}

	fmt.Fprintln(s.stdout, "🚀 starting application...")

	// 执行启动前钩子
	if s.config.BeforeStartHook != nil {
		if err := s.config.BeforeStartHook(); err != nil {
			return fmt.Errorf("before start hook error: %w", err)
		}
	}

	// 准备应用命令
	appCmd := s.tempBinPath
	appArgs := make([]string, len(s.config.AppArgs))
	copy(appArgs, s.config.AppArgs)

	// 创建命令
	cmd := exec.Command(appCmd, appArgs...)

	// 设置环境变量
	env := os.Environ()
	for k, v := range s.config.Env {
		env = append(env, k+"="+v)
	}
	cmd.Env = env

	// 设置工作目录
	cmd.Dir = filepath.Dir(s.config.EntryPoint)

	// 配置输出
	var appOutput io.Writer
	if s.config.ShowAppOutput {
		appOutput = s.stdout
	} else {
		appOutput = io.Discard
	}
	cmd.Stdout = appOutput
	cmd.Stderr = appOutput

	// 启动应用
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start application: %w", err)
	}

	s.cmd = cmd
	s.lastStart = time.Now()
	fmt.Fprintf(s.stdout, "✅ application started (PID: %d)\n", cmd.Process.Pid)

	// 在后台监控应用进程
	go func() {
		if err := cmd.Wait(); err != nil {
			if !s.stopping && !s.restarting {
				fmt.Fprintf(s.stderr, "❌ application exits with an exception: %v\n", err)
				// 如果不是因为我们停止或重启，自动重新构建运行
				_ = s.Restart()
			}
		}
	}()

	return nil
}

// stopApp 停止应用程序
func (s *HotReloadServer) stopApp() error {
	s.processMutex.Lock()
	defer s.processMutex.Unlock()

	if s.cmd == nil || s.cmd.Process == nil {
		return nil // 没有进程在运行
	}

	fmt.Fprintln(s.stdout, "🛑 stopping application...")

	// 创建取消上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 尝试优雅关闭
	done := make(chan error, 1)
	go func() {
		if runtime.GOOS == "windows" {
			// Windows上没有SIGTERM，直接杀死进程
			done <- s.cmd.Process.Kill()
		} else {
			// 发送SIGTERM信号
			if err := s.cmd.Process.Signal(os.Interrupt); err != nil {
				done <- s.cmd.Process.Kill()
			} else {
				done <- nil
			}
		}
	}()

	// 等待进程退出或超时
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("failed to stop the application: %w", err)
		}
	case <-ctx.Done():
		// 超时，强制杀死进程
		if err := s.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to forcefully stop the application: %w", err)
		}
	}

	s.cmd = nil
	return nil
}

// handleFileChange 处理文件变化事件
func (s *HotReloadServer) handleFileChange(event Event) error {
	// 忽略某些文件更改
	if !s.shouldHandleChange(event) {
		return nil
	}

	// 避免并发重建
	if s.isRebuilding() {
		return nil
	}

	fmt.Fprintf(s.stdout, "📝 detect file change: %s\n", event.Path)
	return s.Restart()
}

// shouldHandleChange 判断是否应处理文件变更
func (s *HotReloadServer) shouldHandleChange(event Event) bool {
	path := event.Path

	// 忽略临时文件和编辑器临时文件
	if strings.HasPrefix(filepath.Base(path), ".") ||
		strings.HasSuffix(path, "~") ||
		strings.HasSuffix(path, ".swp") ||
		strings.HasSuffix(path, ".swx") {
		return false
	}

	// 忽略非Go文件，除非配置了监控其他类型
	if !s.watcher.IsExtensionWatched(path) {
		return false
	}

	// 忽略配置的路径
	for _, ignorePath := range s.config.IgnorePaths {
		if strings.Contains(path, ignorePath) {
			return false
		}
	}

	return true
}

// isRebuilding 检查是否正在重建中
func (s *HotReloadServer) isRebuilding() bool {
	s.buildMutex.Lock()
	defer s.buildMutex.Unlock()
	return s.restarting
}

// Done 返回服务器已停止的channel
func (s *HotReloadServer) Done() <-chan struct{} {
	return s.done
}

// IsRunning 检查服务器是否正在运行
func (s *HotReloadServer) IsRunning() bool {
	s.buildMutex.Lock()
	defer s.buildMutex.Unlock()
	return s.running
}

// LastBuildTime 返回最后一次构建时间
func (s *HotReloadServer) LastBuildTime() time.Time {
	return s.lastBuild
}

// LastStartTime 返回最后一次启动时间
func (s *HotReloadServer) LastStartTime() time.Time {
	return s.lastStart
}

// BuildError 返回最后一次构建错误
func (s *HotReloadServer) BuildError() error {
	return s.buildError
}

// StartError 返回最后一次启动错误
func (s *HotReloadServer) StartError() error {
	return s.startError
}

// TemplateFile 返回热重载模板内容
func (s *HotReloadServer) TemplateFile(name string) (string, error) {
	if name == "banner" {
		path := filepath.Join("hotreload", "templates", "banner.tmpl")
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	return "", errors.New("template not found")
}
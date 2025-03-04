package hotreload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fyerfyer/fyer-webframe/web"
)

// Manager 热重载管理器，负责协调文件监控和应用重启
type Manager struct {
	config          *Config          // 热重载配置
	server          *HotReloadServer // 热重载服务器实例
	running         bool             // 是否正在运行
	done            chan struct{}    // 停止信号通道
	mu              sync.RWMutex     // 并发控制锁
	tplMonitor      *TemplateMonitor // 模板监控器
	stdout          io.Writer        // 标准输出
	stderr          io.Writer        // 错误输出
	httpServer      web.Server       // 框架服务器实例
	testRestartFunc func() error     // 仅测试使用
}

// TemplateMonitor 用于监控模板文件变化
type TemplateMonitor struct {
	engine       web.Template
	lastReloaded time.Time
	mu           sync.RWMutex
}

// ManagerOption 定义管理器选项函数
type ManagerOption func(*Manager)

// WithStdout 设置标准输出
func WithServerStdout(stdout io.Writer) ServerOption {
	return func(s *HotReloadServer) {
		s.stdout = stdout
	}
}

// WithStderr 设置错误输出
func WithServerStderr(stderr io.Writer) ServerOption {
	return func(s *HotReloadServer) {
		s.stderr = stderr
	}
}

// WithHTTPServer 设置HTTP服务器实例
func WithHTTPServer(server web.Server) ManagerOption {
	return func(m *Manager) {
		m.httpServer = server
	}
}

// NewManager 创建一个新的热重载管理器
func NewManager(config *Config, opts ...ManagerOption) (*Manager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	manager := &Manager{
		config: config,
		done:   make(chan struct{}),
		stdout: os.Stdout,
		stderr: os.Stderr,
	}

	// 应用选项
	for _, opt := range opts {
		opt(manager)
	}

	// 创建热重载服务器
	server, err := NewHotReloadServer(config, manager,
		WithServerStdout(manager.stdout),
		WithServerStderr(manager.stderr),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create hotreload server: %w", err)
	}
	manager.server = server

	// 设置模板监控器
	if manager.httpServer != nil {
		engine := getTemplateEngine(manager.httpServer)
		if engine != nil {
			manager.tplMonitor = &TemplateMonitor{
				engine:       engine,
				lastReloaded: time.Now(),
			}
		}
	}

	return manager, nil
}

// 获取模板引擎实例
func getTemplateEngine(server web.Server) web.Template {
	// 尝试从HTTP服务器获取模板引擎
	if httpServer, ok := server.(*web.HTTPServer); ok {
		// 假设HTTPServer有一个字段或方法可以访问模板引擎
		// 这里只是示例，可能需要修改
		return httpServer.GetTemplateEngine()
	}
	return nil
}

// Start 启动热重载管理器
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return errors.New("hotreload manager is already running")
	}

	// 启动热重载服务器
	if err := m.server.Start(); err != nil {
		return fmt.Errorf("failed to start hotreload server: %w", err)
	}

	m.running = true

	// 启动模板监控（如果有模板引擎）
	if m.tplMonitor != nil {
		go m.monitorTemplates()
	}

	fmt.Fprintln(m.stdout, "🔥 hotreload manager is under running...")
	return nil
}

// Stop 停止热重载管理器
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	// 停止热重载服务器
	if err := m.server.Stop(); err != nil {
		return fmt.Errorf("failed to stop hotreload server: %w", err)
	}

	m.running = false
	close(m.done)
	fmt.Fprintln(m.stdout, "🛑 hotreload manager is stopped")
	return nil
}

// Restart 重启应用
func (m *Manager) Restart() error {
	if m.testRestartFunc != nil {
		return m.testRestartFunc()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.running {
		return errors.New("hotreload manager is not running")
	}

	fmt.Fprintln(m.stdout, "🔄 trying restart application...")
	return m.server.Restart()
}

// IsRunning 检查管理器是否正在运行
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// Wait 等待管理器停止
func (m *Manager) Wait() {
	<-m.server.Done()
}

// WaitWithContext 在上下文取消之前等待管理器停止
func (m *Manager) WaitWithContext(ctx context.Context) error {
	select {
	case <-m.server.Done():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ReloadTemplates 重新加载模板
func (m *Manager) ReloadTemplates() error {
	if m.tplMonitor == nil || m.tplMonitor.engine == nil {
		return errors.New("haven't set template engine")
	}

	m.tplMonitor.mu.Lock()
	defer m.tplMonitor.mu.Unlock()

	fmt.Fprintln(m.stdout, "📄 reloading templates...")
	err := m.tplMonitor.engine.Reload()
	if err != nil {
		fmt.Fprintf(m.stderr, "❌ failed to reload templates: %v\n", err)
		return err
	}

	m.tplMonitor.lastReloaded = time.Now()
	fmt.Fprintln(m.stdout, "✅ template reload success")
	return nil
}

// monitorTemplates 监控模板文件变更
func (m *Manager) monitorTemplates() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 检查模板文件是否有更新
			if m.hasTemplateChanges() {
				if err := m.ReloadTemplates(); err != nil {
					log.Printf("failed to reload templates: %v", err)
				}
			}
		case <-m.done:
			return
		}
	}
}

// hasTemplateChanges 检测模板文件是否有变更
func (m *Manager) hasTemplateChanges() bool {
	// todo:这里可以实现更复杂的逻辑来检测模板文件变更
	// 例如检查最后修改时间，对比文件哈希等

	if m.tplMonitor == nil {
		return false
	}

	// 如果配置了模板文件路径，检查这些文件的修改时间
	if m.config.tplPattern != "" {
		matches, err := filepath.Glob(m.config.tplPattern)
		if err != nil {
			return false
		}

		m.tplMonitor.mu.RLock()
		lastReloaded := m.tplMonitor.lastReloaded
		m.tplMonitor.mu.RUnlock()

		for _, file := range matches {
			info, err := os.Stat(file)
			if err != nil {
				continue
			}
			if info.ModTime().After(lastReloaded) {
				return true
			}
		}
	}

	return false
}

// SetTemplatePattern 设置模板文件匹配模式
func (m *Manager) SetTemplatePattern(pattern string) {
	m.config.tplPattern = pattern
}

// HandleEvent 处理文件变更事件
func (m *Manager) HandleEvent(event Event) error {
	// 首先检查是否是被忽略的路径
	for _, ignorePath := range m.config.IgnorePaths {
		if strings.Contains(event.Path, ignorePath) {
			// 忽略此文件的变更
			return nil
		}
	}

	// 这个方法会被Watcher回调
	if event.Type == Write || event.Type == Create {
		// 检查是否是模板文件变更
		if m.isTemplateFile(event.Path) {
			return m.ReloadTemplates()
		}
		// 对于其他文件变更，触发应用重启
		return m.Restart()
	}
	return nil
}

// isTemplateFile 判断文件是否为模板文件
func (m *Manager) isTemplateFile(path string) bool {
	// 简单实现：检查文件扩展名
	ext := filepath.Ext(path)
	return ext == ".tmpl" || ext == ".html" || ext == ".gohtml"
}

// AddTemplateMonitor 添加对模板引擎的监控
func (m *Manager) AddTemplateMonitor(engine web.Template, pattern string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tplMonitor = &TemplateMonitor{
		engine:       engine,
		lastReloaded: time.Now(),
	}
	m.config.tplPattern = pattern

	if m.running {
		go m.monitorTemplates()
	}
}

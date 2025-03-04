package hotreload

import (
	"time"
)

// Config 热重载功能的配置选项
type Config struct {
	// 要监控的文件或目录路径模式
	WatchPaths []string

	// 要忽略的文件或目录路径模式
	IgnorePaths []string

	// 应用重启前的延迟时间，用于避免频繁重启
	RestartDelay time.Duration

	// 自定义构建命令，为空则使用默认go build
	BuildCommand string

	// 构建命令的参数
	BuildArgs []string

	// 应用程序入口文件（main包所在路径）
	EntryPoint string

	// 应用程序的启动参数
	AppArgs []string

	// 是否显示构建输出
	ShowBuildOutput bool

	// 是否显示应用输出
	ShowAppOutput bool

	// 是否在启动时显示热重载的欢迎信息
	ShowBanner bool

	// 环境变量设置
	Env map[string]string

	// 构建前要执行的钩子
	BeforeBuildHook func() error

	// 构建后要执行的钩子
	AfterBuildHook func() error

	// 启动前要执行的钩子
	BeforeStartHook func() error

	// 退出前要执行的钩子
	BeforeExitHook func() error

	// 临时文件目录
	TempDir string

	// 模板文件匹配模式，用于模板热重载
	tplPattern string
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		WatchPaths:      []string{"./"},
		IgnorePaths:     []string{".git", "node_modules", "tmp", "vendor"},
		RestartDelay:    time.Millisecond * 500,
		BuildCommand:    "go",
		BuildArgs:       []string{"build", "-o", "tmp/app"},
		EntryPoint:      ".",
		ShowBuildOutput: true,
		ShowAppOutput:   true,
		ShowBanner:      true,
		Env:             make(map[string]string),
		tplPattern:      "",
	}
}

// ConfigOption 定义配置选项函数
type ConfigOption func(*Config)

// NewConfig 创建新的热重载配置
func NewConfig(opts ...ConfigOption) *Config {
	cfg := DefaultConfig()

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// WithWatchPaths 设置要监控的路径
func WithWatchPaths(paths ...string) ConfigOption {
	return func(cfg *Config) {
		cfg.WatchPaths = paths
	}
}

// WithIgnorePaths 设置要忽略的路径
func WithIgnorePaths(paths ...string) ConfigOption {
	return func(cfg *Config) {
		cfg.IgnorePaths = paths
	}
}

// WithRestartDelay 设置重启延迟时间
func WithRestartDelay(delay time.Duration) ConfigOption {
	return func(cfg *Config) {
		cfg.RestartDelay = delay
	}
}

// WithBuildCommand 设置构建命令
func WithBuildCommand(cmd string, args ...string) ConfigOption {
	return func(cfg *Config) {
		cfg.BuildCommand = cmd
		cfg.BuildArgs = args
	}
}

// WithEntryPoint 设置应用入口点
func WithEntryPoint(entryPoint string) ConfigOption {
	return func(cfg *Config) {
		cfg.EntryPoint = entryPoint
	}
}

// WithAppArgs 设置应用启动参数
func WithAppArgs(args ...string) ConfigOption {
	return func(cfg *Config) {
		cfg.AppArgs = args
	}
}

// WithEnvironment 设置环境变量
func WithEnvironment(env map[string]string) ConfigOption {
	return func(cfg *Config) {
		for k, v := range env {
			cfg.Env[k] = v
		}
	}
}

// WithShowBuildOutput 设置是否显示构建输出
func WithShowBuildOutput(show bool) ConfigOption {
	return func(cfg *Config) {
		cfg.ShowBuildOutput = show
	}
}

// WithShowAppOutput 设置是否显示应用输出
func WithShowAppOutput(show bool) ConfigOption {
	return func(cfg *Config) {
		cfg.ShowAppOutput = show
	}
}

// WithShowBanner 设置是否显示欢迎信息
func WithShowBanner(show bool) ConfigOption {
	return func(cfg *Config) {
		cfg.ShowBanner = show
	}
}

// WithBeforeBuildHook 设置构建前钩子
func WithBeforeBuildHook(hook func() error) ConfigOption {
	return func(cfg *Config) {
		cfg.BeforeBuildHook = hook
	}
}

// WithAfterBuildHook 设置构建后钩子
func WithAfterBuildHook(hook func() error) ConfigOption {
	return func(cfg *Config) {
		cfg.AfterBuildHook = hook
	}
}

// WithBeforeStartHook 设置启动前钩子
func WithBeforeStartHook(hook func() error) ConfigOption {
	return func(cfg *Config) {
		cfg.BeforeStartHook = hook
	}
}

// WithBeforeExitHook 设置退出前钩子
func WithBeforeExitHook(hook func() error) ConfigOption {
	return func(cfg *Config) {
		cfg.BeforeExitHook = hook
	}
}

// WithTempDir 设置临时目录
func WithTempDir(dir string) ConfigOption {
	return func(cfg *Config) {
		cfg.TempDir = dir
	}
}

// WithTemplatePattern 设置模板文件匹配模式
func WithTemplatePattern(pattern string) ConfigOption {
	return func(cfg *Config) {
		cfg.tplPattern = pattern
	}
}

// GetTemplatePattern 获取模板文件匹配模式
func (c *Config) GetTemplatePattern() string {
	return c.tplPattern
}
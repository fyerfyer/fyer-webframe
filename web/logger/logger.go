package logger

import (
	"context"
	"io"
	"time"
)

// LogLevel 定义日志级别
type LogLevel int

const (
	// DebugLevel 调试级别
	DebugLevel LogLevel = iota
	// InfoLevel 信息级别
	InfoLevel
	// WarnLevel 警告级别
	WarnLevel
	// ErrorLevel 错误级别
	ErrorLevel
	// FatalLevel 致命错误级别
	FatalLevel
)

// Field 表示结构化日志的字段
type Field struct {
	Key   string
	Value interface{}
}

// Logger 定义日志接口
type Logger interface {
	// Debug 输出调试级别日志
	Debug(msg string, fields ...Field)
	// Info 输出信息级别日志
	Info(msg string, fields ...Field)
	// Warn 输出警告级别日志
	Warn(msg string, fields ...Field)
	// Error 输出错误级别日志
	Error(msg string, fields ...Field)
	// Fatal 输出致命错误级别日志
	Fatal(msg string, fields ...Field)

	// WithContext 添加上下文到日志
	WithContext(ctx context.Context) Logger
	// WithField 添加单个字段
	WithField(key string, value interface{}) Logger
	// WithFields 添加多个字段
	WithFields(fields ...Field) Logger

	// SetLevel 设置日志级别
	SetLevel(level LogLevel)
	// SetOutput 设置日志输出目标
	SetOutput(w io.Writer)
}

// Option 日志配置选项函数
type Option func(*LogConfig)

// LogConfig 日志配置
type LogConfig struct {
	Level      LogLevel
	Output     io.Writer
	TimeFormat string
	Async      bool
	BufferSize int
}

// WithLevel 设置日志级别选项
func WithLevel(level LogLevel) Option {
	return func(cfg *LogConfig) {
		cfg.Level = level
	}
}

// WithOutput 设置日志输出选项
func WithOutput(w io.Writer) Option {
	return func(cfg *LogConfig) {
		cfg.Output = w
	}
}

// WithTimeFormat 设置时间格式选项
func WithTimeFormat(format string) Option {
	return func(cfg *LogConfig) {
		cfg.TimeFormat = format
	}
}

// WithAsync 启用异步日志选项
func WithAsync(bufferSize int) Option {
	return func(cfg *LogConfig) {
		cfg.Async = true
		if bufferSize > 0 {
			cfg.BufferSize = bufferSize
		}
	}
}

// defaultConfig 返回默认日志配置
func defaultConfig() *LogConfig {
	return &LogConfig{
		Level:      InfoLevel,
		TimeFormat: time.RFC3339,
		Async:      false,
		BufferSize: 1024,
	}
}

// 全局默认日志实例
var defaultLogger Logger

// 初始化默认日志实例
func init() {
	defaultLogger = New()
}

// New 创建一个标准日志实例
func New(opts ...Option) Logger {
	// 这个函数的实现会在 zerolog.go 中
	// 这里只是接口定义和工厂函数声明
	return nil
}

// NewAsync 创建一个异步日志实例
func NewAsync(bufferSize int, opts ...Option) Logger {
	options := append(opts, WithAsync(bufferSize))
	return New(options...)
}

// 工具函数：创建各种类型的字段

// String 创建字符串类型的日志字段
func String(key string, value string) Field {
	return Field{Key: key, Value: value}
}

// Int 创建整数类型的日志字段
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 创建64位整数类型的日志字段
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Float64 创建浮点数类型的日志字段
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Bool 创建布尔类型的日志字段
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Time 创建时间类型的日志字段
func Time(key string, value time.Time) Field {
	return Field{Key: key, Value: value}
}

// FieldError Error 创建错误类型的日志字段
func FieldError(err error) Field {
	return Field{Key: "error", Value: err}
}

// Interface 创建任意接口类型的日志字段
func Interface(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// 全局日志函数，使用默认日志实例

// Debug 输出调试级别日志
func Debug(msg string, fields ...Field) {
	defaultLogger.Debug(msg, fields...)
}

// Info 输出信息级别日志
func Info(msg string, fields ...Field) {
	defaultLogger.Info(msg, fields...)
}

// Warn 输出警告级别日志
func Warn(msg string, fields ...Field) {
	defaultLogger.Warn(msg, fields...)
}

// Error 输出错误级别日志
func Error(msg string, fields ...Field) {
	defaultLogger.Error(msg, fields...)
}

// Fatal 输出致命错误级别日志
func Fatal(msg string, fields ...Field) {
	defaultLogger.Fatal(msg, fields...)
}

// SetLevel 设置默认日志实例的级别
func SetLevel(level LogLevel) {
	defaultLogger.SetLevel(level)
}

// SetOutput 设置默认日志实例的输出
func SetOutput(w io.Writer) {
	defaultLogger.SetOutput(w)
}

// GetDefaultLogger 获取默认日志实例
func GetDefaultLogger() Logger {
	return defaultLogger
}

// SetDefaultLogger 设置默认日志实例
func SetDefaultLogger(logger Logger) {
	defaultLogger = logger
}
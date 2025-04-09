package logger

import (
	"context"
	"github.com/rs/zerolog"
	"io"
	"os"
	"sync"
	"time"
)

// zerologLogger 使用 zerolog 实现的日志记录器
type zerologLogger struct {
	zlog  zerolog.Logger
	level LogLevel
	async bool
	mu    sync.Mutex
	ch    chan *logEvent
	wg    sync.WaitGroup
}

// logEvent 表示一个异步日志事件
type logEvent struct {
	level   LogLevel
	message string
	fields  []Field
}

// init 在包初始化时设置默认日志记录器
func init() {
	// 初始化 zerolog 环境
	zerolog.TimeFieldFormat = time.RFC3339

	// 创建并设置默认日志记录器
	zlog := zerolog.New(os.Stderr).With().Timestamp().Logger()
	defaultLogger = &zerologLogger{
		zlog:  zlog,
		level: InfoLevel,
		async: false,
	}
}

// NewLogger 创建一个新的 zerolog 日志记录器
func NewLogger(opts ...Option) Logger {
	// 使用默认配置
	cfg := defaultConfig()

	// 应用选项
	for _, opt := range opts {
		opt(cfg)
	}

	// 设置输出
	var output io.Writer
	if cfg.Output != nil {
		output = cfg.Output
	} else {
		output = os.Stderr
	}

	// 创建 zerolog 日志记录器
	zlog := zerolog.New(output).With().Timestamp().Logger()

	// 根据配置设置日志级别
	setZerologLevel(&zlog, cfg.Level)

	logger := &zerologLogger{
		zlog:  zlog,
		level: cfg.Level,
		async: cfg.Async,
	}

	// 如果启用异步，初始化通道和工作协程
	if cfg.Async {
		logger.ch = make(chan *logEvent, cfg.BufferSize)
		logger.startWorker()
	}

	return logger
}

// startWorker 启动异步日志工作协程
func (l *zerologLogger) startWorker() {
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		for evt := range l.ch {
			// 根据级别和字段处理日志事件
			l.processLogEvent(evt)
		}
	}()
}

// Close 关闭日志记录器，等待所有异步日志写入完成
func (l *zerologLogger) Close() {
	if l.async && l.ch != nil {
		close(l.ch)
		l.wg.Wait()
	}
}

// processLogEvent 处理异步日志事件
func (l *zerologLogger) processLogEvent(evt *logEvent) {
	var event *zerolog.Event

	// 根据日志级别创建事件
	switch evt.level {
	case DebugLevel:
		event = l.zlog.Debug()
	case InfoLevel:
		event = l.zlog.Info()
	case WarnLevel:
		event = l.zlog.Warn()
	case ErrorLevel:
		event = l.zlog.Error()
	case FatalLevel:
		event = l.zlog.Fatal()
	default:
		event = l.zlog.Info()
	}

	// 添加字段
	for _, field := range evt.fields {
		addFieldToEvent(event, field)
	}

	// 写入消息
	event.Msg(evt.message)
}

// Debug 输出调试级别日志
func (l *zerologLogger) Debug(msg string, fields ...Field) {
	if l.level > DebugLevel {
		return
	}

	if l.async {
		l.sendAsync(DebugLevel, msg, fields)
		return
	}

	event := l.zlog.Debug()
	for _, field := range fields {
		addFieldToEvent(event, field)
	}
	event.Msg(msg)
}

// Info 输出信息级别日志
func (l *zerologLogger) Info(msg string, fields ...Field) {
	if l.level > InfoLevel {
		return
	}

	if l.async {
		l.sendAsync(InfoLevel, msg, fields)
		return
	}

	event := l.zlog.Info()
	for _, field := range fields {
		addFieldToEvent(event, field)
	}
	event.Msg(msg)
}

// Warn 输出警告级别日志
func (l *zerologLogger) Warn(msg string, fields ...Field) {
	if l.level > WarnLevel {
		return
	}

	if l.async {
		l.sendAsync(WarnLevel, msg, fields)
		return
	}

	event := l.zlog.Warn()
	for _, field := range fields {
		addFieldToEvent(event, field)
	}
	event.Msg(msg)
}

// Error 输出错误级别日志
func (l *zerologLogger) Error(msg string, fields ...Field) {
	if l.level > ErrorLevel {
		return
	}

	if l.async {
		l.sendAsync(ErrorLevel, msg, fields)
		return
	}

	event := l.zlog.Error()
	for _, field := range fields {
		addFieldToEvent(event, field)
	}
	event.Msg(msg)
}

// Fatal 输出致命错误级别日志
func (l *zerologLogger) Fatal(msg string, fields ...Field) {
	if l.level > FatalLevel {
		return
	}

	if l.async {
		l.sendAsync(FatalLevel, msg, fields)
		return
	}

	event := l.zlog.Fatal()
	for _, field := range fields {
		addFieldToEvent(event, field)
	}
	event.Msg(msg)
}

// sendAsync 发送异步日志事件
func (l *zerologLogger) sendAsync(level LogLevel, msg string, fields []Field) {
	// 创建一个新的日志事件
	evt := &logEvent{
		level:   level,
		message: msg,
		fields:  make([]Field, len(fields)),
	}

	// 复制字段以避免并发问题
	copy(evt.fields, fields)

	// 发送到通道
	select {
	case l.ch <- evt:
		// 成功发送
	default:
		// 通道已满，记录警告
		l.zlog.Warn().Msg("Async log channel is full, dropping log message")
	}
}

// WithContext 添加上下文到日志
func (l *zerologLogger) WithContext(ctx context.Context) Logger {
	if ctx == nil {
		return l
	}

	newLogger := &zerologLogger{
		zlog:  l.zlog.With().Logger(),
		level: l.level,
		async: l.async,
		ch:    l.ch,
	}

	// 从上下文中获取请求ID等信息
	if reqID, ok := ctx.Value("request_id").(string); ok {
		newLogger.zlog = newLogger.zlog.With().Str("request_id", reqID).Logger()
	}

	return newLogger
}

// WithField 添加单个字段
func (l *zerologLogger) WithField(key string, value interface{}) Logger {
	newLogger := &zerologLogger{
		zlog:  l.zlog.With().Interface(key, value).Logger(),
		level: l.level,
		async: l.async,
		ch:    l.ch,
	}
	return newLogger
}

// WithFields 添加多个字段
func (l *zerologLogger) WithFields(fields ...Field) Logger {
	ctx := l.zlog.With()
	
	// 逐个添加字段并更新 ctx
	for _, field := range fields {
		ctx = addFieldToContext(ctx, field)
	}

	newLogger := &zerologLogger{
		zlog:  ctx.Logger(),
		level: l.level,
		async: l.async,
		ch:    l.ch,
	}
	return newLogger
}

// SetLevel 设置日志级别
func (l *zerologLogger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
	setZerologLevel(&l.zlog, level)
}

// SetOutput 设置日志输出目标
func (l *zerologLogger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.zlog = l.zlog.Output(w)
}

// setZerologLevel 将内部日志级别转换为 zerolog 级别
func setZerologLevel(zlog *zerolog.Logger, level LogLevel) {
	var zerologLevel zerolog.Level
	switch level {
	case DebugLevel:
		zerologLevel = zerolog.DebugLevel
	case InfoLevel:
		zerologLevel = zerolog.InfoLevel
	case WarnLevel:
		zerologLevel = zerolog.WarnLevel
	case ErrorLevel:
		zerologLevel = zerolog.ErrorLevel
	case FatalLevel:
		zerologLevel = zerolog.FatalLevel
	default:
		zerologLevel = zerolog.InfoLevel
	}

	*zlog = zlog.Level(zerologLevel)
}

// addFieldToEvent 将字段添加到日志事件
func addFieldToEvent(event *zerolog.Event, field Field) {
	switch v := field.Value.(type) {
	case string:
		event.Str(field.Key, v)
	case int:
		event.Int(field.Key, v)
	case int64:
		event.Int64(field.Key, v)
	case float64:
		event.Float64(field.Key, v)
	case bool:
		event.Bool(field.Key, v)
	case time.Time:
		event.Time(field.Key, v)
	case error:
		event.Err(v)
	default:
		event.Interface(field.Key, v)
	}
}

// 修改 addFieldToContext 函数的参数类型，接受值而不是指针
func addFieldToContext(ctx zerolog.Context, field Field) zerolog.Context {
	switch v := field.Value.(type) {
	case string:
		return ctx.Str(field.Key, v)
	case int:
		return ctx.Int(field.Key, v)
	case int64:
		return ctx.Int64(field.Key, v)
	case float64:
		return ctx.Float64(field.Key, v)
	case bool:
		return ctx.Bool(field.Key, v)
	case time.Time:
		return ctx.Time(field.Key, v)
	case error:
		return ctx.Err(v)
	default:
		return ctx.Interface(field.Key, v)
	}
}
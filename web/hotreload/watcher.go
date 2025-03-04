package hotreload

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// EventType 定义文件系统事件类型
type EventType int

const (
	Create EventType = iota
	Write
	Remove
	Rename
	Chmod
)

// Event 表示文件系统事件
type Event struct {
	Type EventType
	Path string
}

// EventCallback 是文件变更时的回调函数类型
type EventCallback func(event Event) error

// Watcher 负责监视文件系统变更
type Watcher struct {
	fsWatcher      *fsnotify.Watcher
	watchPaths     []string
	ignorePaths    []string
	eventCallbacks []EventCallback
	restartDelay   time.Duration
	lastEventTime  time.Time
	debounceTimer  *time.Timer
	mu             sync.Mutex
	started        bool
	stopping       bool
}

// NewWatcher 创建一个新的监视器实例
func NewWatcher(config *Config) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file system watcher: %w", err)
	}

	watcher := &Watcher{
		fsWatcher:     fsWatcher,
		watchPaths:    config.WatchPaths,
		ignorePaths:   config.IgnorePaths,
		restartDelay:  config.RestartDelay,
		lastEventTime: time.Now(),
	}

	return watcher, nil
}

// Start 启动监视器
func (w *Watcher) Start() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.started {
		return errors.New("watcher started")
	}

	// 添加所有要监视的路径
	for _, path := range w.watchPaths {
		if err := w.addWatchPath(path); err != nil {
			return err
		}
	}

	w.started = true
	w.stopping = false

	// 启动一个 goroutine 来处理事件
	go w.watchEvents()

	return nil
}

// Stop 停止监视器
func (w *Watcher) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.started {
		return nil
	}

	w.stopping = true
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
	}

	err := w.fsWatcher.Close()
	if err != nil {
		return fmt.Errorf("failed to close file system watcher: %w", err)
	}

	w.started = false
	return nil
}

// AddEventCallback 添加事件回调函数
func (w *Watcher) AddEventCallback(callback EventCallback) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.eventCallbacks = append(w.eventCallbacks, callback)
}

// 内部方法：添加要监视的路径
func (w *Watcher) addWatchPath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("path resolution failed %s: %w", path, err)
	}

	// 检查路径是否存在
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("path check failed %s: %w", absPath, err)
	}

	// 如果是目录，则递归添加
	if info.IsDir() {
		if err := w.fsWatcher.Add(absPath); err != nil {
			return fmt.Errorf("failed to add monitor path %s: %w", absPath, err)
		}

		// 递归添加子目录
		return filepath.Walk(absPath, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				// 检查目录是否应该被忽略
				if w.shouldIgnore(filePath) {
					return filepath.SkipDir
				}

				if err := w.fsWatcher.Add(filePath); err != nil {
					log.Printf("warning: cannot monitor path %s: %v", filePath, err)
				}
			}
			return nil
		})
	}

	// 如果是文件，直接添加
	return w.fsWatcher.Add(absPath)
}

// 检查路径是否应该被忽略
func (w *Watcher) shouldIgnore(path string) bool {
	for _, ignorePath := range w.ignorePaths {
		if strings.Contains(path, ignorePath) {
			return true
		}
	}
	return false
}

// 监视和处理文件系统事件
func (w *Watcher) watchEvents() {
	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				// 通道已关闭，退出
				return
			}

			// 检查是否应该忽略此路径
			if w.shouldIgnore(event.Name) {
				continue
			}

			w.handleEvent(event)

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				// 错误通道已关闭，退出
				return
			}
			log.Printf("watcher error: %v", err)
		}
	}
}

// 处理文件系统事件
func (w *Watcher) handleEvent(event fsnotify.Event) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 如果正在停止，则忽略事件
	if w.stopping {
		return
	}

	// 如果有延迟计时器在运行，则先停止它
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
	}

	// 根据事件类型创建我们的事件
	var eventType EventType
	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		eventType = Create
		// 如果新创建的是目录，需要添加监视
		if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
			w.addWatchPath(event.Name)
		}
	case event.Op&fsnotify.Write == fsnotify.Write:
		eventType = Write
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		eventType = Remove
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		eventType = Rename
	case event.Op&fsnotify.Chmod == fsnotify.Chmod:
		eventType = Chmod
		// 通常不需要为权限更改触发重启
		return
	}

	w.lastEventTime = time.Now()

	// 创建防抖动计时器
	w.debounceTimer = time.AfterFunc(w.restartDelay, func() {
		w.triggerCallbacks(Event{
			Type: eventType,
			Path: event.Name,
		})
	})
}

// 触发所有回调
func (w *Watcher) triggerCallbacks(event Event) {
	w.mu.Lock()
	callbacks := append([]EventCallback{}, w.eventCallbacks...)
	w.mu.Unlock()

	for _, callback := range callbacks {
		if err := callback(event); err != nil {
			log.Printf("event callback error: %v", err)
		}
	}
}

// ReloadWatchPaths 重新加载要监视的路径
func (w *Watcher) ReloadWatchPaths() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 如果监视器未启动，直接返回
	if !w.started {
		return nil
	}

	// 先移除所有现有的监视路径
	for _, path := range w.watchPaths {
		absPath, err := filepath.Abs(path)
		if err == nil {
			w.fsWatcher.Remove(absPath)
		}
	}

	// 重新添加所有路径
	for _, path := range w.watchPaths {
		if err := w.addWatchPath(path); err != nil {
			return err
		}
	}

	return nil
}

// IsExtensionWatched 检查文件扩展名是否应该被监控
func (w *Watcher) IsExtensionWatched(filename string) bool {
	// 默认监控所有 .go 文件
	ext := filepath.Ext(filename)
	return ext == ".go"
}

// IsStarted 返回监视器是否已启动
func (w *Watcher) IsStarted() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.started
}

// SetWatchPaths 设置要监视的路径
func (w *Watcher) SetWatchPaths(paths []string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.watchPaths = paths
}

// SetIgnorePaths 设置要忽略的路径
func (w *Watcher) SetIgnorePaths(paths []string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.ignorePaths = paths
}

// SetRestartDelay 设置重启延迟时间
func (w *Watcher) SetRestartDelay(delay time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.restartDelay = delay
}
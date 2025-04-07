package orm

import "log"

// 控制是否输出缓存相关的调试日志
var debugCacheLog = false

// EnableCacheDebugLog 启用缓存调试日志
func EnableCacheDebugLog() {
	debugCacheLog = true
}

// DisableCacheDebugLog 禁用缓存调试日志
func DisableCacheDebugLog() {
	debugCacheLog = false
}

// debugLog 条件性输出调试日志
func debugLog(format string, args ...interface{}) {
	if debugCacheLog {
		// 使用标准库 log 包或者项目中使用的日志库
		log.Printf(format, args...)
	}
}

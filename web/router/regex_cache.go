package router

import (
	"regexp"
	"sync"
)

// RegexCache 提供正则表达式模式缓存功能，避免重复编译
type RegexCache struct {
	cache map[string]*regexp.Regexp
	mu    sync.RWMutex
}

// NewRegexCache 创建一个新的正则表达式缓存
func NewRegexCache() *RegexCache {
	return &RegexCache{
		cache: make(map[string]*regexp.Regexp),
	}
}

// Get 获取一个正则表达式，如果缓存中不存在则编译并缓存它
// 注意：模式会按原样编译，如果需要添加 "^" 和 "$" 边界，请在传入模式时包含
func (rc *RegexCache) Get(pattern string) (*regexp.Regexp, error) {
	// 首先尝试从缓存中读取，这里使用读锁
	rc.mu.RLock()
	cached, ok := rc.cache[pattern]
	rc.mu.RUnlock()

	if ok {
		return cached, nil
	}

	// 如果缓存中没有，则编译并存储，这里需要写锁
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// 双重检查，可能在获取写锁的过程中已经被其他协程编译并缓存
	if cached, ok = rc.cache[pattern]; ok {
		return cached, nil
	}

	// 编译正则表达式
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	// 存储到缓存
	rc.cache[pattern] = compiled
	return compiled, nil
}

// MustGet 获取一个正则表达式，如果编译失败则会panic
func (rc *RegexCache) MustGet(pattern string) *regexp.Regexp {
	regex, err := rc.Get(pattern)
	if err != nil {
		panic("Failed to compile regex: " + err.Error())
	}
	return regex
}

// Clear 清空缓存
func (rc *RegexCache) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.cache = make(map[string]*regexp.Regexp)
}

// Size 返回缓存的大小
func (rc *RegexCache) Size() int {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return len(rc.cache)
}
package orm

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// MemoryCache 是一个简单的内存缓存实现
type MemoryCache struct {
	data       map[string]item
	tagToKeys  map[string]map[string]struct{} // 标签到键的映射，用于批量删除
	keyToTags  map[string][]string            // 键到标签的映射
	mu         sync.RWMutex
	gcInterval time.Duration
	maxEntries int
	stopCh     chan struct{}
}

type item struct {
	value      []byte
	expiration int64 // Unix 时间戳，0 表示永不过期
}

type MemoryCacheOption func(*MemoryCache)

// WithGCInterval 设置垃圾回收间隔
func WithGCInterval(d time.Duration) MemoryCacheOption {
	return func(c *MemoryCache) {
		c.gcInterval = d
	}
}

// WithMaxEntries 设置最大缓存条目数
func WithMaxEntries(n int) MemoryCacheOption {
	return func(c *MemoryCache) {
		c.maxEntries = n
	}
}

// NewMemoryCache 创建一个新的内存缓存
func NewMemoryCache(options ...MemoryCacheOption) *MemoryCache {
	cache := &MemoryCache{
		data:       make(map[string]item),
		tagToKeys:  make(map[string]map[string]struct{}),
		keyToTags:  make(map[string][]string),
		gcInterval: 5 * time.Minute,
		maxEntries: 10000,
		stopCh:     make(chan struct{}),
	}

	for _, option := range options {
		option(cache)
	}

	go cache.gcLoop()

	return cache
}

// gcLoop 定期清理过期的缓存项
func (c *MemoryCache) gcLoop() {
	ticker := time.NewTicker(c.gcInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.deleteExpired()
		case <-c.stopCh:
			return
		}
	}
}

// deleteExpired 删除所有过期的缓存项
func (c *MemoryCache) deleteExpired() {
	now := time.Now().UnixNano()
	c.mu.Lock()
	defer c.mu.Unlock()

	for k, v := range c.data {
		if v.expiration > 0 && v.expiration < now {
			c.delete(k)
		}
	}
}

// delete 内部删除，不加锁
func (c *MemoryCache) delete(key string) {
	// 移除标签关联
	if tags, ok := c.keyToTags[key]; ok {
		for _, tag := range tags {
			delete(c.tagToKeys[tag], key)
			// 如果标签没有关联的键了，删除标签
			if len(c.tagToKeys[tag]) == 0 {
				delete(c.tagToKeys, tag)
			}
		}
		delete(c.keyToTags, key)
	}

	// 删除数据
	delete(c.data, key)
}

// Get 从缓存获取值
func (c *MemoryCache) Get(ctx context.Context, key string, value interface{}) error {
	c.mu.RLock()
	item, found := c.data[key]
	c.mu.RUnlock()

	if !found {
		return ErrCacheMiss
	}

	// 检查是否过期
	if item.expiration > 0 && item.expiration < time.Now().UnixNano() {
		c.mu.Lock()
		c.delete(key)
		c.mu.Unlock()
		return ErrCacheMiss
	}

	// 反序列化数据
	return json.Unmarshal(item.value, value)
}

// Set 设置缓存值
func (c *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// 序列化数据
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}

	// 计算过期时间
	var exp int64
	if ttl > 0 {
		exp = time.Now().Add(ttl).UnixNano()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查是否需要淘汰缓存
	if c.maxEntries > 0 && len(c.data) >= c.maxEntries {
		c.evict()
	}

	c.data[key] = item{
		value:      bytes,
		expiration: exp,
	}

	return nil
}

// SetWithTags 设置缓存值，并关联标签
func (c *MemoryCache) SetWithTags(ctx context.Context, key string, value interface{}, ttl time.Duration, tags ...string) error {
	if err := c.Set(ctx, key, value, ttl); err != nil {
		return err
	}

	if len(tags) == 0 {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 确保内部结构已初始化
	if c.tagToKeys == nil {
		c.tagToKeys = make(map[string]map[string]struct{})
	}
	if c.keyToTags == nil {
		c.keyToTags = make(map[string][]string)
	}

	// 存储键与标签的关系
	c.keyToTags[key] = tags
	fmt.Printf("Setting key %s with tags: %v\n", key, tags) // 调试日志

	// 存储标签与键的关系
	for _, tag := range tags {
		if c.tagToKeys[tag] == nil {
			c.tagToKeys[tag] = make(map[string]struct{})
		}
		c.tagToKeys[tag][key] = struct{}{}
		fmt.Printf("Tag %s now has key: %s\n", tag, key) // 调试日志
	}

	return nil
}

// Delete 删除缓存值
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.delete(key)
	return nil
}

// DeleteByTags 通过标签批量删除缓存
func (c *MemoryCache) DeleteByTags(ctx context.Context, tags ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 确保内部结构已初始化
	if c.tagToKeys == nil {
		c.tagToKeys = make(map[string]map[string]struct{})
	}

	// 收集所有要删除的键
	keysToDelete := make(map[string]struct{})

	fmt.Printf("Deleting cache by tags: %v\n", tags) // 调试日志

	for _, tag := range tags {
		if keys, ok := c.tagToKeys[tag]; ok {
			fmt.Printf("Found %d keys for tag %s\n", len(keys), tag) // 调试日志
			for key := range keys {
				keysToDelete[key] = struct{}{}
			}
			// 删除标签映射
			delete(c.tagToKeys, tag)
		} else {
			fmt.Printf("No keys found for tag %s\n", tag) // 调试日志
		}
	}

	// 删除所有收集到的键
	for key := range keysToDelete {
		fmt.Printf("Deleting key: %s\n", key) // 调试日志
		delete(c.data, key)
		if keyTags, ok := c.keyToTags[key]; ok {
			// 清除该键对应的所有标签引用
			for _, tag := range keyTags {
				if tagKeys, ok := c.tagToKeys[tag]; ok {
					delete(tagKeys, key)
				}
			}
			delete(c.keyToTags, key)
		}
	}

	return nil
}

// Clear 清空缓存
func (c *MemoryCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]item)
	c.tagToKeys = make(map[string]map[string]struct{})
	c.keyToTags = make(map[string][]string)
	return nil
}

// Close 关闭缓存，停止后台goroutine
func (c *MemoryCache) Close() error {
	close(c.stopCh)
	return nil
}

// evict 淘汰部分缓存项，简单地删除最先添加的项目
func (c *MemoryCache) evict() {
	// 简单的策略：删除25%的缓存项
	toDelete := len(c.data) / 4
	if toDelete < 1 {
		toDelete = 1
	}

	deleted := 0
	for key := range c.data {
		c.delete(key)
		deleted++
		if deleted >= toDelete {
			break
		}
	}
}
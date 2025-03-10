package orm

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	// ErrCacheMiss 当缓存中没有找到对应的键时返回
	ErrCacheMiss = errors.New("orm: cache miss")

	// ErrCacheKeyEmpty 当缓存键为空时返回
	ErrCacheKeyEmpty = errors.New("orm: cache key is empty")

	// ErrCacheDisabled 当缓存功能被禁用时返回
	ErrCacheDisabled = errors.New("orm: cache is disabled")
)

// Cache 定义缓存接口，用户可以实现此接口来提供自定义缓存
type Cache interface {
	// Get 从缓存获取值，如果不存在返回 ErrCacheMiss
	Get(ctx context.Context, key string, value interface{}) error

	// Set 设置缓存值
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete 删除缓存值
	Delete(ctx context.Context, key string) error

	// DeleteByTags 通过标签批量删除缓存
	DeleteByTags(ctx context.Context, tags ...string) error

	// Clear 清空缓存
	Clear(ctx context.Context) error
}

// ModelCacheConfig 模型缓存配置
type ModelCacheConfig struct {
	// Enabled 是否启用缓存
	Enabled bool

	// TTL 缓存过期时间
	TTL time.Duration

	// Tags 缓存标签，用于批量失效
	Tags []string

	// KeyGenerator 自定义缓存键生成器
	KeyGenerator func(operation string, query *Query) string

	// Conditions 缓存条件，决定哪些查询应该被缓存
	Conditions []CacheCondition
}

// CacheCondition 缓存条件函数，决定是否应该缓存查询结果
type CacheCondition func(ctx context.Context, qc *QueryContext) bool

// CacheManager 管理与特定模型相关的缓存策略
type CacheManager struct {
	cache            Cache                                                     // 缓存实现
	modelCacheConfig map[string]*ModelCacheConfig                              // 模型缓存配置
	defaultTTL       time.Duration                                             // 默认过期时间
	enabled          bool                                                      // 是否全局启用缓存
	keyGenerator     func(model string, operation string, query *Query) string // 默认缓存键生成器
	prefix           string                                                    // 缓存键前缀
}

// NewCacheManager 创建一个新的缓存管理器
func NewCacheManager(cache Cache) *CacheManager {
	return &CacheManager{
		cache:            cache,
		modelCacheConfig: make(map[string]*ModelCacheConfig),
		defaultTTL:       5 * time.Minute, // 默认5分钟过期
		enabled:          true,
		keyGenerator:     defaultKeyGenerator,
	}
}

// WithDefaultTTL 设置默认缓存过期时间
func (cm *CacheManager) WithDefaultTTL(ttl time.Duration) *CacheManager {
	cm.defaultTTL = ttl
	return cm
}

// WithKeyGenerator 设置默认缓存键生成器
func (cm *CacheManager) WithKeyGenerator(generator func(model string, operation string, query *Query) string) *CacheManager {
	cm.keyGenerator = generator
	return cm
}

// SetModelCacheConfig 为特定模型设置缓存配置
func (cm *CacheManager) SetModelCacheConfig(modelName string, config *ModelCacheConfig) {
	cm.modelCacheConfig[modelName] = config
}

// GetModelCacheConfig 获取特定模型的缓存配置
func (cm *CacheManager) GetModelCacheConfig(modelName string) (*ModelCacheConfig, bool) {
	config, ok := cm.modelCacheConfig[modelName]
	return config, ok
}

// Enable 启用缓存功能
func (cm *CacheManager) Enable() {
	cm.enabled = true
}

// Disable 禁用缓存功能
func (cm *CacheManager) Disable() {
	cm.enabled = false
}

// IsEnabled 检查缓存功能是否启用
func (cm *CacheManager) IsEnabled() bool {
	return cm.enabled
}

// defaultKeyGenerator 默认缓存键生成器
func defaultKeyGenerator(model string, operation string, query *Query) string {
	if query == nil {
		return ""
	}

	// 一个非常简单的键生成器，实际使用时可能需要更复杂的实现
	// 例如，对 SQL 语句和参数进行哈希处理
	return model + ":" + operation + ":" + query.SQL
}

// ShouldCache 判断是否应该缓存查询结果
func (cm *CacheManager) ShouldCache(ctx context.Context, qc *QueryContext) bool {
    if !cm.enabled {
        fmt.Println("Cache is globally disabled")
        return false
    }
    
    // 只对查询操作进行缓存
    if qc.QueryType != "query" {
        fmt.Println("Not a query operation")
        return false
    }
    
    // 获取模型配置
    if qc.Model == nil {
        fmt.Println("No model in query context")
        return false
    }
    
    modelName := qc.Model.GetTableName()
    fmt.Printf("Checking cache config for model: %s\n", modelName)
    
    config, ok := cm.modelCacheConfig[modelName]
    
    // 如果没有找到模型配置或缓存被禁用，则不缓存
    if !ok || !config.Enabled {
        fmt.Printf("No config found or cache disabled for model %s\n", modelName)
        return false
    }
    
    fmt.Printf("Cache enabled for model %s\n", modelName)
    
    // 检查缓存条件
    for _, condition := range config.Conditions {
        if !condition(ctx, qc) {
            fmt.Println("Cache condition not met")
            return false
        }
    }
    
    // 检查 Builder 是否支持缓存
    if s, ok := qc.Builder.(*Selector[any]); ok {
        return s.useCache
    }
    
    return true
}

// GenerateKey 生成缓存键
func (cm *CacheManager) GenerateKey(qc *QueryContext) string {
	if qc.Model == nil || qc.Query == nil {
		return ""
	}

	modelName := qc.Model.GetTableName()
	config, ok := cm.modelCacheConfig[modelName]

	// 如果有自定义键生成器，使用它
	if ok && config.KeyGenerator != nil {
		return config.KeyGenerator(qc.QueryType, qc.Query)
	}

	// 否则使用默认键生成器
	return cm.keyGenerator(modelName, qc.QueryType, qc.Query)
}

// GetTTL 获取缓存TTL
func (cm *CacheManager) GetTTL(modelName string) time.Duration {
	config, ok := cm.modelCacheConfig[modelName]
	if ok && config.TTL > 0 {
		return config.TTL
	}
	return cm.defaultTTL
}

// GetTags 获取缓存标签
func (cm *CacheManager) GetTags(modelName string) []string {
	config, ok := cm.modelCacheConfig[modelName]
	if ok {
		return config.Tags
	}
	return nil
}

// InvalidateCache 使缓存失效的方法
func (cm *CacheManager) InvalidateCache(ctx context.Context, modelName string, tags ...string) error {
	if !cm.enabled || cm.cache == nil {
		return ErrCacheDisabled
	}

	// 修复: 使用提供的标签或模型的默认标签删除缓存
	if len(tags) > 0 {
		// 打印调试信息
		fmt.Printf("Invalidating cache with tags: %v\n", tags)
		return cm.cache.DeleteByTags(ctx, tags...)
	}

	// 获取模型的标签
	modelTags := cm.GetTags(modelName)
	if len(modelTags) > 0 {
		fmt.Printf("Invalidating cache with model tags: %v\n", modelTags)
		return cm.cache.DeleteByTags(ctx, modelTags...)
	}

	// 没有标签可用时，尝试清空与此模型相关的所有缓存
	fmt.Printf("No tags provided or defined for model %s, attempting to clear all cache\n", modelName)

	// 没有标签时，可以尝试使用模型名称作为前缀，清除所有相关缓存
	// 这需要缓存实现支持按前缀删除的功能
	if prefixCache, ok := cm.cache.(interface {
		DeleteByPrefix(ctx context.Context, prefix string) error
	}); ok {
		return prefixCache.DeleteByPrefix(ctx, modelName+":")
	}

	// 如果缓存实现不支持按前缀删除，我们可能无法精确删除相关缓存
	// 在这种情况下可以考虑清空所有缓存，但这可能太激进
	// return cm.cache.Clear(ctx)

	return fmt.Errorf("cannot invalidate cache: no tags provided or defined for model %s", modelName)
}
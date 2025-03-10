package orm

import (
	"time"
)

// FindOptions 定义查询选项
type FindOptions struct {
	Offset    int
	Limit     int
	OrderBy   []OrderBy
	shardKeys map[string]interface{} // 用于存储分片键
	// 缓存相关选项
	UseCache  bool          // 是否使用缓存
	CacheTTL  time.Duration // 缓存过期时间
	CacheTags []string      // 缓存标签
}

// FindOption 是FindOptions的构建器选项
type FindOption func(*FindOptions)

// WithLimit 设置查询结果数量限制
func WithLimit(limit int) FindOption {
	return func(o *FindOptions) {
		o.Limit = limit
	}
}

// WithOffset 设置查询结果的偏移量
func WithOffset(offset int) FindOption {
	return func(o *FindOptions) {
		o.Offset = offset
	}
}

// WithOrderBy 设置结果排序方式
func WithOrderBy(orderBy ...OrderBy) FindOption {
	return func(o *FindOptions) {
		o.OrderBy = orderBy
	}
}

// WithCache 启用缓存
func WithCache() FindOption {
	return func(o *FindOptions) {
		o.UseCache = true
	}
}

// WithCacheTTL 设置缓存过期时间
func WithCacheTTL(ttl time.Duration) FindOption {
	return func(o *FindOptions) {
		o.UseCache = true
		o.CacheTTL = ttl
	}
}

// WithCacheTags 设置缓存标签，用于批量失效
func WithCacheTags(tags ...string) FindOption {
	return func(o *FindOptions) {
		o.UseCache = true
		o.CacheTags = tags
	}
}

// UpdateOptions 定义更新选项
type UpdateOptions struct {
	ReturnOld bool
	// 缓存相关选项
	InvalidateCache bool     // 是否使缓存失效
	InvalidateTags  []string // 要失效的缓存标签
}

// UpdateOption 是UpdateOptions的构建器选项
type UpdateOption func(*UpdateOptions)

// WithReturnOld 设置是否返回更新前的文档
func WithReturnOld(returnOld bool) UpdateOption {
	return func(o *UpdateOptions) {
		o.ReturnOld = returnOld
	}
}

// WithInvalidateCache 设置是否使相关缓存失效
func WithInvalidateCache(invalidate bool) UpdateOption {
	return func(o *UpdateOptions) {
		o.InvalidateCache = invalidate
	}
}

// WithInvalidateTags 设置要使失效的缓存标签
func WithInvalidateTags(tags ...string) UpdateOption {
	return func(o *UpdateOptions) {
		o.InvalidateCache = true
		o.InvalidateTags = tags
	}
}

// InsertOptions 定义插入选项
type InsertOptions struct {
	ReturnID   bool
	IgnoreDups bool
	// 缓存相关选项
	InvalidateCache bool     // 是否使缓存失效
	InvalidateTags  []string // 要失效的缓存标签
}

// InsertOption 是InsertOptions的构建器选项
type InsertOption func(*InsertOptions)

// WithReturnID 设置是否返回插入后的ID
func WithReturnID(returnID bool) InsertOption {
	return func(o *InsertOptions) {
		o.ReturnID = returnID
	}
}

// WithIgnoreDups 设置是否忽略重复键错误
func WithIgnoreDups(ignoreDups bool) InsertOption {
	return func(o *InsertOptions) {
		o.IgnoreDups = ignoreDups
	}
}

// WithInsertInvalidateCache 设置插入操作是否使缓存失效
func WithInsertInvalidateCache() InsertOption {
	return func(o *InsertOptions) {
		o.InvalidateCache = true
	}
}

// WithInsertInvalidateTags 设置插入操作要使失效的缓存标签
func WithInsertInvalidateTags(tags ...string) InsertOption {
	return func(o *InsertOptions) {
		o.InvalidateCache = true
		o.InvalidateTags = tags
	}
}

// DeleteOptions 定义删除选项
type DeleteOptions struct {
	Limit int
	// 缓存相关选项
	InvalidateCache bool     // 是否使缓存失效
	InvalidateTags  []string // 要失效的缓存标签
}

// DeleteOption 是DeleteOptions的构建器选项
type DeleteOption func(*DeleteOptions)

// WithDeleteLimit 设置删除的最大记录数
func WithDeleteLimit(limit int) DeleteOption {
	return func(o *DeleteOptions) {
		o.Limit = limit
	}
}

// WithDeleteInvalidateCache 设置删除操作是否使缓存失效
func WithDeleteInvalidateCache() DeleteOption {
	return func(o *DeleteOptions) {
		o.InvalidateCache = true
	}
}

// WithDeleteInvalidateTags 设置删除操作要使失效的缓存标签
func WithDeleteInvalidateTags(tags ...string) DeleteOption {
	return func(o *DeleteOptions) {
		o.InvalidateCache = true
		o.InvalidateTags = tags
	}
}

// DBOptions 数据库选项
type DBOptions struct {
	// 缓存选项
	Cache            Cache                        // 缓存实现
	DefaultCacheTTL  time.Duration                // 默认缓存过期时间
	EnableCache      bool                         // 是否启用缓存
	CacheKeyPrefix   string                       // 缓存键前缀
	ModelCacheConfig map[string]*ModelCacheConfig // 模型缓存配置
}

// WithDBCache 设置缓存实现
func WithDBCache(cache Cache) DBOption {
	return func(db *DB) error {
		if db.cacheManager == nil {
			db.cacheManager = NewCacheManager(cache)
		} else {
			db.cacheManager.cache = cache
		}
		db.cacheManager.Enable()
		return nil
	}
}

// WithDefaultCacheTTL 设置默认缓存过期时间
func WithDefaultCacheTTL(ttl time.Duration) DBOption {
	return func(db *DB) error {
		if db.cacheManager == nil {
			return ErrCacheDisabled
		}
		db.cacheManager.WithDefaultTTL(ttl)
		return nil
	}
}

// WithCacheKeyPrefix 设置缓存键前缀
func WithCacheKeyPrefix(prefix string) DBOption {
	return func(db *DB) error {
		if db.cacheManager == nil {
			return ErrCacheDisabled
		}
		db.cacheManager.WithKeyPrefix(prefix)
		return nil
	}
}

// WithModelCacheConfig 为特定模型设置缓存配置
func WithModelCacheConfig(modelName string, config *ModelCacheConfig) DBOption {
	return func(db *DB) error {
		if db.cacheManager == nil {
			return ErrCacheDisabled
		}
		db.cacheManager.SetModelCacheConfig(modelName, config)
		return nil
	}
}

// DisableCache 禁用缓存功能
func DisableCache() DBOption {
	return func(db *DB) error {
		if db.cacheManager != nil {
			db.cacheManager.Disable()
		}
		return nil
	}
}


// 为 CacheManager 添加 WithKeyPrefix 方法
func (cm *CacheManager) WithKeyPrefix(prefix string) *CacheManager {
	cm.prefix = prefix
	return cm
}
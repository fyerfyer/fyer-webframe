package orm

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"
)

// ShardingError 定义分片错误
type ShardingError struct {
	Op  string // 操作名称
	Err error  // 原始错误
}

// Error 实现 error 接口
func (e *ShardingError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("sharding %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("sharding %s", e.Op)
}

// Unwrap 支持 errors.Unwrap
func (e *ShardingError) Unwrap() error {
	return e.Err
}

// ShardingRouter 负责分片路由
type ShardingRouter interface {
	// CalculateRoute 计算路由到哪个分片
	CalculateRoute(ctx context.Context, model interface{}, values map[string]interface{}) (dbName, tableName string, err error)

	// RegisterStrategy 注册分片策略
	RegisterStrategy(modelName string, strategy ShardingStrategy)

	// GetStrategy 获取模型的分片策略
	GetStrategy(modelName string) (ShardingStrategy, bool)
}

// ShardingModelInfo 保存模型的分片信息
type ShardingModelInfo struct {
	Strategy       ShardingStrategy // 分片策略
	DefaultDBName  string           // 默认数据库名
	DefaultTable   string           // 默认表名
	ModelType      reflect.Type     // 模型类型
	ShardTableFunc func(int) string // 自定义表名生成函数
}

// DefaultShardingRouter 是默认的分片路由实现
type DefaultShardingRouter struct {
	mu              sync.RWMutex
	strategies      map[string]ShardingStrategy   // 模型名称 -> 分片策略
	modelInfo       map[string]*ShardingModelInfo // 模型名称 -> 模型信息
	aliases         map[string]string             // 模型别名 -> 模型名称
	routeCache      sync.Map                      // 缓存最近的路由结果
	cacheMaxSize    int                           // 缓存最大大小
	cacheExpiration time.Duration                 // 缓存过期时间
	enableCache     bool                          // 是否启用缓存
}

// ShardingRouterOption 是路由器的配置选项
type ShardingRouterOption func(*DefaultShardingRouter)

// WithCacheSize 设置路由缓存大小
func WithCacheSize(size int) ShardingRouterOption {
	return func(r *DefaultShardingRouter) {
		r.cacheMaxSize = size
	}
}

// WithCacheExpiration 设置缓存过期时间
func WithCacheExpiration(d time.Duration) ShardingRouterOption {
	return func(r *DefaultShardingRouter) {
		r.cacheExpiration = d
	}
}

// WithCacheEnabled 设置是否启用缓存
func WithCacheEnabled(enabled bool) ShardingRouterOption {
	return func(r *DefaultShardingRouter) {
		r.enableCache = enabled
	}
}

// NewShardingRouter 创建一个默认的分片路由器
func NewShardingRouter(opts ...ShardingRouterOption) *DefaultShardingRouter {
	router := &DefaultShardingRouter{
		strategies:      make(map[string]ShardingStrategy),
		modelInfo:       make(map[string]*ShardingModelInfo),
		aliases:         make(map[string]string),
		cacheMaxSize:    1000,                 // 默认缓存1000条记录
		cacheExpiration: 5 * time.Minute,      // 默认5分钟过期
		enableCache:     true,                 // 默认启用缓存
	}

	// 应用选项
	for _, opt := range opts {
		opt(router)
	}

	// 启动缓存清理协程
	if router.enableCache {
		go router.startCacheCleaner()
	}

	return router
}

// RegisterStrategy 注册分片策略
func (r *DefaultShardingRouter) RegisterStrategy(modelName string, strategy ShardingStrategy) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.strategies[modelName] = strategy

	// 如果已经有模型信息，更新策略
	if info, exists := r.modelInfo[modelName]; exists {
		info.Strategy = strategy
	} else {
		// 创建新的模型信息
		r.modelInfo[modelName] = &ShardingModelInfo{
			Strategy: strategy,
		}
	}
}

// RegisterModelInfo 注册模型完整信息
func (r *DefaultShardingRouter) RegisterModelInfo(modelName string, info *ShardingModelInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.modelInfo[modelName] = info

	// 同时更新策略映射
	if info.Strategy != nil {
		r.strategies[modelName] = info.Strategy
	}
}

// RegisterAlias 注册模型别名
func (r *DefaultShardingRouter) RegisterAlias(alias, modelName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.aliases[alias] = modelName
}

// GetStrategy 获取模型的分片策略
func (r *DefaultShardingRouter) GetStrategy(modelName string) (ShardingStrategy, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 检查别名
	if actualName, ok := r.aliases[modelName]; ok {
		modelName = actualName
	}

	strategy, ok := r.strategies[modelName]
	return strategy, ok
}

// getCacheKey 生成缓存键
func (r *DefaultShardingRouter) getCacheKey(modelName string, shardKey string, shardValue interface{}) string {
	return fmt.Sprintf("%s:%s:%v", modelName, shardKey, shardValue)
}

type cacheEntry struct {
	DBName     string
	TableName  string
	ExpireTime time.Time
}

// CalculateRoute 计算路由
func (r *DefaultShardingRouter) CalculateRoute(ctx context.Context, model interface{}, values map[string]interface{}) (string, string, error) {
	var modelName string

	// 如果model是字符串，直接使用
	if name, ok := model.(string); ok {
		modelName = name
	} else {
		// 否则获取模型名称
		modelName = getModelName(model)
	}

	// 检查别名
	r.mu.RLock()
	if actualName, ok := r.aliases[modelName]; ok {
		modelName = actualName
	}

	// 获取模型信息
	modelInfo, infoOk := r.modelInfo[modelName]
	if !infoOk {
		// 如果没有完整的模型信息，尝试获取策略
		strategy, strategyOk := r.strategies[modelName]
		if !strategyOk {
			r.mu.RUnlock()
			return "", "", &ShardingError{Op: "calculate_route", Err: ErrModelNotRegistered}
		}

		// 创建临时模型信息
		modelInfo = &ShardingModelInfo{
			Strategy: strategy,
		}
	}
	r.mu.RUnlock()

	// 获取分片键
	shardKey := modelInfo.Strategy.GetShardKey()
	shardKeyValue, ok := values[shardKey]
	if !ok {
		return "", "", &ShardingError{Op: "get_shard_key", Err: ErrNoShardKeyFound}
	}

	// 检查缓存
	if r.enableCache {
		cacheKey := r.getCacheKey(modelName, shardKey, shardKeyValue)
		if entry, ok := r.routeCache.Load(cacheKey); ok {
			cachedEntry := entry.(cacheEntry)
			if time.Now().Before(cachedEntry.ExpireTime) {
				return cachedEntry.DBName, cachedEntry.TableName, nil
			}
			// 缓存过期，删除
			r.routeCache.Delete(cacheKey)
		}
	}

	// 使用策略计算路由
	dbIndex, tableIndex, err := modelInfo.Strategy.Route(shardKeyValue)
	if err != nil {
		return "", "", &ShardingError{Op: "route", Err: err}
	}

	// 获取分片数据库和表名
	dbName, tableName, err := modelInfo.Strategy.GetShardName(dbIndex, tableIndex)
	if err != nil {
		return "", "", &ShardingError{Op: "get_shard_name", Err: err}
	}

	// 如果有自定义表名函数，使用它
	if modelInfo.ShardTableFunc != nil {
		tableName = modelInfo.ShardTableFunc(tableIndex)
	}

	// 缓存结果
	if r.enableCache {
		cacheKey := r.getCacheKey(modelName, shardKey, shardKeyValue)
		r.routeCache.Store(cacheKey, cacheEntry{
			DBName:     dbName,
			TableName:  tableName,
			ExpireTime: time.Now().Add(r.cacheExpiration),
		})
	}

	return dbName, tableName, nil
}

// CalculateRouteForStruct 为结构体实例计算路由
func (r *DefaultShardingRouter) CalculateRouteForStruct(ctx context.Context, modelName string, modelInstance interface{}) (string, string, error) {
	// 获取模型信息
	r.mu.RLock()
	modelInfo, ok := r.modelInfo[modelName]
	if !ok {
		r.mu.RUnlock()
		return "", "", &ShardingError{Op: "get_model_info", Err: ErrModelNotRegistered}
	}
	r.mu.RUnlock()

	// 从结构体中提取分片键值
	shardKeyName := modelInfo.Strategy.GetShardKey()
	shardKeyValue, err := extractShardKeyValue(modelInstance, shardKeyName)
	if err != nil {
		return "", "", &ShardingError{Op: "extract_shard_key", Err: err}
	}

	// 使用提取的值计算路由
	values := map[string]interface{}{shardKeyName: shardKeyValue}
	return r.CalculateRoute(ctx, modelName, values)
}

// startCacheCleaner 启动缓存清理协程
func (r *DefaultShardingRouter) startCacheCleaner() {
	ticker := time.NewTicker(r.cacheExpiration / 2)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		r.routeCache.Range(func(key, value interface{}) bool {
			entry := value.(cacheEntry)
			if now.After(entry.ExpireTime) {
				r.routeCache.Delete(key)
			}
			return true
		})
	}
}

// ClearCache 清除路由缓存
func (r *DefaultShardingRouter) ClearCache() {
	r.routeCache = sync.Map{}
}

// FallbackRouter 是一个简单的fallback路由器
// 它总是返回同一个数据库和表名，用于处理未分片的模型
type FallbackRouter struct {
	defaultDB    string
	defaultTable string
}

// NewFallbackRouter 创建一个fallback路由器
func NewFallbackRouter(defaultDB, defaultTable string) *FallbackRouter {
	return &FallbackRouter{
		defaultDB:    defaultDB,
		defaultTable: defaultTable,
	}
}

// CalculateRoute 总是返回默认的数据库和表名
func (r *FallbackRouter) CalculateRoute(ctx context.Context, model interface{}, values map[string]interface{}) (string, string, error) {
	var tableName string

	// 如果model是字符串，直接使用
	if name, ok := model.(string); ok {
		tableName = name
	} else {
		// 否则尝试获取表名
		tableName = getModelName(model)
	}

	if r.defaultTable != "" {
		tableName = r.defaultTable
	}

	return r.defaultDB, tableName, nil
}

// RegisterStrategy 实现接口但不做任何事情
func (r *FallbackRouter) RegisterStrategy(modelName string, strategy ShardingStrategy) {}

// GetStrategy 总是返回nil和false
func (r *FallbackRouter) GetStrategy(modelName string) (ShardingStrategy, bool) {
	return nil, false
}

// CompositeRouter 组合多个路由器
type CompositeRouter struct {
	routers []ShardingRouter
	mu      sync.RWMutex
}

// NewCompositeRouter 创建一个组合路由器
func NewCompositeRouter(routers ...ShardingRouter) *CompositeRouter {
	return &CompositeRouter{
		routers: routers,
	}
}

// AddRouter 添加路由器
func (r *CompositeRouter) AddRouter(router ShardingRouter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routers = append(r.routers, router)
}

// CalculateRoute 依次尝试每个路由器，直到成功
func (r *CompositeRouter) CalculateRoute(ctx context.Context, model interface{}, values map[string]interface{}) (string, string, error) {
	r.mu.RLock()
	routers := r.routers
	r.mu.RUnlock()

	var lastErr error
	for _, router := range routers {
		dbName, tableName, err := router.CalculateRoute(ctx, model, values)
		if err == nil {
			return dbName, tableName, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return "", "", lastErr
	}

	return "", "", errors.New("no router available")
}

// RegisterStrategy 在所有路由器中注册策略
func (r *CompositeRouter) RegisterStrategy(modelName string, strategy ShardingStrategy) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, router := range r.routers {
		router.RegisterStrategy(modelName, strategy)
	}
}

// GetStrategy 从第一个能找到策略的路由器返回
func (r *CompositeRouter) GetStrategy(modelName string) (ShardingStrategy, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, router := range r.routers {
		if strategy, ok := router.GetStrategy(modelName); ok {
			return strategy, true
		}
	}

	return nil, false
}

// extractShardKeyFromOptions 从查询选项中提取分片键
func extractShardKeyFromOptions(opts []FindOption, shardKey string) (interface{}, bool) {
	// 创建临时的 FindOptions 实例来收集选项
	options := &FindOptions{}

	// 应用所有选项
	for _, opt := range opts {
		opt(options)
	}

	// 检查 OrderBy 选项中是否包含分片键
	for _, order := range options.OrderBy {
		if col, ok := order.expr.(*Column); ok {
			if col.name == shardKey {
				return nil, true // 找到了分片键但无法获取值
			}
		}
	}

	// 如果找不到分片键，返回 false
	return nil, false
}

// WithShardKey 添加分片键信息到查询选项
func WithShardKey(key string, value interface{}) FindOption {
	return func(o *FindOptions) {
		// 如果 FindOptions 还没有存储分片信息的字段
		// 可以在该结构体中添加一个映射来存储
		if o.shardKeys == nil {
			o.shardKeys = make(map[string]interface{})
		}
		o.shardKeys[key] = value
	}
}
package sharding

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
)

var (
	ErrModelNotFound     = errors.New("model not registered in sharding router")
	ErrStrategyNotFound  = errors.New("no sharding strategy found for model")
	ErrNoShardKeyFound   = errors.New("shard key not found in query values")
	ErrInvalidShardValue = errors.New("invalid shard key value")
	ErrEmptyValues       = errors.New("values map cannot be empty")
)

// ModelInfo 保存模型的分片信息
type ModelInfo struct {
	Strategy       Strategy // 分片策略
	ShardKey       string   // 分片键
	TablePrefix    string   // 表名前缀
	DefaultDBIndex int      // 默认数据库索引
}

// Router 分片路由器接口
type Router interface {
	// RegisterStrategy 注册模型使用的分片策略
	RegisterStrategy(modelName string, strategy Strategy)

	// RegisterModelInfo 注册模型的完整分片信息
	RegisterModelInfo(modelName string, info *ModelInfo)

	// CalculateRoute 根据模型名称和查询值计算路由
	CalculateRoute(ctx context.Context, modelName string, values map[string]interface{}) (string, string, error)

	// GetModelInfo 获取模型分片信息
	GetModelInfo(modelName string) (*ModelInfo, bool)
}

// DefaultRouter 默认路由器实现
type DefaultRouter struct {
	// 模型名称 -> 模型的分片信息
	models map[string]*ModelInfo
	// 模型别名映射表
	aliases map[string]string
	// 缓存最近的路由结果，提高性能
	routeCache sync.Map // key: modelName+shardKeyValue, value: [dbName, tableName]
	// 缓存最大大小
	cacheSize int
	// 保护并发访问
	mu sync.RWMutex
}

// NewDefaultRouter 创建默认路由器
func NewDefaultRouter() *DefaultRouter {
	return &DefaultRouter{
		models:    make(map[string]*ModelInfo),
		aliases:   make(map[string]string),
		cacheSize: 1000, // 默认缓存1000个路由结果
	}
}

// WithCacheSize 设置路由缓存大小
func (r *DefaultRouter) WithCacheSize(size int) *DefaultRouter {
	r.cacheSize = size
	return r
}

// RegisterStrategy 注册分片策略
func (r *DefaultRouter) RegisterStrategy(modelName string, strategy Strategy) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查模型信息是否已存在
	info, exists := r.models[modelName]
	if !exists {
		// 创建新的模型信息
		info = &ModelInfo{
			Strategy: strategy,
		}

		// 尝试从策略中获取分片键
		// 修改类型断言，使用类型判断
		switch s := strategy.(type) {
		case *HashStrategy:
			info.ShardKey = s.BaseStrategy.ShardKey
		case *ModStrategy:
			info.ShardKey = s.BaseStrategy.ShardKey
		case *RangeStrategy:
			info.ShardKey = s.BaseStrategy.ShardKey
		case *DateStrategy:
			info.ShardKey = s.BaseStrategy.ShardKey
		default:
			// 如果是其他类型的策略，尝试通过反射获取ShardKey
			val := reflect.ValueOf(strategy)
			if val.Kind() == reflect.Ptr && !val.IsNil() {
				// 尝试访问BaseStrategy字段
				baseField := val.Elem().FieldByName("BaseStrategy")
				if baseField.IsValid() && baseField.Kind() == reflect.Ptr && !baseField.IsNil() {
					shardKeyField := baseField.Elem().FieldByName("ShardKey")
					if shardKeyField.IsValid() && shardKeyField.Kind() == reflect.String {
						info.ShardKey = shardKeyField.String()
					}
				}
			}
		}

		r.models[modelName] = info
	} else {
		// 更新现有模型信息的策略
		info.Strategy = strategy

		// 同样尝试更新分片键
		switch s := strategy.(type) {
		case *HashStrategy:
			info.ShardKey = s.BaseStrategy.ShardKey
		case *ModStrategy:
			info.ShardKey = s.BaseStrategy.ShardKey
		case *RangeStrategy:
			info.ShardKey = s.BaseStrategy.ShardKey
		case *DateStrategy:
			info.ShardKey = s.BaseStrategy.ShardKey
		}
	}
}

// RegisterModelInfo 注册模型的完整分片信息
func (r *DefaultRouter) RegisterModelInfo(modelName string, info *ModelInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.models[modelName] = info
}

// RegisterModelAlias 注册模型别名
func (r *DefaultRouter) RegisterModelAlias(alias, modelName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.aliases[alias] = modelName
}

// GetModelInfo 获取模型分片信息
func (r *DefaultRouter) GetModelInfo(modelName string) (*ModelInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 先检查别名
	if aliasedModel, ok := r.aliases[modelName]; ok {
		modelName = aliasedModel
	}

	info, ok := r.models[modelName]
	return info, ok
}

// CalculateRoute 计算路由
func (r *DefaultRouter) CalculateRoute(ctx context.Context, modelName string, values map[string]interface{}) (string, string, error) {
	// 检查是否为空
	if len(values) == 0 {
		return "", "", ErrEmptyValues
	}

	// 优先检查别名
	r.mu.RLock()
	if aliasedModel, ok := r.aliases[modelName]; ok {
		modelName = aliasedModel
	}
	r.mu.RUnlock()

	// 获取模型分片信息
	info, ok := r.GetModelInfo(modelName)
	if !ok {
		return "", "", ErrModelNotFound
	}

	// 检查是否有分片策略
	if info.Strategy == nil {
		return "", "", ErrStrategyNotFound
	}

	// 检查缓存
	if shardKeyValue, ok := values[info.ShardKey]; ok {
		cacheKey := getCacheKey(modelName, shardKeyValue)
		if result, found := r.routeCache.Load(cacheKey); found {
			routes := result.([]string)
			return routes[0], routes[1], nil
		}

		// 计算路由
		dbIndex, tableIndex, err := info.Strategy.Route(shardKeyValue)
		if err != nil {
			return "", "", err
		}

		// 获取分片数据库和表名
		dbName, tableName, err := info.Strategy.GetShardName(dbIndex, tableIndex)
		if err != nil {
			return "", "", err
		}

		// 缓存结果
		r.routeCache.Store(cacheKey, []string{dbName, tableName})

		return dbName, tableName, nil
	}

	// 如果没有找到分片键，但有默认设置，则使用默认值
	if info.DefaultDBIndex >= 0 {
		dbName, tableName, err := info.Strategy.GetShardName(info.DefaultDBIndex, 0)
		if err != nil {
			return "", "", err
		}
		return dbName, tableName, nil
	}

	return "", "", ErrNoShardKeyFound
}

// CalculateRouteForValue 为单个值计算路由
func (r *DefaultRouter) CalculateRouteForValue(ctx context.Context, modelName string, shardKeyName string, shardKeyValue interface{}) (string, string, error) {
	values := map[string]interface{}{shardKeyName: shardKeyValue}
	return r.CalculateRoute(ctx, modelName, values)
}

// CalculateRouteForModel 为模型实例计算路由
func (r *DefaultRouter) CalculateRouteForModel(ctx context.Context, modelName string, modelInstance interface{}) (string, string, error) {
	// 获取模型分片信息
	info, ok := r.GetModelInfo(modelName)
	if !ok {
		return "", "", ErrModelNotFound
	}

	// 从模型实例中提取分片键的值
	v := reflect.ValueOf(modelInstance)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return "", "", errors.New("model instance must be a struct or pointer to struct")
	}

	// 查找字段
	var shardKeyValue interface{}
	found := false

	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		if strings.EqualFold(field.Name, info.ShardKey) {
			shardKeyValue = v.Field(i).Interface()
			found = true
			break
		}
	}

	if !found {
		return "", "", ErrNoShardKeyFound
	}

	// 计算路由
	dbIndex, tableIndex, err := info.Strategy.Route(shardKeyValue)
	if err != nil {
		return "", "", err
	}

	// 获取分片数据库和表名
	dbName, tableName, err := info.Strategy.GetShardName(dbIndex, tableIndex)
	if err != nil {
		return "", "", err
	}

	// 缓存结果
	cacheKey := getCacheKey(modelName, shardKeyValue)
	r.routeCache.Store(cacheKey, []string{dbName, tableName})

	return dbName, tableName, nil
}

// ClearCache 清除路由缓存
func (r *DefaultRouter) ClearCache() {
	r.routeCache = sync.Map{}
}

// getCacheKey 生成缓存键
func getCacheKey(modelName string, shardKeyValue interface{}) string {
	return modelName + ":" + reflect.TypeOf(shardKeyValue).String() + ":" + reflect.ValueOf(shardKeyValue).String()
}

// RoutingConfig 路由配置选项
type RoutingConfig struct {
	// 路由信息
	Router Router
	// 默认数据库
	DefaultDB string
	// 设置读写分离
	EnableReadWriteSeparation bool
	// 主库、从库比例 (1:n)
	MasterSlaveRatio int
	// 分片字段获取方式
	ShardKeyResolver func(context.Context, map[string]interface{}) (interface{}, error)
}

// RoutingOption 路由配置选项
type RoutingOption func(*RoutingConfig)

// WithDefaultDB 设置默认数据库
func WithDefaultDB(dbName string) RoutingOption {
	return func(c *RoutingConfig) {
		c.DefaultDB = dbName
	}
}

// WithReadWriteSeparation 启用读写分离
func WithReadWriteSeparation(enable bool, ratio int) RoutingOption {
	return func(c *RoutingConfig) {
		c.EnableReadWriteSeparation = enable
		c.MasterSlaveRatio = ratio
	}
}

// WithShardKeyResolver 设置分片键解析器
func WithShardKeyResolver(resolver func(context.Context, map[string]interface{}) (interface{}, error)) RoutingOption {
	return func(c *RoutingConfig) {
		c.ShardKeyResolver = resolver
	}
}
package orm

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/fyerfyer/fyer-webframe/orm/internal/sharding"
)

var (
	// ErrNoShardKeyFound 当查询中未找到分片键时返回
	ErrNoShardKeyFound = errors.New("orm: no shard key found in query")

	// ErrInvalidShardKey 当分片键值无效时返回
	ErrInvalidShardKey = errors.New("orm: invalid shard key value")

	// ErrModelNotRegistered 当模型未注册分片策略时返回
	ErrModelNotRegistered = errors.New("orm: model not registered for sharding")

	// ErrShardNotAvailable 当目标分片不可用时返回
	ErrShardNotAvailable = errors.New("orm: target shard not available")

	// ErrShardingDisabled 当分片功能未启用时返回
	ErrShardingDisabled = errors.New("orm: sharding is not enabled")
)

// ShardingStrategy 分片策略接口
// 由具体的分片策略实现
type ShardingStrategy interface {
	// Route 计算给定键值应该路由到哪个分片
	Route(key interface{}) (dbIndex, tableIndex int, err error)

	// GetShardName 获取分片的数据库和表名
	GetShardName(dbIndex, tableIndex int) (dbName, tableName string, err error)

	// GetShardKey 获取分片键名称
	GetShardKey() string
}

// ShardingManager 管理分片数据库连接和路由
// 每个分片DB实例包含一个ShardingManager
type ShardingManager struct {
	mu         sync.RWMutex
	shards     map[string]*DB        // 分片名称到DB的映射
	router     ShardingRouter        // 分片路由器
	defaultDB  *DB                   // 默认DB
	modelCache map[string]*modelInfo // 模型缓存
	enabled    bool                  // 是否启用分片
}

// modelInfo 保存模型的分片信息
type modelInfo struct {
	strategy      ShardingStrategy // 分片策略
	defaultDBName string           // 默认分片DB名称
}

// NewShardingManager 创建分片管理器
func NewShardingManager(defaultDB *DB, router ShardingRouter) *ShardingManager {
	if router == nil {
		router = NewShardingRouter()
	}

	return &ShardingManager{
		shards:     make(map[string]*DB),
		router:     router,
		defaultDB:  defaultDB,
		modelCache: make(map[string]*modelInfo),
		enabled:    true,
	}
}

// RegisterShard 注册分片数据库
func (m *ShardingManager) RegisterShard(name string, db *DB) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shards[name] = db
}

// GetShard 获取指定分片数据库
func (m *ShardingManager) GetShard(name string) (*DB, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	db, ok := m.shards[name]
	return db, ok
}

// SetDefaultDB 设置默认数据库
func (m *ShardingManager) SetDefaultDB(db *DB) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultDB = db
}

// GetDefaultDB 获取默认数据库
func (m *ShardingManager) GetDefaultDB() *DB {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultDB
}

// SetRouter 设置路由器
func (m *ShardingManager) SetRouter(router ShardingRouter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.router = router
}

// GetRouter 获取路由器
func (m *ShardingManager) GetRouter() ShardingRouter {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.router
}

// RegisterModelInfo 注册模型分片信息
func (m *ShardingManager) RegisterModelInfo(modelName string, strategy ShardingStrategy, defaultDBName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 同时注册到路由器和本地缓存
	m.router.RegisterStrategy(modelName, strategy)
	m.modelCache[modelName] = &modelInfo{
		strategy:      strategy,
		defaultDBName: defaultDBName,
	}
}

// GetModelInfo 获取模型分片信息
func (m *ShardingManager) GetModelInfo(modelName string) (*modelInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	info, ok := m.modelCache[modelName]
	return info, ok
}

// Route 根据模型和查询值路由到正确的分片
func (m *ShardingManager) Route(ctx context.Context, modelName string, values map[string]interface{}) (*DB, string, error) {
	if !m.enabled {
		return m.defaultDB, "", ErrShardingDisabled
	}

	// 计算路由
	dbName, tableName, err := m.router.CalculateRoute(ctx, modelName, values)
	if err != nil {
		// 路由失败，使用默认数据库
		info, ok := m.GetModelInfo(modelName)
		if !ok {
			return m.defaultDB, "", nil
		}

		// 尝试使用模型指定的默认分片
		if info.defaultDBName != "" {
			db, ok := m.GetShard(info.defaultDBName)
			if ok {
				return db, tableName, nil
			}
		}

		return m.defaultDB, tableName, nil
	}

	// 获取对应的分片数据库
	db, ok := m.GetShard(dbName)
	if !ok {
		// 如果获取分片失败，尝试使用默认数据库
		return m.defaultDB, tableName, fmt.Errorf("shard %s not found: %w", dbName, ErrShardNotAvailable)
	}

	return db, tableName, nil
}

// RouteWithKey 根据分片键值直接路由
func (m *ShardingManager) RouteWithKey(ctx context.Context, modelName string, shardKey string, shardValue interface{}) (*DB, string, error) {
	values := map[string]interface{}{
		shardKey: shardValue,
	}
	return m.Route(ctx, modelName, values)
}

// RouteForModel 为模型实例计算路由
func (m *ShardingManager) RouteForModel(ctx context.Context, modelName string, model interface{}) (*DB, string, error) {
	if !m.enabled {
		return m.defaultDB, "", ErrShardingDisabled
	}

	// 获取模型的分片信息
	info, ok := m.GetModelInfo(modelName)
	if !ok {
		return m.defaultDB, "", ErrModelNotRegistered
	}

	// 从模型实例中提取分片键的值
	shardKeyValue, err := extractShardKeyValue(model, info.strategy.GetShardKey())
	if err != nil {
		return m.defaultDB, "", err
	}

	// 使用分片策略计算路由
	dbIndex, tableIndex, err := info.strategy.Route(shardKeyValue)
	if err != nil {
		// 如果路由失败，使用默认分片
		if info.defaultDBName != "" {
			db, ok := m.GetShard(info.defaultDBName)
			if ok {
				return db, "", nil
			}
		}
		return m.defaultDB, "", err
	}

	// 获取分片数据库和表名
	dbName, tableName, err := info.strategy.GetShardName(dbIndex, tableIndex)
	if err != nil {
		return m.defaultDB, "", err
	}

	// 获取对应的分片数据库
	db, ok := m.GetShard(dbName)
	if !ok {
		// 如果获取分片失败，尝试使用默认数据库
		return m.defaultDB, tableName, fmt.Errorf("shard %s not found: %w", dbName, ErrShardNotAvailable)
	}

	return db, tableName, nil
}

// Enable 启用分片功能
func (m *ShardingManager) Enable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = true
}

// Disable 禁用分片功能
func (m *ShardingManager) Disable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = false
}

// IsEnabled 检查分片功能是否启用
func (m *ShardingManager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
}

// Close 关闭所有分片连接
func (m *ShardingManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error

	// 关闭所有分片连接
	for name, db := range m.shards {
		if err := db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close shard %s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing shards: %v", errs)
	}
	return nil
}

// extractShardKeyValue 从模型实例中提取分片键值
func extractShardKeyValue(model interface{}, shardKey string) (interface{}, error) {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, errors.New("model instance must be a struct or pointer to struct")
	}

	// 查找字段
	var found bool
	var fieldValue reflect.Value

	// 首先尝试直接按名称查找
	fieldValue = v.FieldByName(shardKey)
	if fieldValue.IsValid() {
		found = true
	} else {
		// 不区分大小写地查找字段
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			if field.Name == shardKey {
				fieldValue = v.Field(i)
				found = true
				break
			}
		}
	}

	if !found {
		return nil, ErrNoShardKeyFound
	}

	return fieldValue.Interface(), nil
}

// ShardingDBOption 是ShardingDB配置选项
type ShardingDBOption func(*ShardingDB)

// ShardingDB 扩展DB，支持分片功能
type ShardingDB struct {
	*DB                              // 嵌入原有DB
	shardingManager *ShardingManager // 分片管理器
}

// NewShardingDB 创建支持分片的DB
func NewShardingDB(defaultDB *DB, router ShardingRouter, opts ...ShardingDBOption) *ShardingDB {
	shardDB := &ShardingDB{
		DB:              defaultDB,
		shardingManager: NewShardingManager(defaultDB, router),
	}

	// 应用配置选项
	for _, opt := range opts {
		opt(shardDB)
	}

	return shardDB
}

// WithShardingRouter 设置分片路由器
func WithShardingRouter(router ShardingRouter) ShardingDBOption {
	return func(sdb *ShardingDB) {
		sdb.shardingManager.SetRouter(router)
	}
}

// WithDefaultDB 设置默认数据库
func WithDefaultDB(db *DB) ShardingDBOption {
	return func(sdb *ShardingDB) {
		sdb.shardingManager.SetDefaultDB(db)
	}
}

// RegisterShard 注册分片数据库
func (sdb *ShardingDB) RegisterShard(name string, db *DB) {
	sdb.shardingManager.RegisterShard(name, db)
}

// RegisterShardStrategy 注册模型分片策略
func (sdb *ShardingDB) RegisterShardStrategy(modelName string, strategy ShardingStrategy, defaultDBName string) {
	sdb.shardingManager.RegisterModelInfo(modelName, strategy, defaultDBName)
}

// ShardConfig 分片配置
type ShardConfig struct {
	Driver      string
	DSN         string
	DialectName string
	DBOptions   []DBOption
}

// ConfigureShards 配置分片
func (sdb *ShardingDB) ConfigureShards(shardConfigs map[string]ShardConfig) error {
	for name, config := range shardConfigs {
		// 创建分片数据库
		db, err := OpenDB(config.Driver, config.DSN, config.DialectName, config.DBOptions...)
		if err != nil {
			return fmt.Errorf("failed to open shard %s: %w", name, err)
		}

		// 注册分片
		sdb.RegisterShard(name, db)
	}

	return nil
}

// GetShardDB 获取指定分片的数据库
func (sdb *ShardingDB) GetShardDB(name string) (*DB, bool) {
	return sdb.shardingManager.GetShard(name)
}

// GetDefaultDB 获取默认数据库
func (sdb *ShardingDB) GetDefaultDB() *DB {
	return sdb.shardingManager.GetDefaultDB()
}

// Route 路由到特定分片
func (sdb *ShardingDB) Route(ctx context.Context, modelName string, values map[string]interface{}) (*DB, string, error) {
	return sdb.shardingManager.Route(ctx, modelName, values)
}

// RouteWithKey 根据分片键值路由
func (sdb *ShardingDB) RouteWithKey(ctx context.Context, modelName string, shardKey string, shardValue interface{}) (*DB, string, error) {
	return sdb.shardingManager.RouteWithKey(ctx, modelName, shardKey, shardValue)
}

// EnableSharding 启用分片功能
func (sdb *ShardingDB) EnableSharding() {
	sdb.shardingManager.Enable()
}

// DisableSharding 禁用分片功能
func (sdb *ShardingDB) DisableSharding() {
	sdb.shardingManager.Disable()
}

// Close 关闭所有分片连接
func (sdb *ShardingDB) Close() error {
	// 先关闭分片管理器中的所有分片
	if err := sdb.shardingManager.Close(); err != nil {
		return err
	}

	// 然后关闭默认DB
	return sdb.DB.Close()
}

// AsShardingClient 将DB转换为支持分片的Client
func (sdb *ShardingDB) AsShardingClient() *ShardingClient {
	// 创建普通客户端
	client := sdb.DB.NewClient()

	// 包装为支持分片的客户端
	return &ShardingClient{
		Client:          client,
		shardingManager: sdb.shardingManager,
	}
}

// NewClient 创建支持分片的客户端
func (sdb *ShardingDB) NewClient() *ShardingClient {
	// 创建普通客户端
	client := sdb.DB.NewClient()

	// 包装为支持分片的客户端
	return &ShardingClient{
		Client:          client,
		shardingManager: sdb.shardingManager,
	}
}

// ShardedCollection 支持分片的集合
type ShardedCollection struct {
	modelType       interface{}
	modelName       string
	shardingManager *ShardingManager
}

// Find 查找单条记录
func (sc *ShardedCollection) Find(ctx context.Context, where ...Condition) (interface{}, error) {
	// 从查询条件中提取分片键值
	values, err := extractShardKeyFromConditions(where, sc.modelName, sc.shardingManager)
	if err != nil {
		// 如果无法确定分片，则使用默认数据库
		defaultDB := sc.shardingManager.GetDefaultDB()
		client := defaultDB.NewClient()
		coll := client.Collection(sc.modelType)
		return coll.Find(ctx, where...)
	}

	// 路由到正确的分片
	db, tableName, err := sc.shardingManager.Route(ctx, sc.modelName, values)
	if err != nil {
		// 路由失败，使用默认数据库
		defaultDB := sc.shardingManager.GetDefaultDB()
		client := defaultDB.NewClient()
		coll := client.Collection(sc.modelType)
		return coll.Find(ctx, where...)
	}

	// 创建对应分片的Client
	client := db.NewClient()
	coll := client.Collection(sc.modelType)

	// 处理表名重写
	if tableName != "" && tableName != sc.modelName {
		// TODO: 重写表名的方法
	}

	// 执行查询
	return coll.Find(ctx, where...)
}

// FindAll 查找多条记录
func (sc *ShardedCollection) FindAll(ctx context.Context, where ...Condition) ([]interface{}, error) {
	// 实现类似 Find 的逻辑，但返回多条记录
	// ...

	// 临时返回
	return nil, errors.New("not implemented")
}

// Insert 插入记录
func (sc *ShardedCollection) Insert(ctx context.Context, model interface{}) (Result, error) {
	// 为模型实例计算路由
	db, tableName, err := sc.shardingManager.RouteForModel(ctx, sc.modelName, model)
	if err != nil {
		// 路由失败，使用默认数据库
		defaultDB := sc.shardingManager.GetDefaultDB()
		client := defaultDB.NewClient()
		coll := client.Collection(sc.modelType)
		return coll.Insert(ctx, model)
	}

	// 创建对应分片的Client
	client := db.NewClient()
	coll := client.Collection(sc.modelType)

	// 处理表名重写
	if tableName != "" && tableName != sc.modelName {
		// TODO: 重写表名
	}

	// 执行插入
	return coll.Insert(ctx, model)
}

// Update 更新记录
func (sc *ShardedCollection) Update(ctx context.Context, update map[string]interface{}, where ...Condition) (Result, error) {
	// 实现类似 Find 的路由逻辑，然后执行更新
	// ...

	// 临时返回
	return Result{}, errors.New("not implemented")
}

// Delete 删除记录
func (sc *ShardedCollection) Delete(ctx context.Context, where ...Condition) (Result, error) {
	// 实现类似 Find 的路由逻辑，然后执行删除
	// ...

	// 临时返回
	return Result{}, errors.New("not implemented")
}

// defaultShardingRouter 默认路由器实现
type defaultShardingRouter struct {
	mu         sync.RWMutex
	strategies map[string]ShardingStrategy
}

// RegisterStrategy 注册分片策略
func (r *defaultShardingRouter) RegisterStrategy(modelName string, strategy ShardingStrategy) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.strategies[modelName] = strategy
}

// GetStrategy 获取模型的分片策略
func (r *defaultShardingRouter) GetStrategy(modelName string) (ShardingStrategy, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	strategy, ok := r.strategies[modelName]
	return strategy, ok
}

// CalculateRoute 计算路由
func (r *defaultShardingRouter) CalculateRoute(ctx context.Context, model interface{}, values map[string]interface{}) (string, string, error) {
	var modelName string

	// 如果model是字符串，直接使用
	if name, ok := model.(string); ok {
		modelName = name
	} else {
		// 否则尝试获取模型名称
		modelName = getModelName(model)
	}

	// 获取模型的分片策略
	strategy, ok := r.GetStrategy(modelName)
	if !ok {
		return "", "", ErrModelNotRegistered
	}

	// 从values中获取分片键值
	shardKey := strategy.GetShardKey()
	shardKeyValue, ok := values[shardKey]
	if !ok {
		return "", "", ErrNoShardKeyFound
	}

	// 使用策略计算路由
	dbIndex, tableIndex, err := strategy.Route(shardKeyValue)
	if err != nil {
		return "", "", err
	}

	// 获取分片数据库和表名
	return strategy.GetShardName(dbIndex, tableIndex)
}

// extractShardKeyFromConditions 从条件中提取分片键值
func extractShardKeyFromConditions(conditions []Condition, modelName string, manager *ShardingManager) (map[string]interface{}, error) {
	// 获取模型的分片信息
	info, ok := manager.GetModelInfo(modelName)
	if !ok {
		return nil, ErrModelNotRegistered
	}

	// 获取分片键
	shardKey := info.strategy.GetShardKey()

	// 从条件中查找分片键
	values := make(map[string]interface{})
	found := false

	// 遍历所有条件
	for _, cond := range conditions {
		// 尝试类型断言为Predicate
		if pred, ok := cond.(*Predicate); ok {
			// 检查是否为相等操作符，且左侧为列
			if pred.op == opEQ {
				if col, ok := pred.left.(*Column); ok {
					// 检查列名是否为分片键
					if col.name == shardKey {
						// 提取右侧的值
						if val, ok := pred.right.(*Value); ok {
							values[shardKey] = val.val
							found = true
							break
						}
					}
				}
			}
		}
	}

	if !found {
		return nil, ErrNoShardKeyFound
	}

	return values, nil
}

// WithHashStrategy 为模型创建哈希分片策略
func WithHashStrategy(dbPrefix string, dbCount int, tablePrefix string, tableCount int, shardKey string) ShardingStrategy {
	return &shardingStrategyAdapter{
		inner: sharding.NewHashStrategy(dbPrefix, dbCount, tablePrefix, tableCount, shardKey),
	}
}

// WithModStrategy 为模型创建取模分片策略
func WithModStrategy(dbPrefix string, dbCount int, tablePrefix string, tableCount int, shardKey string) ShardingStrategy {
	return &shardingStrategyAdapter{
		inner: sharding.NewModStrategy(dbPrefix, dbCount, tablePrefix, tableCount, shardKey),
	}
}

// WithRangeStrategy 为模型创建范围分片策略
func WithRangeStrategy(dbPrefix string, dbCount int, tablePrefix string, tableCount int, shardKey string, ranges []int64) ShardingStrategy {
	return &shardingStrategyAdapter{
		inner: sharding.NewRangeStrategy(dbPrefix, dbCount, tablePrefix, tableCount, shardKey, ranges),
	}
}

// WithDateStrategy 为模型创建日期分片策略
func WithDateStrategy(dbPrefix string, dbCount int, tablePrefix string, tableCount int, shardKey string, dateFormat string) ShardingStrategy {
	return &shardingStrategyAdapter{
		inner: sharding.NewDateStrategy(dbPrefix, dbCount, tablePrefix, tableCount, shardKey, dateFormat),
	}
}

// shardingStrategyAdapter 适配器，使内部分片策略实现ShardingStrategy接口
type shardingStrategyAdapter struct {
	inner sharding.Strategy
}

// Route 实现ShardingStrategy.Route
func (s *shardingStrategyAdapter) Route(key interface{}) (int, int, error) {
	return s.inner.Route(key)
}

// GetShardName 实现ShardingStrategy.GetShardName
func (s *shardingStrategyAdapter) GetShardName(dbIndex, tableIndex int) (string, string, error) {
	return s.inner.GetShardName(dbIndex, tableIndex)
}

// GetShardKey 实现ShardingStrategy.GetShardKey
func (s *shardingStrategyAdapter) GetShardKey() string {
	// 直接尝试使用类型判断代替类型断言
	switch st := s.inner.(type) {
	case *sharding.HashStrategy:
		return st.BaseStrategy.ShardKey
	case *sharding.ModStrategy:
		return st.BaseStrategy.ShardKey
	case *sharding.RangeStrategy:
		return st.BaseStrategy.ShardKey
	case *sharding.DateStrategy:
		return st.BaseStrategy.ShardKey
	}

	// 通过反射获取ShardKey
	v := reflect.ValueOf(s.inner)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		if bs := v.FieldByName("BaseStrategy"); bs.IsValid() && !bs.IsNil() {
			if bs.Kind() == reflect.Ptr {
				bs = bs.Elem()
			}
			shardKey := bs.FieldByName("ShardKey")
			if shardKey.IsValid() && shardKey.Kind() == reflect.String {
				return shardKey.String()
			}
		}
	}

	// 默认返回空
	return ""
}

// ExecuteOnAllShards 在所有分片上执行操作
func (sdb *ShardingDB) ExecuteOnAllShards(ctx context.Context, fn func(db *DB) error) []error {
	var errors []error
	var mu sync.Mutex

	// 获取所有分片
	shards := make(map[string]*DB)
	sdb.shardingManager.mu.RLock()
	for name, db := range sdb.shardingManager.shards {
		shards[name] = db
	}
	sdb.shardingManager.mu.RUnlock()

	// 并发执行
	var wg sync.WaitGroup
	for name, db := range shards {
		wg.Add(1)
		go func(name string, db *DB) {
			defer wg.Done()

			if err := fn(db); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("shard %s: %w", name, err))
				mu.Unlock()
			}
		}(name, db)
	}

	// 等待所有操作完成
	wg.Wait()

	return errors
}

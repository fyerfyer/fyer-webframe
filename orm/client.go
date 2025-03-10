package orm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// Client 是对底层ORM框架的简洁封装，提供更方便的CRUD操作
type Client struct {
	db *DB
}

// New 创建一个新的ORM客户端
func New(db *DB) *Client {
	return &Client{db: db}
}

// Collection 返回指定模型的集合操作器
func (c *Client) Collection(modelType interface{}) *Collection {
	return &Collection{
		client:    c,
		modelType: modelType,
		modelName: getModelName(modelType),
	}
}

// getModelName 获取模型名称
func getModelName(model interface{}) string {
	if namer, ok := model.(interface{ TableName() string }); ok {
		return namer.TableName()
	}

	t := reflect.TypeOf(model)
	// 处理指针类型
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// 默认使用类型名称
	if t.Kind() == reflect.Struct {
		return t.Name()
	}

	return fmt.Sprintf("%T", model)
}

// Transaction 执行事务
func (c *Client) Transaction(ctx context.Context, fn func(tc *Client) error) error {
	return c.db.Tx(ctx, func(tx *Tx) error {
		// 创建一个基于事务的客户端
		txClient := &Client{db: tx.getDB()}
		return fn(txClient)
	}, nil)
}

// Close 关闭客户端连接
func (c *Client) Close() error {
	return c.db.Close()
}

// Raw 执行原始SQL查询
func (c *Client) Raw(ctx context.Context, sql string, args ...interface{}) (*sql.Rows, error) {
	return c.db.queryContext(ctx, sql, args...)
}

// Exec 执行原始SQL命令
func (c *Client) Exec(ctx context.Context, sql string, args ...interface{}) (Result, error) {
	result, err := c.db.execContext(ctx, sql, args...)
	if err != nil {
		return Result{err: err}, err
	}
	return Result{res: result}, nil
}

// Count 执行计数查询
func (c *Client) Count(ctx context.Context, model interface{}, where ...Condition) (int64, error) {
	// 获取数据库和模型信息
	db := c.db
	m, err := db.getModel(model)
	if err != nil {
		return 0, err
	}

	// 手动构建COUNT查询SQL
	builder := &strings.Builder{}
	args := make([]any, 0)

	builder.WriteString("SELECT COUNT(*) FROM ")
	builder.WriteString(db.dialect.Quote(m.table))

	// 添加WHERE条件
	if len(where) > 0 {
		builder.WriteString(" WHERE ")
		for i, cond := range where {
			if pred, ok := cond.(*Predicate); ok {
				pred.model = m
			}
			cond.Build(builder, &args)
			if i < len(where)-1 {
				builder.WriteString(" AND ")
			}
		}
	}

	builder.WriteString(";")
	query := builder.String()

	// 执行查询
	rows, err := db.queryContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	// 获取结果
	if !rows.Next() {
		return 0, errors.New("no rows returned for count query")
	}

	var count int64
	if err := rows.Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

// RegisterModel 注册模型到客户端
func (c *Client) RegisterModel(name string, model interface{}) {
	Register(name, model)
}

// GetRegisteredModel 获取注册的模型
func (c *Client) GetRegisteredModel(name string) (interface{}, bool) {
	return DefaultModelRegistry.Get(name)
}

// GetDB 获取底层数据库连接
// 这个方法为高级用户提供直接访问底层ORM的能力
func (c *Client) GetDB() *DB {
	return c.db
}

// WithCache 启用缓存，返回具有缓存功能的客户端
func (c *Client) WithCache() *Client {
	if c.db.cacheManager != nil {
		c.db.cacheManager.Enable()
	}
	return c
}

// WithoutCache 禁用缓存，返回禁用缓存的客户端
func (c *Client) WithoutCache() *Client {
	if c.db.cacheManager != nil {
		c.db.cacheManager.Disable()
	}
	return c
}

// InvalidateCache 使指定模型的缓存失效
func (c *Client) InvalidateCache(ctx context.Context, modelName string, tags ...string) error {
	if c.db.cacheManager == nil || !c.db.cacheManager.IsEnabled() {
		return ErrCacheDisabled
	}
	return c.db.InvalidateCache(ctx, modelName, tags...)
}

// SetModelCacheConfig 为特定模型设置缓存配置
func (c *Client) SetModelCacheConfig(modelName string, config *ModelCacheConfig) {
	c.db.SetModelCacheConfig(modelName, config)
}


//=================== 分片相关接口 ===================

// ShardingClient 是支持分片功能的客户端
type ShardingClient struct {
	*Client
	shardingManager *ShardingManager
}

// NewShardingClient 创建一个支持分片的客户端
func NewShardingClient(db *DB) *ShardingClient {
	if !db.IsSharded() || db.shardingManager == nil {
		// 自动启用分片
		db.EnableSharding(NewShardingManager(db, NewShardingRouter()))
	}

	return &ShardingClient{
		Client:          New(db),
		shardingManager: db.shardingManager,
	}
}

// ShardedCollection 获取支持分片的集合
func (c *ShardingClient) ShardedCollection(modelType interface{}) *ShardedCollection {
	modelName := getModelName(modelType)
	return &ShardedCollection{
		modelType:       modelType,
		modelName:       modelName,
		shardingManager: c.shardingManager,
	}
}

// RegisterShardStrategy 注册分片策略
func (c *ShardingClient) RegisterShardStrategy(modelName string, strategy ShardingStrategy, defaultDBName string) {
	c.shardingManager.RegisterModelInfo(modelName, strategy, defaultDBName)
}

// RouteWithKey 根据分片键值路由请求到正确的分片
func (c *ShardingClient) RouteWithKey(ctx context.Context, modelName string, shardKey string, shardValue interface{}) (*DB, string, error) {
	return c.shardingManager.RouteWithKey(ctx, modelName, shardKey, shardValue)
}

// ExecuteOnShard 在指定分片上执行操作
func (c *ShardingClient) ExecuteOnShard(ctx context.Context, shardName string, fn func(db *DB) error) error {
	db, ok := c.shardingManager.GetShard(shardName)
	if !ok {
		return fmt.Errorf("shard %s not found", shardName)
	}

	return fn(db)
}

// ExecuteOnAllShards 在所有分片上执行操作
func (c *ShardingClient) ExecuteOnAllShards(ctx context.Context, fn func(db *DB) error) []error {
	var errors []error

	c.shardingManager.mu.RLock()
	shards := make(map[string]*DB)
	for name, db := range c.shardingManager.shards {
		shards[name] = db
	}
	c.shardingManager.mu.RUnlock()

	// 并发执行
	var wg sync.WaitGroup
	var mu sync.Mutex

	for name, db := range shards {
		wg.Add(1)
		go func(name string, db *DB) {
			defer wg.Done()

			err := fn(db)
			if err != nil {
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

// WithShardKey 创建一个带有分片键信息的查询上下文
func (c *ShardingClient) WithShardKey(modelName string, shardKey string, shardValue interface{}) *ShardingQueryContext {
	return &ShardingQueryContext{
		client:     c,
		modelName:  modelName,
		shardKey:   shardKey,
		shardValue: shardValue,
	}
}

// ShardingQueryContext 包含分片查询的上下文信息
type ShardingQueryContext struct {
	client     *ShardingClient
	modelName  string
	shardKey   string
	shardValue interface{}
}

// Collection 获取指定模型的集合，带有分片路由信息
func (sqc *ShardingQueryContext) Collection(modelType interface{}) *Collection {
	db, tableName, err := sqc.client.RouteWithKey(context.Background(), sqc.modelName, sqc.shardKey, sqc.shardValue)
	if err != nil {
		// 如果路由失败，使用默认数据库
		db = sqc.client.shardingManager.GetDefaultDB()
	}

	// 创建一个基于特定分片的客户端
	shardClient := New(db)

	coll := shardClient.Collection(modelType)

	// 如果需要重写表名
	if tableName != "" && tableName != coll.modelName {
		// 这里可以添加表名重写逻辑
	}

	return coll
}

// Exec 执行原始SQL命令，会自动路由到正确的分片
func (sqc *ShardingQueryContext) Exec(ctx context.Context, sql string, args ...interface{}) (Result, error) {
	db, _, err := sqc.client.RouteWithKey(ctx, sqc.modelName, sqc.shardKey, sqc.shardValue)
	if err != nil {
		// 如果路由失败，使用默认数据库
		db = sqc.client.shardingManager.GetDefaultDB()
	}

	shardClient := New(db)
	return shardClient.Exec(ctx, sql, args...)
}

// Raw 执行原始SQL查询，会自动路由到正确的分片
func (sqc *ShardingQueryContext) Raw(ctx context.Context, sql string, args ...interface{}) (*sql.Rows, error) {
	db, _, err := sqc.client.RouteWithKey(ctx, sqc.modelName, sqc.shardKey, sqc.shardValue)
	if err != nil {
		// 如果路由失败，使用默认数据库
		db = sqc.client.shardingManager.GetDefaultDB()
	}

	shardClient := New(db)
	return shardClient.Raw(ctx, sql, args...)
}
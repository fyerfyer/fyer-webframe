package sharding

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrShardNotFound      = errors.New("shard not found")
	ErrShardAlreadyExists = errors.New("shard already exists")
	ErrInvalidShardName   = errors.New("invalid shard name")
	ErrNoDefaultShard     = errors.New("no default shard configured")
	ErrShardManagerClosed = errors.New("shard manager is closed")
)

// DBProvider 定义数据库提供者，用于创建数据库连接
type DBProvider interface {
	// GetDB 返回指定分片的数据库连接
	GetDB(dbName string) (*sql.DB, error)
}

// ShardConfig 分片配置
type ShardConfig struct {
	Driver      string            // 数据库驱动
	DSN         string            // 连接字符串
	MaxIdleConn int               // 最大空闲连接数
	MaxOpenConn int               // 最大打开连接数
	MaxLifetime time.Duration     // 连接最大生命周期
	Options     map[string]string // 其他选项
}

// ShardedDB 分片数据库，包含多个物理分片
type ShardedDB struct {
	sync.RWMutex
	shards       map[string]*sql.DB     // 分片名称到数据库连接的映射
	configs      map[string]ShardConfig // 分片配置
	router       Router                 // 分片路由器
	defaultShard string                 // 默认分片
	provider     DBProvider             // 数据库提供者
	initialized  bool                   // 是否已初始化
	closed       bool                   // 是否已关闭
	stats        *ShardStats            // 分片统计
}

// ShardStats 分片使用统计
type ShardStats struct {
	sync.RWMutex
	RouteCount     map[string]int64 // 路由次数统计
	RouteMiss      int64            // 路由失败次数
	CacheHit       int64            // 缓存命中次数
	CacheMiss      int64            // 缓存未命中次数
	FallbackCount  int64            // 降级到默认分片次数
	LastAccessTime map[string]time.Time // 最后访问时间
}

// NewShardStats 创建分片统计
func NewShardStats() *ShardStats {
	return &ShardStats{
		RouteCount:     make(map[string]int64),
		LastAccessTime: make(map[string]time.Time),
	}
}

// NewShardedDB 创建分片数据库
func NewShardedDB(router Router) *ShardedDB {
	return &ShardedDB{
		shards:  make(map[string]*sql.DB),
		configs: make(map[string]ShardConfig),
		router:  router,
		stats:   NewShardStats(),
	}
}

// WithDBProvider 设置数据库提供者
func (sdb *ShardedDB) WithDBProvider(provider DBProvider) *ShardedDB {
	sdb.Lock()
	defer sdb.Unlock()
	sdb.provider = provider
	return sdb
}

// WithDefaultShard 设置默认分片
func (sdb *ShardedDB) WithDefaultShard(shardName string) *ShardedDB {
	sdb.Lock()
	defer sdb.Unlock()
	sdb.defaultShard = shardName
	return sdb
}

// AddShard 添加分片
func (sdb *ShardedDB) AddShard(name string, config ShardConfig) error {
	sdb.Lock()
	defer sdb.Unlock()

	if sdb.closed {
		return ErrShardManagerClosed
	}

	if _, exists := sdb.shards[name]; exists {
		return ErrShardAlreadyExists
	}

	// 保存配置
	sdb.configs[name] = config

	// 如果提供了数据库连接池配置，立即创建连接
	if config.DSN != "" {
		db, err := sql.Open(config.Driver, config.DSN)
		if err != nil {
			return fmt.Errorf("failed to open database connection: %w", err)
		}

		// 设置连接池参数
		if config.MaxIdleConn > 0 {
			db.SetMaxIdleConns(config.MaxIdleConn)
		}
		if config.MaxOpenConn > 0 {
			db.SetMaxOpenConns(config.MaxOpenConn)
		}
		if config.MaxLifetime > 0 {
			db.SetConnMaxLifetime(config.MaxLifetime)
		}

		// 测试连接
		if err := db.Ping(); err != nil {
			db.Close()
			return fmt.Errorf("failed to ping database: %w", err)
		}

		sdb.shards[name] = db
	}

	return nil
}

// RemoveShard 移除分片
func (sdb *ShardedDB) RemoveShard(name string) error {
	sdb.Lock()
	defer sdb.Unlock()

	if sdb.closed {
		return ErrShardManagerClosed
	}

	db, exists := sdb.shards[name]
	if !exists {
		return ErrShardNotFound
	}

	// 关闭数据库连接
	if err := db.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	// 移除分片
	delete(sdb.shards, name)
	delete(sdb.configs, name)

	return nil
}

// GetShard 获取指定分片的数据库连接
func (sdb *ShardedDB) GetShard(name string) (*sql.DB, error) {
	sdb.RLock()
	db, exists := sdb.shards[name]
	sdb.RUnlock()

	if exists {
		// 更新统计信息
		sdb.stats.Lock()
		sdb.stats.RouteCount[name]++
		sdb.stats.LastAccessTime[name] = time.Now()
		sdb.stats.Unlock()
		return db, nil
	}

	// 如果分片不存在但有配置，尝试创建
	sdb.Lock()
	defer sdb.Unlock()

	// 再次检查，可能在获取锁的过程中已被创建
	if db, exists = sdb.shards[name]; exists {
		return db, nil
	}

	// 检查配置是否存在
	config, configExists := sdb.configs[name]
	if !configExists {
		// 如果提供了数据库提供者，尝试从提供者获取
		if sdb.provider != nil {
			providerDB, err := sdb.provider.GetDB(name)
			if err != nil {
				sdb.stats.Lock()
				sdb.stats.RouteMiss++
				sdb.stats.Unlock()
				return nil, fmt.Errorf("shard not found and provider failed: %w", err)
			}
			sdb.shards[name] = providerDB
			return providerDB, nil
		}
		sdb.stats.Lock()
		sdb.stats.RouteMiss++
		sdb.stats.Unlock()
		return nil, ErrShardNotFound
	}

	// 创建数据库连接
	db, err := sql.Open(config.Driver, config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// 设置连接池参数
	if config.MaxIdleConn > 0 {
		db.SetMaxIdleConns(config.MaxIdleConn)
	}
	if config.MaxOpenConn > 0 {
		db.SetMaxOpenConns(config.MaxOpenConn)
	}
	if config.MaxLifetime > 0 {
		db.SetConnMaxLifetime(config.MaxLifetime)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// 保存连接
	sdb.shards[name] = db

	// 更新统计信息
	sdb.stats.Lock()
	sdb.stats.RouteCount[name] = 1
	sdb.stats.LastAccessTime[name] = time.Now()
	sdb.stats.Unlock()

	return db, nil
}

// GetDefaultShard 获取默认分片
func (sdb *ShardedDB) GetDefaultShard() (*sql.DB, error) {
	sdb.RLock()
	defaultShard := sdb.defaultShard
	sdb.RUnlock()

	if defaultShard == "" {
		return nil, ErrNoDefaultShard
	}

	return sdb.GetShard(defaultShard)
}

// Route 根据模型和查询值路由到正确的分片
func (sdb *ShardedDB) Route(ctx context.Context, modelName string, values map[string]interface{}) (*sql.DB, string, string, error) {
	if sdb.closed {
		return nil, "", "", ErrShardManagerClosed
	}

	// 使用路由器计算路由
	dbName, tableName, err := sdb.router.CalculateRoute(ctx, modelName, values)
	if err != nil {
		// 路由失败，使用默认分片
		sdb.stats.Lock()
		sdb.stats.FallbackCount++
		sdb.stats.Unlock()

		defaultDB, defaultErr := sdb.GetDefaultShard()
		if defaultErr != nil {
			return nil, "", "", fmt.Errorf("route failed and no default shard: %w", err)
		}

		// 返回默认分片和原始表名
		return defaultDB, dbName, tableName, nil
	}

	// 获取对应的分片数据库连接
	db, err := sdb.GetShard(dbName)
	if err != nil {
		// 如果获取分片失败，尝试使用默认分片
		sdb.stats.Lock()
		sdb.stats.FallbackCount++
		sdb.stats.Unlock()

		defaultDB, defaultErr := sdb.GetDefaultShard()
		if defaultErr != nil {
			return nil, "", "", fmt.Errorf("failed to get shard and no default shard: %w", err)
		}

		return defaultDB, dbName, tableName, nil
	}

	return db, dbName, tableName, nil
}

// RouteWithKey 根据模型名和分片键值直接路由
func (sdb *ShardedDB) RouteWithKey(ctx context.Context, modelName string, shardKey string, shardValue interface{}) (*sql.DB, string, string, error) {
	values := map[string]interface{}{
		shardKey: shardValue,
	}
	return sdb.Route(ctx, modelName, values)
}

// RouteForModel 为模型实例计算路由
func (sdb *ShardedDB) RouteForModel(ctx context.Context, modelName string, model interface{}) (*sql.DB, string, string, error) {
	if router, ok := sdb.router.(*DefaultRouter); ok {
		dbName, tableName, err := router.CalculateRouteForModel(ctx, modelName, model)
		if err != nil {
			// 路由失败，使用默认分片
			sdb.stats.Lock()
			sdb.stats.FallbackCount++
			sdb.stats.Unlock()

			defaultDB, defaultErr := sdb.GetDefaultShard()
			if defaultErr != nil {
				return nil, "", "", fmt.Errorf("route failed and no default shard: %w", err)
			}

			// 未能确定正确的表名，使用模型名作为表名
			return defaultDB, "", modelName, nil
		}

		// 获取对应的分片数据库连接
		db, err := sdb.GetShard(dbName)
		if err != nil {
			// 如果获取分片失败，尝试使用默认分片
			sdb.stats.Lock()
			sdb.stats.FallbackCount++
			sdb.stats.Unlock()

			defaultDB, defaultErr := sdb.GetDefaultShard()
			if defaultErr != nil {
				return nil, "", "", fmt.Errorf("failed to get shard and no default shard: %w", err)
			}

			return defaultDB, dbName, tableName, nil
		}

		return db, dbName, tableName, nil
	}

	// 如果不是DefaultRouter，则使用常规路由
	return sdb.Route(ctx, modelName, map[string]interface{}{})
}

// GetShardNames 获取所有分片名称
func (sdb *ShardedDB) GetShardNames() []string {
	sdb.RLock()
	defer sdb.RUnlock()

	names := make([]string, 0, len(sdb.shards))
	for name := range sdb.shards {
		names = append(names, name)
	}

	return names
}

// ExecuteOnAllShards 在所有分片上执行指定操作
func (sdb *ShardedDB) ExecuteOnAllShards(ctx context.Context, fn func(db *sql.DB, shardName string) error) []error {
	sdb.RLock()
	shardsCopy := make(map[string]*sql.DB)
	for name, db := range sdb.shards {
		shardsCopy[name] = db
	}
	sdb.RUnlock()

	var errors []error
	var errorsMu sync.Mutex

	// 对每个分片执行操作
	var wg sync.WaitGroup
	for name, db := range shardsCopy {
		wg.Add(1)
		go func(db *sql.DB, name string) {
			defer wg.Done()

			// 检查上下文是否已取消
			select {
			case <-ctx.Done():
				errorsMu.Lock()
				errors = append(errors, fmt.Errorf("context canceled for shard %s: %w", name, ctx.Err()))
				errorsMu.Unlock()
				return
			default:
			}

			// 执行操作
			if err := fn(db, name); err != nil {
				errorsMu.Lock()
				errors = append(errors, fmt.Errorf("shard %s: %w", name, err))
				errorsMu.Unlock()
			}
		}(db, name)
	}

	// 等待所有操作完成
	wg.Wait()

	return errors
}

// ExecuteOnShard 在指定分片上执行操作
func (sdb *ShardedDB) ExecuteOnShard(ctx context.Context, shardName string, fn func(db *sql.DB) error) error {
	db, err := sdb.GetShard(shardName)
	if err != nil {
		return err
	}

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return fmt.Errorf("context canceled: %w", ctx.Err())
	default:
	}

	return fn(db)
}

// GetStats 获取分片统计信息
func (sdb *ShardedDB) GetStats() ShardStats {
	sdb.stats.RLock()
	defer sdb.stats.RUnlock()

	// 复制统计信息
	stats := ShardStats{
		RouteCount:     make(map[string]int64),
		RouteMiss:      sdb.stats.RouteMiss,
		CacheHit:       sdb.stats.CacheHit,
		CacheMiss:      sdb.stats.CacheMiss,
		FallbackCount:  sdb.stats.FallbackCount,
		LastAccessTime: make(map[string]time.Time),
	}

	for k, v := range sdb.stats.RouteCount {
		stats.RouteCount[k] = v
	}

	for k, v := range sdb.stats.LastAccessTime {
		stats.LastAccessTime[k] = v
	}

	return stats
}

// Close 关闭所有分片连接
func (sdb *ShardedDB) Close() error {
	sdb.Lock()
	defer sdb.Unlock()

	if sdb.closed {
		return ErrShardManagerClosed
	}

	var errs []error

	// 关闭所有分片连接
	for name, db := range sdb.shards {
		if err := db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close shard %s: %w", name, err))
		}
	}

	// 清空分片映射
	sdb.shards = make(map[string]*sql.DB)
	sdb.closed = true

	if len(errs) > 0 {
		return fmt.Errorf("errors closing shards: %v", errs)
	}
	return nil
}

// ShardDBProvider 是基于配置的数据库提供者实现
type ShardDBProvider struct {
	configs map[string]ShardConfig
	dbs     sync.Map
}

// NewShardDBProvider 创建新的数据库提供者
func NewShardDBProvider(configs map[string]ShardConfig) *ShardDBProvider {
	return &ShardDBProvider{
		configs: configs,
	}
}

// GetDB 根据分片名称获取数据库连接
func (p *ShardDBProvider) GetDB(dbName string) (*sql.DB, error) {
	// 先检查缓存
	if db, ok := p.dbs.Load(dbName); ok {
		return db.(*sql.DB), nil
	}

	// 检查配置
	config, exists := p.configs[dbName]
	if !exists {
		return nil, ErrShardNotFound
	}

	// 创建新连接
	db, err := sql.Open(config.Driver, config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// 设置连接池参数
	if config.MaxIdleConn > 0 {
		db.SetMaxIdleConns(config.MaxIdleConn)
	}
	if config.MaxOpenConn > 0 {
		db.SetMaxOpenConns(config.MaxOpenConn)
	}
	if config.MaxLifetime > 0 {
		db.SetConnMaxLifetime(config.MaxLifetime)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// 缓存连接
	p.dbs.Store(dbName, db)

	return db, nil
}
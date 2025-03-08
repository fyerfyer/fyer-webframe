package orm

import (
	"context"
	"database/sql"

	"github.com/fyerfyer/fyer-kit/pool"
	"github.com/fyerfyer/fyer-webframe/orm/internal/ferr"
)

// DB 是orm用来管理数据库连接和缓存之类持久化内容的结构体
type DB struct {
	model       *modelCache  // 元数据缓存
	sqlDB       *sql.DB      // 数据库连接
	dialect     Dialect      // 数据库方言
	handler     Handler      // 处理器
	middlewares []Middleware // 中间件

	// 连接池相关
	pooledDB *PooledDB // 连接池封装

	// 自动迁移相关
	schemaManager *SchemaManager // 架构管理器
}

// queryContext 查询
func (db *DB) queryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if db.pooledDB != nil && db.pooledDB.IsPooled() {
		// 从池中获取连接
		sqlDB, conn, err := db.pooledDB.GetConn(ctx)
		if err != nil {
			return nil, err
		}

		// 执行查询
		rows, err := sqlDB.QueryContext(ctx, query, args...)

		// 查询执行后直接归还连接
		db.pooledDB.PutConn(conn, err)

		return rows, err
	}

	return db.sqlDB.QueryContext(ctx, query, args...)
}

func (db *DB) execContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if db.pooledDB != nil && db.pooledDB.IsPooled() {
		// 从池中获取连接
		sqlDB, conn, err := db.pooledDB.GetConn(ctx)
		if err != nil {
			return nil, err
		}

		// 执行命令
		res, err := sqlDB.ExecContext(ctx, query, args...)

		// 归还连接
		db.pooledDB.PutConn(conn, err)

		return res, err
	}

	return db.sqlDB.ExecContext(ctx, query, args...)
}

// DBOption 定义配置项
type DBOption func(*DB) error

// getModel 获取元数据
func (db *DB) getModel(val any) (*model, error) {
	m, err := db.model.get(val)
	if err != nil {
		return nil, err
	}
	// 设置方言
	m.SetDialect(db.dialect)
	return m, nil
}

// getDB 获取db对象
func (db *DB) getDB() *DB {
	return db
}

// Open 使用已有数据库创建db对象
func Open(db *sql.DB, dialectName string, opts ...DBOption) (*DB, error) {
	dialect, ok := dialects[dialectName]
	if !ok {
		return nil, ferr.ErrInvalidDialect(dialectName)
	}

	d := &DB{
		model:   NewModelCache(),
		sqlDB:   db,
		dialect: dialect,
	}

	// 初始化核心处理器
	d.handler = &CoreHandler{db: d}

	// 初始化Schema管理器
	d.schemaManager = NewSchemaManager(d)

	// 应用配置项
	for _, opt := range opts {
		if err := opt(d); err != nil {
			return nil, err
		}
	}

	return d, nil
}

// OpenDB 使用dsn和驱动创建数据库后创建db对象
func OpenDB(driver, dsn string, dialectName string, opts ...DBOption) (*DB, error) {
	sqlDB, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}

	return Open(sqlDB, dialectName, opts...)
}

// Close 关闭数据库连接
func (db *DB) Close() error {
	if db.pooledDB != nil && db.pooledDB.IsPooled() {
		return db.pooledDB.Close()
	}
	return db.sqlDB.Close()
}

// BeginTx 开启事务
func (db *DB) BeginTx(ctx context.Context, opt *sql.TxOptions) (*Tx, error) {
	if db.pooledDB != nil && db.pooledDB.IsPooled() {
		// 从池中获取专用连接
		sqlDB, conn, err := db.pooledDB.GetConn(ctx)
		if err != nil {
			return nil, err
		}

		// 在此连接上开启事务
		tx, err := sqlDB.BeginTx(ctx, opt)
		if err != nil {
			db.pooledDB.PutConn(conn, err)
			return nil, err
		}

		// 返回包含连接的事务
		return &Tx{
			db:       db,
			tx:       tx,
			poolConn: conn,
		}, nil
	}

	tx, err := db.sqlDB.BeginTx(ctx, opt)
	if err != nil {
		return nil, err
	}

	return &Tx{
		db: db,
		tx: tx,
	}, nil
}

// Tx 事务闭包处理
func (db *DB) Tx(ctx context.Context, fn func(tx *Tx) error, opt *sql.TxOptions) (err error) {
	var tx *Tx
	tx, err = db.BeginTx(ctx, opt)
	if err != nil {
		return err
	}

	panicked := true
	defer func() {
		if panicked || err != nil {
			_ = tx.RollBack()
		}
	}()

	err = fn(tx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	panicked = false
	return err
}

// Use 添加中间件
func (db *DB) Use(middlewares ...Middleware) {
	db.middlewares = append(db.middlewares, middlewares...)
	db.handler = BuildChain(&CoreHandler{db: db}, db.middlewares)
}

func (db *DB) getHandler() Handler {
	return db.handler
}

func (db *DB) HandleQuery(ctx context.Context, qc *QueryContext) (*QueryResult, error) {
	return db.handler.QueryHandler(ctx, qc)
}

// PoolStats 返回连接池统计信息
func (db *DB) PoolStats() pool.Stats {
	if db.pooledDB != nil && db.pooledDB.IsPooled() {
		return db.pooledDB.Stats()
	}
	return pool.Stats{}
}

// 实现 Layer 接口的连接池相关方法
func (db *DB) getConn(ctx context.Context) (*sql.DB, pool.Connection, error) {
	if db.pooledDB != nil && db.pooledDB.IsPooled() {
		return db.pooledDB.GetConn(ctx)
	}
	return db.sqlDB, nil, nil
}

func (db *DB) putConn(conn pool.Connection, err error) {
	if db.pooledDB != nil && db.pooledDB.IsPooled() {
		db.pooledDB.PutConn(conn, err)
	}
}

// NewClient 创建一个封装的ORM客户端
func (db *DB) NewClient() *Client {
	return New(db)
}

// ======== 自动迁移相关接口 ========

// AutoMigrate 自动迁移模型到数据库
// 依次将传入的模型在数据库中创建表或更新表结构
func (db *DB) AutoMigrate(ctx context.Context, models ...interface{}) error {
	for _, model := range models {
		if err := db.schemaManager.MigrateModel(ctx, model); err != nil {
			return err
		}
	}
	return nil
}

// AutoMigrateWithOptions 自动迁移模型到数据库，支持选项
func (db *DB) AutoMigrateWithOptions(ctx context.Context, opts []MigrateOption, models ...interface{}) error {
	for _, model := range models {
		if err := db.schemaManager.MigrateModel(ctx, model, opts...); err != nil {
			return err
		}
	}
	return nil
}

// MigrateModel 迁移单个模型
func (db *DB) MigrateModel(ctx context.Context, model interface{}, opts ...MigrateOption) error {
	return db.schemaManager.MigrateModel(ctx, model, opts...)
}

// RegisterModel 注册模型的同时提供自动迁移选项
func (db *DB) RegisterModel(name string, model interface{}, autoMigrate bool, opts ...MigrateOption) error {
	// 注册模型
	Register(name, model)

	// 如果需要自动迁移
	if autoMigrate {
		return db.schemaManager.MigrateModel(context.Background(), model, opts...)
	}

	return nil
}

// RegisterModels 同时注册多个模型
func (db *DB) RegisterModels(autoMigrate bool, models map[string]interface{}) error {
	for name, model := range models {
		if err := db.RegisterModel(name, model, autoMigrate); err != nil {
			return err
		}
	}
	return nil
}

// MigrateOptions 返回当前DB的迁移选项
func (db *DB) MigrateOptions() *MigrateOptions {
	options := &MigrateOptions{
		Strategy: AlterIfNeeded,
		CreateMigrationLog: true,
	}
	return options
}
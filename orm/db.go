package orm

import (
	"context"
	"database/sql"

	"github.com/fyerfyer/fyer-webframe/orm/internal/ferr"
)

// DB 是orm用来管理数据库连接和缓存之类持久化内容的结构体
type DB struct {
	model       *modelCache  // 元数据缓存
	sqlDB       *sql.DB      // 数据库连接
	dialect     Dialect      // 数据库方言
	handler     Handler      // 处理器
	middlewares []Middleware // 中间件
}

// queryContext 查询
func (db *DB) queryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return db.sqlDB.QueryContext(ctx, query, args...)
}

func (db *DB) execContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
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

// BeginTx 开启事务
func (db *DB) BeginTx(ctx context.Context, opt *sql.TxOptions) (*Tx, error) {
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
			err = tx.RollBack()
		}
		err = tx.Commit()
	}()

	err = fn(tx)
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

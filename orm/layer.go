package orm

import (
	"context"
	"database/sql"
)

// Layer 用于将db和tx固结在一起
type Layer interface {
	// getModel 获取元数据
	// 我们需要获取元数据，但 layer 可能是 DB 或 TX
	// 所以需要一个方法来统一获取元数据
	getModel(val any) (*model, error)

	// getDB 获取DB
	getDB() *DB

	// 中间件相关方法
	// getHandler 获取处理器
	getHandler() Handler
	HandleQuery(ctx context.Context, qc *QueryContext) (*QueryResult, error)

	queryContext (ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	execContext (ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}
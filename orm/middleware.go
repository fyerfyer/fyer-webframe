package orm

import (
	"context"
	"database/sql"
	"fmt"
)

// Handler 处理器接口定义
type Handler interface {
	QueryHandler (ctx context.Context, qc *QueryContext)(*QueryResult, error)
}

// Middleware 中间件定义
type Middleware func(Handler) Handler

// QueryContext 查询上下文定义
type QueryContext struct {
	QueryType  string
	Query      *Query
	Model      *model
	Builder    QueryBuilder
}

// QueryResult 查询结果定义
type QueryResult struct {
	Result Result
	Rows   *sql.Rows
	Err    error
}

// BuildChain 构建处理器调用链
func BuildChain(core Handler, ms []Middleware) Handler {
	h := core
	// 从后往前构建,保证最先添加的中间件最先执行
	for i := len(ms) - 1; i >= 0; i-- {
		h = ms[i](h)
	}
	return h
}

// CoreHandler 核心处理器
// CoreHandler 是整个中间件链的最后一环，它负责实际执行数据库操作。
type CoreHandler struct {
	db *DB
}

func (c *CoreHandler) QueryHandler(ctx context.Context, qc *QueryContext) (*QueryResult, error) {
	switch qc.QueryType {
	case "query":
		rows, err := c.db.queryContext(ctx, qc.Query.SQL, qc.Query.Args...)
		return &QueryResult{
			Rows: rows,
			Err:  err,
		}, err
	case "exec":
		res, err := c.db.execContext(ctx, qc.Query.SQL, qc.Query.Args...)
		return &QueryResult{
			Result: Result{
				res: res,
				err: err,
			},
			Err: err,
		}, err
	default:
		return nil, fmt.Errorf("unknown query type: %s", qc.QueryType)
	}
}



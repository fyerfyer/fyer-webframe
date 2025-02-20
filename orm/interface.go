package orm

import (
	"context"
	"database/sql"
)

// Querier 抽取公共的查询接口
type Querier[T any] interface {
	Build() (*Query, error)
	Where(conditions ...Condition) Querier[T]
}

// SelectorInterface 查询接口
type SelectorInterface[T any] interface {
	Querier[T]
	Select(cols ...Selectable) *Selector[T]
	OrderBy(col string, desc bool) *Selector[T]
	Limit(limit int) *Selector[T]
	Offset(offset int) *Selector[T]
	Get(ctx context.Context) (*T, error)
	GetMulti(ctx context.Context) ([]*T, error)
}

// DeleterInterface 删除接口
type DeleterInterface[T any] interface {
	Querier[T]
	Delete() *Deleter[T]
	Limit(limit int) *Deleter[T]
}

// TableNamer 表名接口
type TableNamer interface {
	TableName() string
}

// Executor 添加执行器接口
type Executor interface {
	Exec(ctx context.Context) (sql.Result, error)
}
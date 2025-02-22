package orm

import (
	"context"
)

// QueryBuilder 查询构建接口
type QueryBuilder interface {
	Build() (*Query, error)
}

// Executor 执行接口
type Executor interface {
	Exec(ctx context.Context) (Result, error)
}

// SelectorInterface 查询接口
type SelectorInterface[T any] interface {
	QueryBuilder
	Select(cols ...Selectable) *Selector[T]
	Where(conditions ...Condition) *Selector[T]
	GroupBy(cols ...Selectable) *Selector[T]
	Having(conditions ...Condition) *Selector[T]
	OrderBy(orders ...OrderBy) *Selector[T]
	Limit(num int) *Selector[T]
	Offset(num int) *Selector[T]
	Get(ctx context.Context) (*T, error)
	GetMulti(ctx context.Context) ([]*T, error)
}

// DeleterInterface 删除接口
type DeleterInterface[T any] interface {
	QueryBuilder
	Executor
	Delete(cols ...Selectable) *Deleter[T]
	Where(conditions ...Condition) *Deleter[T]
	Limit(num int) *Deleter[T]
	Offset(num int) *Deleter[T]
}

// InserterInterface 插入接口
type InserterInterface[T any] interface {
	QueryBuilder
	Executor
	Insert(cols []string, vals ...*T) *Inserter[T]
	Upsert(conflictCols []*Column, cols []*Column) *Inserter[T]
}

// TableNamer 表名接口
type TableNamer interface {
	TableName() string
}
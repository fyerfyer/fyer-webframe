package orm

import "context"

type Querier[T any] interface {
	Get(ctx context.Context) (*T, error)        // 获取单个结果
	GetMulti(ctx context.Context) ([]*T, error) // 获取多个结果
}

type SelectorInterface[T any] interface {
	From(table string) SelectorInterface[T]             // 指定表名
	Where(conditions ...Condition) SelectorInterface[T] // Where条件
	Select(cols ...string) SelectorInterface[T]         // 指定列
	OrderBy(col string, desc bool) SelectorInterface[T] // 排序
	Limit(limit int) SelectorInterface[T]               // 限制行数
	Offset(offset int) SelectorInterface[T]             // 偏移量
	Build() (*Query, error)                             // 构建SQL
}

type DeleterInterface[T any] interface {
	From(table string) DeleterInterface[T]             // 指定表名
	Where(conditions ...Condition) DeleterInterface[T] // Where条件
	Build() (*Query, error)                            // 构建SQL
}

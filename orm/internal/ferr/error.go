package ferr

import (
	"errors"
	"fmt"
)

var (
	ErrNoRows            = fmt.Errorf("data not found")
	ErrTooManyRows       = fmt.Errorf("too many rows")
	ErrInsertRowNotFound = fmt.Errorf("insert row not found")
	ErrUpsertRowNotFound = fmt.Errorf("upsert row not found")
	ErrPointerOnly       = errors.New("orm: 只支持指向结构体的指针，例如 *User")
)

var (
	ErrPoolClosed              = errors.New("orm: 连接池已关闭")
	ErrPoolFull                = errors.New("orm: 连接池已满")
	ErrPoolExhausted           = errors.New("orm: 连接池资源耗尽")
	ErrConnTimeout             = errors.New("orm: 获取连接超时")
	ErrInvalidConnection       = errors.New("orm: 无效的数据库连接")
	ErrTransactionOnBrokenConn = errors.New("orm: 无法在已损坏的连接上创建事务")
	ErrTooManyClients          = errors.New("orm: 等待连接的客户端过多")
	ErrDBClosed                = errors.New("orm: 在已关闭的数据库上执行操作")
)

func ErrInvalidColumn(col string) error {
	return fmt.Errorf("invalid column name: %s", col)
}

func ErrInvalidTag(tag string) error {
	return fmt.Errorf("orm: 无效的标签 %s", tag)
}

func ErrInvalidSelectable(col any) error {
	return fmt.Errorf("invalid selectable column: %v", col)
}

func ErrInvalidSubqueryColumn(col any) error {
	return fmt.Errorf("invalid subquery column: %v", col)
}

func ErrInvalidJoinCondition(cond any) error {
	return fmt.Errorf("invalid join condition: %v", cond)
}

func ErrInvalidTableReference(table any) error {
	return fmt.Errorf("invalid table reference: %v", table)
}

func ErrInvalidInsertValue(v any) error {
	return fmt.Errorf("invalid insert value: %v", v)
}

func ErrInvalidDialect(v any) error {
	return fmt.Errorf("invalid dialect: %v", v)
}

func ErrInvalidOrderBy(v any) error {
	return fmt.Errorf("invalid order by column: %v", v)
}

func ErrDialTimeout(duration string) error {
	return fmt.Errorf("orm: 连接数据库超时，超时时间 %s", duration)
}

func ErrHealthCheckFailed(reason string) error {
	return fmt.Errorf("orm: 连接健康检查失败: %s", reason)
}

func ErrCreateConnectionFailed(err error) error {
	return fmt.Errorf("orm: 创建数据库连接失败: %w", err)
}

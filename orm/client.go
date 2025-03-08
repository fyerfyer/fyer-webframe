package orm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
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

// CreateSchema 创建模式（表）
//func (c *Client) CreateSchema(ctx context.Context, model interface{}) error {
//	return nil
//}

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
package orm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/fyerfyer/fyer-webframe/orm/internal/ferr"
)

type Selector[T any] struct {
	builder *strings.Builder
	table   string
	model   *model
	args    []any
	layer   Layer
}

func RegisterSelector[T any](layer Layer) *Selector[T] {
	var val T

	var m *model
	switch layer := layer.(type) {
	case *DB:
		var err error
		m, err = layer.getModel(val)
		if err != nil {
			panic(err)
		}
	case *Tx:
		var err error
		m, err = layer.db.getModel(val)
		if err != nil {
			panic(err)
		}
	}

	// 处理表名
	if tablename, ok := any(val).(TableNamer); ok {
		m.table = tablename.TableName()
	}
	if tablename, ok := any(&val).(TableNamer); ok {
		m.table = tablename.TableName()
	}

	return &Selector[T]{
		builder: &strings.Builder{},
		model:   m,
		layer:   layer,
	}
}

func (s *Selector[T]) Select(cols ...Selectable) *Selector[T] {
	if cols == nil {
		s.builder.WriteString("SELECT * FROM " + "`" + s.model.table + "`")
		return s
	}

	s.builder.WriteString("SELECT ")
	for i := 0; i < len(cols); i++ {
		switch col := cols[i].(type) {
		case *Column:
			// 注入模型信息
			col.model = s.model
			col.Build(s.builder)
			if i != len(cols)-1 {
				s.builder.WriteByte(',')
			}
			s.builder.WriteByte(' ')
		case *Aggregate: // 修改类型断言
			col.model = s.model
			col.Build(s.builder)
			if i != len(cols)-1 {
				s.builder.WriteByte(',')
			}
			s.builder.WriteByte(' ')
		case RawExpr:
			col.Build(s.builder)
			s.builder.WriteByte(' ')
			s.args = append(s.args, col.args...)
		default:
			panic(ferr.ErrInvalidSelectable(col))
		}
	}

	s.builder.WriteString("FROM " + "`" + s.model.table + "`")
	return s
}

func (s *Selector[T]) Where(conditions ...Condition) *Selector[T] {
	s.builder.WriteString(" WHERE ")
	for i := 0; i < len(conditions); i++ {
		if pred, ok := conditions[i].(*Predicate); ok {
			pred.model = s.model
		}
		conditions[i].Build(s.builder, &s.args)
		if i != len(conditions)-1 {
			s.builder.WriteString(" AND ")
		}
	}
	return s
}

func (s *Selector[T]) Limit(num int) *Selector[T] {
	s.builder.WriteString(" LIMIT " + strconv.Itoa(num))
	return s
}

func (s *Selector[T]) Offset(num int) *Selector[T] {
	s.builder.WriteString(" OFFSET " + strconv.Itoa(num))
	return s
}

func (s *Selector[T]) GroupBy(cols ...Selectable) *Selector[T] {
	s.builder.WriteString(" GROUP BY ")
	if len(cols) > 1 {
		s.builder.WriteByte('(')
	}
	for i := 0; i < len(cols); i++ {
		switch col := cols[i].(type) {
		case *Column:
			// 注入模型信息
			col.model = s.model
			col.Build(s.builder)
			if i != len(cols)-1 {
				s.builder.WriteString(", ")
			}
		case *Aggregate:
			col.model = s.model
			col.Build(s.builder)
			if i != len(cols)-1 {
				s.builder.WriteString(", ")
			}
		default:
			panic(ferr.ErrInvalidSelectable(col))
		}
	}
	if len(cols) > 1 {
		s.builder.WriteByte(')')
	}
	return s
}

func (s *Selector[T]) OrderBy(orders ...OrderBy) *Selector[T] {
	if len(orders) == 0 {
		return s
	}

	s.builder.WriteString(" ORDER BY ")
	for i, order := range orders {
		if i > 0 {
			s.builder.WriteByte(',')
			s.builder.WriteByte(' ')
		}

		switch expr := order.expr.(type) {
		case *Column:
			// 如果是列引用，允许使用别名
			expr.model = s.model
			expr.allowAlias = true
			expr.Build(s.builder)
		case *Aggregate: // 修改类型断言
			expr.model = s.model
			expr.Build(s.builder)
		case RawExpr:
			expr.Build(s.builder)
			s.args = append(s.args, expr.args...)
		default:
			panic(ferr.ErrInvalidOrderBy(order.expr))
		}

		if order.desc {
			s.builder.WriteString(" DESC")
		}
	}
	return s
}

func (s *Selector[T]) Having(conditions ...Condition) *Selector[T] {
	if len(conditions) == 0 {
		return s
	}

	s.builder.WriteString(" HAVING ")
	for i, condition := range conditions {
		if i > 0 {
			s.builder.WriteString(" AND ")
		}

		if pred, ok := condition.(*Predicate); ok {
			pred.model = s.model
			switch left := pred.left.(type) {
			case *Column:
				// 注入模型信息并允许使用别名
				left.model = s.model
				left.allowAlias = true
			case *Aggregate: // 修改类型断言
				left.model = s.model
			}
		}

		condition.Build(s.builder, &s.args)
	}
	return s
}

func (s *Selector[T]) Build() (*Query, error) {
	s.builder.WriteByte(';')
	return &Query{
		SQL:  s.builder.String(),
		Args: s.args,
	}, nil
}

// scanRow 将一行数据扫描到结构体中
func (s *Selector[T]) scanRow(rows *sql.Rows) (*T, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	t := new(T)
	vals := make([]any, len(cols))
	// new返回的是指针
	valElem := reflect.ValueOf(t).Elem()

	for i, col := range cols {
		if fieldName, ok := s.model.colNameMap[col]; ok {
			field := valElem.FieldByName(fieldName)
			vals[i] = field.Addr().Interface()
		} else {
			var v any
			vals[i] = &v
		}
	}

	if err = rows.Scan(vals...); err != nil {
		return nil, err
	}

	return t, nil
}

// Get 获取单行数据
func (s *Selector[T]) Get(ctx context.Context) (*T, error) {
	q, err := s.Build()
	if err != nil {
		return nil, err
	}

	// 构建查询上下文
	qc := &QueryContext{
		QueryType: "query",
		Query:     q,
		Model:     s.model,
		Builder:   s,
	}

	// 确保 layer 初始化了 handler
	if s.layer.getHandler() == nil {
		return nil, fmt.Errorf("handler not initialized")
	}

	res, err := s.layer.HandleQuery(ctx, qc)
	if err != nil {
		return nil, err
	}

	rows := res.Rows
	defer rows.Close()

	if !rows.Next() {
		return nil, ferr.ErrNoRows
	}

	t, err := s.scanRow(rows)
	if err != nil {
		return nil, err
	}

	if rows.Next() {
		return nil, ferr.ErrTooManyRows
	}

	return t, nil
}

// GetMulti 获取多行数据
func (s *Selector[T]) GetMulti(ctx context.Context) ([]*T, error) {
	q, err := s.Build()
	if err != nil {
		return nil, err
	}

	qc := &QueryContext{
		QueryType: "query",
		Query:     q,
		Model:     s.model,
		Builder:   s,
	}

	res, err := s.layer.HandleQuery(ctx, qc)
	if err != nil {
		return nil, err
	}

	rows := res.Rows
	defer rows.Close()

	result := make([]*T, 0)
	for rows.Next() {
		t, err := s.scanRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, t)
	}

	return result, rows.Err()
}

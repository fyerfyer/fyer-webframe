package orm

import (
	"context"
	"database/sql"
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
	db      *DB
}

func RegisterSelector[T any](db *DB) *Selector[T] {
	var val T
	m, err := db.getModel(val)
	if err != nil {
		panic(err)
	}

	// 结构体或者结构体指针实现TableNameInterface接口即可
	if tablename, ok := any(val).(TableNamer); ok {
		m.table = tablename.TableName()
	}

	// 尝试取指针
	if tablename, ok := any(&val).(TableNamer); ok {
		m.table = tablename.TableName()
	}

	return &Selector[T]{
		builder: &strings.Builder{},
		model:   m,
		db:      db,
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
		case Aggregate:
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
		if pred, ok := conditions[i].(Predicate); ok {
			if col, ok := pred.left.(*Column); ok {
				// 注入模型信息
				col.model = s.model
			}
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
				s.builder.WriteByte(',')
			}
			s.builder.WriteByte(' ')
		default:
			panic(ferr.ErrInvalidSelectable(col))
		}
	}
	if len(cols) > 1 {
		s.builder.WriteByte(')')
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

	rows, err := s.db.sqlDB.QueryContext(ctx, q.SQL, q.Args...)
	if err != nil {
		return nil, err
	}
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

	rows, err := s.db.sqlDB.QueryContext(ctx, q.SQL, q.Args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*T
	for rows.Next() {
		t, err := s.scanRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, t)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

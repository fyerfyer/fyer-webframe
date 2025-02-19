package orm

import (
	"github.com/fyerfyer/fyer-webframe/orm/internal/ferr"
	"strconv"
	"strings"
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
	if tablename, ok := any(val).(TableNameInterface); ok {
		m.table = tablename.TableName()
	}

	// 尝试取指针
	if tablename, ok := any(&val).(TableNameInterface); ok {
		m.table = tablename.TableName()
	}

	return &Selector[T]{
		builder: &strings.Builder{},
		model:   m,
		db:      db,
	}
}

func (s *Selector[T]) Select(cols ...string) *Selector[T] {
	if cols == nil {
		s.builder.WriteString("SELECT * FROM " + "`" + s.model.table + "`")
	} else {
		s.builder.WriteString("SELECT ")
		for i := 0; i < len(cols); i++ {
			colName, ok := s.model.fieldsMap[cols[i]]
			if !ok {
				panic(ferr.ErrInvalidColumn(cols[i]))
			}
			s.builder.WriteString("`" + colName.colName + "`")
			if i != len(cols)-1 {
				s.builder.WriteByte(',')
			}
			s.builder.WriteByte(' ')
		}
		s.builder.WriteString("FROM " + "`" + s.model.table + "`")
	}
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

func (s *Selector[T]) Build() (*Query, error) {
	s.builder.WriteByte(';')
	return &Query{
		SQL:  s.builder.String(),
		Args: s.args,
	}, nil
}

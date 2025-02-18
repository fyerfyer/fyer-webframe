package orm

import (
	"github.com/fyerfyer/fyer-webframe/orm/utils"
	"strconv"
	"strings"
)

type Selector[T any] struct {
	builder *strings.Builder
	table   string
	args    []any
	orderBy *OrderBy
}

type OrderBy struct {
	col  string
	desc bool
}

func NewSelector[T any]() *Selector[T] {

	tableName := utils.GetTableName[T]()

	return &Selector[T]{
		builder: &strings.Builder{},
		table:   tableName,
	}
}

func (s *Selector[T]) Select(args ...string) *Selector[T] {
	if args == nil {
		s.builder.WriteString("SELECT * FROM " + "`" + s.table + "`")
	} else {
		s.builder.WriteString("SELECT ")
		for i := 0; i < len(args); i++ {
			s.builder.WriteString("`" + args[i] + "`")
			if i != len(args)-1 {
				s.builder.WriteByte(',')
			}
			s.builder.WriteByte(' ')
		}
		s.builder.WriteString("FROM " + "`" + s.table + "`")
	}
	return s
}

func (s *Selector[T]) Where(conditions ...Condition) *Selector[T] {
	s.builder.WriteString(" WHERE ")
	for i := 0; i < len(conditions); i++ {
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

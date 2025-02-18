package orm

import (
	"github.com/fyerfyer/fyer-webframe/orm/utils"
	"strconv"
	"strings"
)

type Deleter[T any] struct {
	builder *strings.Builder
	table   string
	args    []any
}

func NewDeleter[T any]() *Deleter[T] {
	tableName := utils.GetTableName[T]()
	return &Deleter[T]{
		builder: &strings.Builder{},
		table:   tableName,
	}
}

func (d *Deleter[T]) Delete(args ...string) *Deleter[T] {
	if args == nil {
		d.builder.WriteString("DELETE * FROM " + "`" + d.table + "`")
	} else {
		d.builder.WriteString("SELECT ")
		for i := 0; i < len(args); i++ {
			d.builder.WriteString("`" + args[i] + "`")
			if i != len(args)-1 {
				d.builder.WriteByte(',')
			}
			d.builder.WriteByte(' ')
		}
		d.builder.WriteString("FROM " + "`" + d.table + "`")
	}
	return d
}

func (d *Deleter[T]) Where(conditions ...Condition) *Deleter[T] {
	d.builder.WriteString(" WHERE ")
	for i := 0; i < len(conditions); i++ {
		conditions[i].Build(d.builder, &d.args)
		if i != len(conditions)-1 {
			d.builder.WriteString(" AND ")
		}
	}
	return d
}

func (d *Deleter[T]) Limit(num int) *Deleter[T] {
	d.builder.WriteString(" LIMIT " + strconv.Itoa(num))
	return d
}

func (d *Deleter[T]) Offset(num int) *Deleter[T] {
	d.builder.WriteString(" OFFSET " + strconv.Itoa(num))
	return d
}

func (d *Deleter[T]) Build() (*Query, error) {
	d.builder.WriteByte(';')
	return &Query{
		SQL:  d.builder.String(),
		Args: d.args,
	}, nil
}

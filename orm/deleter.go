package orm

import (
	"github.com/fyerfyer/fyer-webframe/orm/internal/ferr"
	"strconv"
	"strings"
)

type Deleter[T any] struct {
	builder *strings.Builder
	model   *model
	args    []any
	db      *DB
}

func NewDeleter[T any](db *DB) *Deleter[T] {
	var val T
	m, err := db.getModel(val)
	if err != nil {
		panic(err)
	}

	return &Deleter[T]{
		builder: &strings.Builder{},
		model:   m,
		db:      db,
	}
}

func (d *Deleter[T]) Delete(cols ...string) *Deleter[T] {
	// 检查字段是否存在
	if len(cols) > 0 {
		for _, col := range cols {
			if _, ok := d.model.fieldsMap[col]; !ok {
				panic(ferr.ErrInvalidColumn(col))
			}
		}
	}

	if cols == nil {
		d.builder.WriteString("DELETE * FROM " + "`" + d.model.table + "`")
	} else {
		d.builder.WriteString("SELECT ")
		for i := 0; i < len(cols); i++ {
			d.builder.WriteString("`" + cols[i] + "`")
			if i != len(cols)-1 {
				d.builder.WriteByte(',')
			}
			d.builder.WriteByte(' ')
		}
		d.builder.WriteString("FROM " + "`" + d.model.table + "`")
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

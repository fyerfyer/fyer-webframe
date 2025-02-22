package orm

import (
	"context"
	"github.com/fyerfyer/fyer-webframe/orm/internal/ferr"
	"strconv"
	"strings"
)

type Deleter[T any] struct {
	builder *strings.Builder
	model   *model
	args    []any
	layer   Layer
}

func RegisterDeleter[T any](layer Layer) *Deleter[T] {
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

	return &Deleter[T]{
		builder: &strings.Builder{},
		model:   m,
		layer:   layer,
	}
}

func (d *Deleter[T]) Delete(cols ...Selectable) *Deleter[T] {
	if cols == nil {
		d.builder.WriteString("DELETE FROM " + "`" + d.model.table + "`")
		return d
	}

	d.builder.WriteString("DELETE ")
	for i := 0; i < len(cols); i++ {
		switch col := cols[i].(type) {
		case *Column:
			// 注入模型信息
			col.model = d.model
			col.Build(d.builder)
			if i != len(cols)-1 {
				d.builder.WriteByte(',')
			}
			d.builder.WriteByte(' ')
		case *Aggregate:
			col.Build(d.builder)
			if i != len(cols)-1 {
				d.builder.WriteByte(',')
			}
			d.builder.WriteByte(' ')
		case RawExpr:
			col.Build(d.builder)
			d.builder.WriteByte(' ')
			d.args = append(d.args, col.args...)
		default:
			panic(ferr.ErrInvalidSelectable(col))
		}
	}

	d.builder.WriteString("FROM " + "`" + d.model.table + "`")
	return d
}

func (d *Deleter[T]) Where(conditions ...Condition) *Deleter[T] {
	d.builder.WriteString(" WHERE ")
	for i := 0; i < len(conditions); i++ {
		if pred, ok := conditions[i].(*Predicate); ok {
			if col, ok := pred.left.(*Column); ok {
				// 注入模型信息
				col.model = d.model
			}
		}
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

// Exec 添加执行方法
func (d *Deleter[T]) Exec(ctx context.Context) (Result, error) {
	q, err := d.Build()
	if err != nil {
		return Result{}, err
	}

	qc := &QueryContext{
		QueryType: "exec",
		Query:     q,
		Model:     d.model,
		Builder:   d,
	}

	res, err := d.layer.HandleQuery(ctx, qc)
	return Result{
		res: res.Result.res,
		err: err,
	}, err
}

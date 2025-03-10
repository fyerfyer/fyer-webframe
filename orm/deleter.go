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
	dialect Dialect

	// 缓存相关字段
	invalidateCache bool     // 是否使缓存失效
	invalidateTags  []string // 要失效的缓存标签
}

// WithInvalidateCache 设置是否使相关缓存失效
func (d *Deleter[T]) WithInvalidateCache() *Deleter[T] {
	d.invalidateCache = true
	return d
}

// WithInvalidateTags 设置要使失效的缓存标签
func (d *Deleter[T]) WithInvalidateTags(tags ...string) *Deleter[T] {
	d.invalidateCache = true
	d.invalidateTags = tags
	return d
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

	dialect := layer.getDB().dialect
	m.dialect = dialect
	m.index = 1

	return &Deleter[T]{
		builder: &strings.Builder{},
		model:   m,
		dialect: dialect,
		layer:   layer,
	}
}

func (d *Deleter[T]) Delete(cols ...Selectable) *Deleter[T] {
	if cols == nil {
		d.builder.WriteString("DELETE FROM ")
		d.builder.WriteString(d.dialect.Quote(d.model.table))
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

	d.builder.WriteString("FROM ")
	d.builder.WriteString(d.dialect.Quote(d.model.table))
	return d
}

func (d *Deleter[T]) Where(conditions ...Condition) *Deleter[T] {
	d.builder.WriteString(" WHERE ")
	for i := 0; i < len(conditions); i++ {
		if pred, ok := conditions[i].(*Predicate); ok {
			pred.model = d.model
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

// Exec 添加了缓存失效逻辑
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

	// 如果执行成功且需要使缓存失效
	if err == nil && d.invalidateCache {
		db := d.layer.getDB()
		if db.cacheManager != nil && db.cacheManager.IsEnabled() {
			modelName := d.model.GetTableName()
			// 传入标签或使用模型的默认标签
			_ = db.cacheManager.InvalidateCache(ctx, modelName, d.invalidateTags...)
		}
	}

	return Result{
		res: res.Result.res,
		err: err,
	}, err
}

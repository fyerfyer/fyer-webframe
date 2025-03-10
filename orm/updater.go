package orm

import (
	"context"
	"strconv"
	"strings"
)

// Updater 实现更新操作的构建器
type Updater[T any] struct {
	builder     *strings.Builder
	model       *model
	args        []any
	layer       Layer
	dialect     Dialect
	hasSet      bool
	setCnt      int
	tableName   string        // 用于分片时替换表名

	// 缓存相关字段
	invalidateCache bool     // 是否使缓存失效
	invalidateTags  []string // 要失效的缓存标签
}

// WithInvalidateCache 设置是否使相关缓存失效
func (u *Updater[T]) WithInvalidateCache() *Updater[T] {
	u.invalidateCache = true
	return u
}

// WithInvalidateTags 设置要使失效的缓存标签
func (u *Updater[T]) WithInvalidateTags(tags ...string) *Updater[T] {
	u.invalidateCache = true
	u.invalidateTags = tags
	return u
}

// RegisterUpdater 创建一个新的更新构建器
func RegisterUpdater[T any](layer Layer) *Updater[T] {
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

	dialect := layer.getDB().dialect
	m.dialect = dialect
	m.index = 1

	return &Updater[T]{
		builder: &strings.Builder{},
		model:   m,
		dialect: dialect,
		layer:   layer,
	}
}

// Update 开始构建更新语句
func (u *Updater[T]) Update() *Updater[T] {
	u.builder.WriteString("UPDATE ")
	table := u.model.table
	if u.tableName != "" {
		table = u.tableName
	}
	u.builder.WriteString(u.dialect.Quote(table))
	return u
}

// Set 设置要更新的字段和值
func (u *Updater[T]) Set(col *Column, val any) *Updater[T] {
	u.setCnt ++
	if u.setCnt > 1 {
		u.builder.WriteString(", ")
	}
	u.setClauses([]*Column{col}, []any{val})
	return u
}

// SetMulti 批量设置多个字段和值
func (u *Updater[T]) SetMulti(vals map[string]any) *Updater[T] {
	u.setCnt = 0

	var (
		cols []*Column
		values []any
	)

	for colName, val := range vals {
		cols = append(cols, &Column{name: colName})
		values = append(values, val)
	}

	u.setClauses(cols, values)
	return u
}

func (u *Updater[T]) setClauses(cols []*Column, vals []any) *Updater[T] {
	if !u.hasSet{
		u.hasSet = true
		u.builder.WriteString(" SET ")
	}

	if len(cols) != len(vals) {
		panic("columns and values length mismatch")
	}

	for i, _ := range cols {
		if i > 0 {
			u.builder.WriteString(", ")
		}
		col := cols[i]
		col.model = u.model
		col.Build(u.builder)

		// 构建赋值操作
		u.builder.WriteString(" = ")
		val := vals[i]

		switch val := val.(type) {
		case Expression:
			// 如果是表达式，递归构建
			switch expr := val.(type) {
			case *Column:
				expr.model = u.model
				expr.Build(u.builder)
			case *Predicate:
				expr.model = u.model
				expr.Build(u.builder, &u.args)
			case *Aggregate:
				expr.model = u.model
				expr.Build(u.builder)
			case RawExpr:
				expr.Build(u.builder)
				u.args = append(u.args, expr.args...)
			default:
				u.builder.WriteString(u.dialect.Placeholder(u.model.index))
				u.model.index++
				u.args = append(u.args, val)
			}
		default:
			// 普通值，添加占位符
			u.builder.WriteString(u.dialect.Placeholder(u.model.index))
			u.model.index++
			u.args = append(u.args, val)
		}
	}
	return u
}

// Where 添加条件子句
func (u *Updater[T]) Where(conditions ...Condition) *Updater[T] {
	u.setCnt = 0
	u.builder.WriteString(" WHERE ")
	for i := 0; i < len(conditions); i++ {
		if pred, ok := conditions[i].(*Predicate); ok {
			pred.model = u.model
		}
		conditions[i].Build(u.builder, &u.args)
		if i != len(conditions)-1 {
			u.builder.WriteString(" AND ")
		}
	}
	return u
}

// Limit 限制更新的行数
func (u *Updater[T]) Limit(num int) *Updater[T] {
	u.setCnt = 0
	u.builder.WriteString(" LIMIT " + strconv.Itoa(num))
	return u
}

// Build 构建SQL查询
func (u *Updater[T]) Build() (*Query, error) {
	if !u.hasSet {
		panic("no set clause")
	}
	u.builder.WriteByte(';')
	return &Query{
		SQL:  u.builder.String(),
		Args: u.args,
	}, nil
}

// Exec 执行更新操作
func (u *Updater[T]) Exec(ctx context.Context) (Result, error) {
	q, err := u.Build()
	if err != nil {
		return Result{}, err
	}

	qc := &QueryContext{
		QueryType: "exec",
		Query:     q,
		Model:     u.model,
		Builder:   u,
	}

	res, err := u.layer.HandleQuery(ctx, qc)
	// 如果执行成功且需要使缓存失效
	if err == nil && u.invalidateCache {
		// 获取数据库实例
		db := u.layer.getDB()
		// 如果DB有缓存管理器，则使相关缓存失效
		if db != nil && db.cacheManager != nil {
			modelName := u.model.GetTableName()
			if len(u.invalidateTags) > 0 {
				// 如果指定了标签，使用标签使缓存失效
				_ = db.cacheManager.cache.DeleteByTags(ctx, u.invalidateTags...)
			} else {
				// 否则使用模型名作为标签使缓存失效
				_ = db.cacheManager.cache.DeleteByTags(ctx, modelName)
			}
		}
	}

	return Result{
		res: res.Result.res,
		err: err,
	}, err
}
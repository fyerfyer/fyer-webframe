package orm

import (
	"context"
	"github.com/fyerfyer/fyer-webframe/orm/internal/ferr"
	"reflect"
	"strings"
)

type Inserter[T any] struct {
	builder *strings.Builder
	values  []any
	model   *model
	dialect Dialect
	layer   Layer

	// 缓存相关字段
	invalidateCache bool     // 是否使缓存失效
	invalidateTags  []string // 要失效的缓存标签
}

// WithInvalidateCache 设置是否使相关缓存失效
func (i *Inserter[T]) WithInvalidateCache() *Inserter[T] {
	i.invalidateCache = true
	return i
}

// WithInvalidateTags 设置要使失效的缓存标签
func (i *Inserter[T]) WithInvalidateTags(tags ...string) *Inserter[T] {
	i.invalidateCache = true
	i.invalidateTags = tags
	return i
}

func RegisterInserter[T any](layer Layer) *Inserter[T] {
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

	// 结构体或者结构体指针实现TableNamer接口即可
	if tablename, ok := any(val).(TableNamer); ok {
		m.table = tablename.TableName()
	}

	// 尝试取指针
	if tablename, ok := any(&val).(TableNamer); ok {
		m.table = tablename.TableName()
	}

	dialect := layer.getDB().dialect
	m.dialect = dialect
	m.index = 1

	return &Inserter[T]{
		builder: &strings.Builder{},
		model:   m,
		dialect: dialect,
		layer:   layer,
	}
}

// Insert 支持指定列的插入
func (i *Inserter[T]) Insert(cols []string, vals ...*T) *Inserter[T] {
	if vals == nil || len(vals) == 0 {
		panic(ferr.ErrInsertRowNotFound)
	}

	i.builder.WriteString("INSERT INTO ")
	i.builder.WriteString(i.dialect.Quote(i.model.table) + " ")

	colsString := strings.Builder{}
	placeholders := strings.Builder{}
	basePlaceHolders := strings.Builder{}

	// 使用cols来确定要插入的列
	fields := make([]string, 0, len(cols))
	if len(cols) > 0 {
		// 使用指定的列
		for _, colName := range cols {
			fields = append(fields, colName)
		}
	} else {
		// 使用全部列
		typ := reflect.TypeOf(vals[0]).Elem()
		for j := 0; j < typ.NumField(); j++ {
			fields = append(fields, typ.Field(j).Name)
		}
	}

	// 构建列名部分
	colsString.WriteByte('(')
	basePlaceHolders.WriteByte('(')
	for idx, fieldName := range fields {
		col, ok := i.model.fieldsMap[fieldName]
		if !ok {
			panic(ferr.ErrInvalidColumn(fieldName))
		}
		colsString.WriteString(i.dialect.Quote(col.colName))
		basePlaceHolders.WriteString(i.dialect.Placeholder(i.model.index))
		i.model.index ++
		if idx != len(fields)-1 {
			colsString.WriteString(", ")
			basePlaceHolders.WriteString(", ")
		}
	}
	colsString.WriteByte(')')
	basePlaceHolders.WriteByte(')')

	// 构建值部分
	for index, val := range vals {
		v := reflect.ValueOf(val).Elem()
		placeholders.WriteString(basePlaceHolders.String())
		if index != len(vals)-1 {
			placeholders.WriteString(", ")
		}

		// 只取指定列的值
		for _, fieldName := range fields {
			valField := v.FieldByName(fieldName)
			i.values = append(i.values, valField.Interface())
		}
	}

	i.builder.WriteString(colsString.String())
	i.builder.WriteString(" VALUES ")
	i.builder.WriteString(placeholders.String())
	return i
}

func (i *Inserter[T]) Upsert(conflictCols []*Column, cols []*Column) *Inserter[T] {
	db := i.layer.getDB()

	dialect, ok := db.dialect.(interface {
		BuildUpsert(builder *strings.Builder, conflictCols []*Column, cols []*Column)
		setModel(m *model)
	})
	if !ok {
		panic(ferr.ErrInvalidDialect(db.dialect))
	}

	// 注入模型信息
	dialect.setModel(i.model)
	dialect.BuildUpsert(i.builder, conflictCols, cols)
	return i
}

func (i *Inserter[T]) Build() (*Query, error) {
	i.builder.WriteByte(';')

	return &Query{
		SQL:  i.builder.String(),
		Args: i.values,
	}, nil
}

// Exec 添加了缓存失效逻辑
func (i *Inserter[T]) Exec(ctx context.Context) (Result, error) {
	q, err := i.Build()
	if err != nil {
		return Result{}, err
	}

	qc := &QueryContext{
		QueryType: "exec",
		Query:     q,
		Model:     i.model,
		Builder:   i,
	}

	res, err := i.layer.HandleQuery(ctx, qc)

	// 如果执行成功且需要使缓存失效
	if err == nil && i.invalidateCache {
		db := i.layer.getDB()
		if db.cacheManager != nil && db.cacheManager.IsEnabled() {
			modelName := i.model.GetTableName()
			// 传入标签或使用模型的默认标签
			_ = db.cacheManager.InvalidateCache(ctx, modelName, i.invalidateTags...)
		}
	}

	return Result{
		res: res.Result.res,
		err: err,
	}, err
}
package orm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/fyerfyer/fyer-webframe/orm/internal/utils"
)

// Collection 代表对特定模型类型的操作集合
type Collection struct {
	client    *Client
	modelType interface{}
	modelName string
}

// Find 查找单个记录
func (c *Collection) Find(ctx context.Context, where ...Condition) (interface{}, error) {
	// 获取数据库和模型信息
	db := c.client.GetDB()
	m, err := db.getModel(c.modelType)
	if err != nil {
		return nil, err
	}

	// 手动构建SQL
	builder := &strings.Builder{}
	args := make([]any, 0)

	builder.WriteString("SELECT * FROM ")
	builder.WriteString(db.dialect.Quote(m.table))

	if len(where) > 0 {
		builder.WriteString(" WHERE ")
		for i, cond := range where {
			if pred, ok := cond.(*Predicate); ok {
				pred.model = m
			}
			cond.Build(builder, &args)
			if i < len(where)-1 {
				builder.WriteString(" AND ")
			}
		}
	}

	builder.WriteString(";")
	query := builder.String()

	// 执行查询
	rows, err := db.queryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 处理结果
	if !rows.Next() {
		return nil, sql.ErrNoRows
	}

	// 创建结果实例
	result := reflect.New(reflect.TypeOf(c.modelType).Elem()).Interface()

	// 获取列信息
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// 准备扫描到目标结构体
	values := make([]interface{}, len(cols))
	resultVal := reflect.ValueOf(result).Elem()

	for i, col := range cols {
		// 根据列名找到对应的结构体字段
		if fieldName, ok := m.colNameMap[col]; ok {
			field := resultVal.FieldByName(fieldName)
			if field.IsValid() && field.CanAddr() {
				values[i] = field.Addr().Interface()
			} else {
				// 如果找不到对应字段，使用一个占位符
				var placeholder interface{}
				values[i] = &placeholder
			}
		} else {
			var placeholder interface{}
			values[i] = &placeholder
		}
	}

	// 扫描数据
	if err := rows.Scan(values...); err != nil {
		return nil, err
	}

	return result, nil
}

// FindAll 查找所有匹配的记录
func (c *Collection) FindAll(ctx context.Context, where ...Condition) ([]interface{}, error) {
	// 获取数据库和模型信息
	db := c.client.GetDB()
	m, err := db.getModel(c.modelType)
	if err != nil {
		return nil, err
	}

	// 手动构建SQL
	builder := &strings.Builder{}
	args := make([]any, 0)

	builder.WriteString("SELECT * FROM ")
	builder.WriteString(db.dialect.Quote(m.table))

	if len(where) > 0 {
		builder.WriteString(" WHERE ")
		for i, cond := range where {
			if pred, ok := cond.(*Predicate); ok {
				pred.model = m
			}
			cond.Build(builder, &args)
			if i < len(where)-1 {
				builder.WriteString(" AND ")
			}
		}
	}

	builder.WriteString(";")
	query := builder.String()

	// 执行查询
	rows, err := db.queryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 获取列信息
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// 准备结果集
	var results []interface{}
	modelType := reflect.TypeOf(c.modelType).Elem()

	// 处理每一行数据
	for rows.Next() {
		// 创建新的结构体实例
		result := reflect.New(modelType).Interface()
		resultVal := reflect.ValueOf(result).Elem()

		// 准备扫描目标
		values := make([]interface{}, len(cols))
		for i, col := range cols {
			if fieldName, ok := m.colNameMap[col]; ok {
				field := resultVal.FieldByName(fieldName)
				if field.IsValid() && field.CanAddr() {
					values[i] = field.Addr().Interface()
				} else {
					var placeholder interface{}
					values[i] = &placeholder
				}
			} else {
				var placeholder interface{}
				values[i] = &placeholder
			}
		}

		// 扫描数据
		if err := rows.Scan(values...); err != nil {
			return nil, err
		}

		results = append(results, result)
	}

	// 检查行处理错误
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// Insert 插入记录
func (c *Collection) Insert(ctx context.Context, model interface{}) (Result, error) {
	// 获取数据库和模型信息
	db := c.client.GetDB()
	m, err := db.getModel(c.modelType)
	if err != nil {
		return Result{}, err
	}

	// 检查传入模型类型是否匹配
	modelType := reflect.TypeOf(c.modelType).Elem()
	inputType := reflect.TypeOf(model)
	if inputType.Kind() == reflect.Ptr {
		inputType = inputType.Elem()
	}
	if inputType != modelType {
		return Result{}, fmt.Errorf("model type mismatch: expected %s, got %s", modelType.Name(), inputType.Name())
	}

	// 构建插入SQL
	builder := &strings.Builder{}
	args := make([]any, 0)
	modelVal := reflect.ValueOf(model)
	if modelVal.Kind() == reflect.Ptr {
		modelVal = modelVal.Elem()
	}

	builder.WriteString("INSERT INTO ")
	builder.WriteString(db.dialect.Quote(m.table))
	builder.WriteString(" (")

	// 构建列名部分
	i := 0
	fieldNames := make([]string, 0, len(m.fieldsMap))
	for fieldName, field := range m.fieldsMap {
		builder.WriteString(db.dialect.Quote(field.colName))
		fieldNames = append(fieldNames, fieldName)
		if i < len(m.fieldsMap)-1 {
			builder.WriteString(", ")
		}
		i++
	}

	// 构建值部分
	builder.WriteString(") VALUES (")
	for i, fieldName := range fieldNames {
		builder.WriteString(db.dialect.Placeholder(i + 1))
		if i < len(fieldNames)-1 {
			builder.WriteString(", ")
		}

		// 获取字段值
		fieldVal := modelVal.FieldByName(fieldName)
		if fieldVal.IsValid() {
			args = append(args, fieldVal.Interface())
		} else {
			args = append(args, nil)
		}
	}
	builder.WriteString(");")

	// 执行插入
	result, err := db.execContext(ctx, builder.String(), args...)
	return Result{res: result}, err
}

// Update 更新记录
func (c *Collection) Update(ctx context.Context, update map[string]interface{}, where ...Condition) (Result, error) {
	// 获取数据库和模型信息
	db := c.client.GetDB()
	m, err := db.getModel(c.modelType)
	if err != nil {
		return Result{}, err
	}

	// 构建更新SQL
	builder := &strings.Builder{}
	args := make([]any, 0, len(update)+len(where))

	builder.WriteString("UPDATE ")
	builder.WriteString(db.dialect.Quote(m.table))
	builder.WriteString(" SET ")

	// 构建SET部分
	i := 0
	for fieldName, value := range update {
		field, ok := m.fieldsMap[fieldName]
		if !ok {
			// 尝试使用蛇形命名法
			snakeFieldName := utils.CamelToSnake(fieldName)
			for _, f := range m.fieldsMap {
				if f.colName == snakeFieldName {
					field = f
					ok = true
					break
				}
			}
			if !ok {
				return Result{}, fmt.Errorf("unknown field: %s", fieldName)
			}
		}

		builder.WriteString(db.dialect.Quote(field.colName))
		builder.WriteString(" = ")
		builder.WriteString(db.dialect.Placeholder(i + 1))
		args = append(args, value)

		if i < len(update)-1 {
			builder.WriteString(", ")
		}
		i++
	}

	// 构建WHERE部分
	if len(where) > 0 {
		builder.WriteString(" WHERE ")
		for i, cond := range where {
			if pred, ok := cond.(*Predicate); ok {
				pred.model = m
			}
			cond.Build(builder, &args)
			if i < len(where)-1 {
				builder.WriteString(" AND ")
			}
		}
	}

	builder.WriteString(";")

	// 执行更新
	result, err := db.execContext(ctx, builder.String(), args...)
	return Result{res: result}, err
}

// Delete 删除记录
func (c *Collection) Delete(ctx context.Context, where ...Condition) (Result, error) {
	// 获取数据库和模型信息
	db := c.client.GetDB()
	m, err := db.getModel(c.modelType)
	if err != nil {
		return Result{}, err
	}

	// 构建删除SQL
	builder := &strings.Builder{}
	args := make([]any, 0)

	builder.WriteString("DELETE FROM ")
	builder.WriteString(db.dialect.Quote(m.table))

	// 构建WHERE部分
	if len(where) > 0 {
		builder.WriteString(" WHERE ")
		for i, cond := range where {
			if pred, ok := cond.(*Predicate); ok {
				pred.model = m
			}
			cond.Build(builder, &args)
			if i < len(where)-1 {
				builder.WriteString(" AND ")
			}
		}
	}

	builder.WriteString(";")

	// 执行删除
	result, err := db.execContext(ctx, builder.String(), args...)
	return Result{res: result}, err
}

// FindWithOptions 使用选项查找记录
func (c *Collection) FindWithOptions(ctx context.Context, opts FindOptions, where ...Condition) ([]interface{}, error) {
	// 获取数据库和模型信息
	db := c.client.GetDB()
	m, err := db.getModel(c.modelType)
	if err != nil {
		return nil, err
	}

	// 手动构建SQL
	builder := &strings.Builder{}
	args := make([]any, 0)

	builder.WriteString("SELECT * FROM ")
	builder.WriteString(db.dialect.Quote(m.table))

	// 构建WHERE部分
	if len(where) > 0 {
		builder.WriteString(" WHERE ")
		for i, cond := range where {
			if pred, ok := cond.(*Predicate); ok {
				pred.model = m
			}
			cond.Build(builder, &args)
			if i < len(where)-1 {
				builder.WriteString(" AND ")
			}
		}
	}

	// 添加ORDER BY
	if len(opts.OrderBy) > 0 {
		builder.WriteString(" ORDER BY ")
		for i, order := range opts.OrderBy {
			switch expr := order.expr.(type) {
			case *Column:
				expr.model = m
				expr.Build(builder)
			default:
				return nil, errors.New("unsupported order by expression")
			}

			if order.desc {
				builder.WriteString(" DESC")
			} else {
				builder.WriteString(" ASC")
			}

			if i < len(opts.OrderBy)-1 {
				builder.WriteString(", ")
			}
		}
	}

	// 添加LIMIT
	if opts.Limit > 0 {
		builder.WriteString(" LIMIT ")
		builder.WriteString(fmt.Sprintf("%d", opts.Limit))
	}

	// 添加OFFSET
	if opts.Offset > 0 {
		builder.WriteString(" OFFSET ")
		builder.WriteString(fmt.Sprintf("%d", opts.Offset))
	}

	builder.WriteString(";")
	query := builder.String()

	// 执行查询
	rows, err := db.queryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 获取列信息
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// 准备结果集
	var results []interface{}
	modelType := reflect.TypeOf(c.modelType).Elem()

	// 处理每一行数据
	for rows.Next() {
		// 创建新的结构体实例
		result := reflect.New(modelType).Interface()
		resultVal := reflect.ValueOf(result).Elem()

		// 准备扫描目标
		values := make([]interface{}, len(cols))
		for i, col := range cols {
			if fieldName, ok := m.colNameMap[col]; ok {
				field := resultVal.FieldByName(fieldName)
				if field.IsValid() && field.CanAddr() {
					values[i] = field.Addr().Interface()
				} else {
					var placeholder interface{}
					values[i] = &placeholder
				}
			} else {
				var placeholder interface{}
				values[i] = &placeholder
			}
		}

		// 扫描数据
		if err := rows.Scan(values...); err != nil {
			return nil, err
		}

		results = append(results, result)
	}

	// 检查行处理错误
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
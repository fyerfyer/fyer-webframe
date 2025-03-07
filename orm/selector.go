package orm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unsafe"

	"github.com/fyerfyer/fyer-webframe/orm/internal/ferr"
)

var sqlWithFrom = ""

type Selector[T any] struct {
	builder       *strings.Builder
	model         *model
	dialect       Dialect
	subqueryCache *map[string]map[string]bool // 子查询缓存，只需要查询列名是否存在即可
	cols          []string                    // 查询列，用于构建子查询缓存
	delayCols     []*Column                   // 延迟处理的子查询列
	args          []any
	layer         Layer
}

func RegisterSelector[T any](layer Layer) *Selector[T] {
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

	return &Selector[T]{
		builder: &strings.Builder{},
		model:   m,
		layer:   layer,
		dialect: dialect,
	}
}

func (s *Selector[T]) Select(cols ...Selectable) *Selector[T] {
	sqlWithFrom = "FROM " + s.dialect.Quote(s.model.table)
	if cols == nil {
		s.builder.WriteString("SELECT * ")
		s.builder.WriteString(sqlWithFrom)
		return s
	}

	s.builder.WriteString("SELECT ")
	for i := 0; i < len(cols); i++ {
		switch col := cols[i].(type) {
		case *Column:
			// 如果是列引用，则需要解析并传入对应结构体
			// 注意：子查询传入的是字符串、并且col的table名称已经设置好，这种情况不需要解析，等到延迟验证那步再验证就行
			if col.table == "" {
				if col.tableStruct != nil {
					var err error
					col.fromModel, err = s.layer.getModel(col.tableStruct)
					if err != nil {
						panic(err)
					}
					col.table = col.fromModel.table
				} else {
					// 注入模型信息
					col.model = s.model
				}
			}
			col.Build(s.builder)
			if col.alias != "" {
				s.cols = append(s.cols, col.alias)
			} else {
				s.cols = append(s.cols, col.name)
			}
			if col.shouldDelay {
				s.delayCols = append(s.delayCols, col)
			}
			if i != len(cols)-1 {
				s.builder.WriteByte(',')
			}
			s.builder.WriteByte(' ')
		case *Aggregate: // 修改类型断言
			col.model = s.model
			col.Build(s.builder)
			if col.alias != "" {
				s.cols = append(s.cols, col.alias)
			}
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

	s.builder.WriteString(sqlWithFrom)
	return s
}

func (s *Selector[T]) From(table any) *Selector[T] {
	if sqlWithFrom != "" {
		sqlWithoutFrom := strings.TrimSuffix(s.builder.String(), sqlWithFrom)
		s.builder.Reset()
		s.builder.WriteString(sqlWithoutFrom)
	}
	switch table := table.(type) {

	// 传入字符串的话只有一种可能性：别名
	case string:
		return s.from(&Value{val: table})
	case TableReference:
		return s.from(table)
	default:
		panic(ferr.ErrInvalidTableReference(table))
	}
}

func (s *Selector[T]) from(table TableReference) *Selector[T] {
	s.builder.WriteString("FROM ")
	switch table := table.(type) {
	case *SubQuery[T]:
		table.Build(s.builder, &s.args)
	case *Join:
		table.Build(s.builder, &s.args)
	case *Value:
		s.builder.WriteString(s.dialect.Quote(table.val.(string)))
	}
	return s
}

func (s *Selector[T]) Where(conditions ...Condition) *Selector[T] {
	s.builder.WriteString(" WHERE ")
	for i := 0; i < len(conditions); i++ {
		if pred, ok := conditions[i].(*Predicate); ok {
			pred.model = s.model
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
			if col.shouldDelay {
				s.delayCols = append(s.delayCols, col)
			}
			if i != len(cols)-1 {
				s.builder.WriteString(", ")
			}
		case *Aggregate:
			col.model = s.model
			col.Build(s.builder)
			if i != len(cols)-1 {
				s.builder.WriteString(", ")
			}
		default:
			panic(ferr.ErrInvalidSelectable(col))
		}
	}
	if len(cols) > 1 {
		s.builder.WriteByte(')')
	}
	return s
}

func (s *Selector[T]) OrderBy(orders ...OrderBy) *Selector[T] {
	if len(orders) == 0 {
		return s
	}

	s.builder.WriteString(" ORDER BY ")
	for i, order := range orders {
		if i > 0 {
			s.builder.WriteByte(',')
			s.builder.WriteByte(' ')
		}

		switch expr := order.expr.(type) {
		case *Column:
			// 如果是列引用，允许使用别名
			expr.model = s.model
			expr.allowAlias = true
			expr.Build(s.builder)
		case *Aggregate: // 修改类型断言
			expr.model = s.model
			expr.Build(s.builder)
		case RawExpr:
			expr.Build(s.builder)
			s.args = append(s.args, expr.args...)
		default:
			panic(ferr.ErrInvalidOrderBy(order.expr))
		}

		if order.desc {
			s.builder.WriteString(" DESC")
		}
	}
	return s
}

func (s *Selector[T]) Having(conditions ...Condition) *Selector[T] {
	if len(conditions) == 0 {
		return s
	}

	s.builder.WriteString(" HAVING ")
	for i, condition := range conditions {
		if i > 0 {
			s.builder.WriteString(" AND ")
		}

		if pred, ok := condition.(*Predicate); ok {
			pred.model = s.model
			switch left := pred.left.(type) {
			case *Column:
				// 注入模型信息并允许使用别名
				left.model = s.model
				left.allowAlias = true
			case *Aggregate: // 修改类型断言
				left.model = s.model
			}
		}

		condition.Build(s.builder, &s.args)
	}
	return s
}

func (s *Selector[T]) Join(joinType JoinType, target TableReference) *Selector[T] {
	join := &Join{
		JoinType: string(joinType),
		Target:   target,
	}

	res := join.Build(s.builder, &s.args)
	if queryCache, ok := res.(*map[string]map[string]bool); ok {
		s.subqueryCache = queryCache
	}
	return s
}

func (s *Selector[T]) On(conditions ...Condition) *Selector[T] {
	s.builder.WriteString(" ON ")
	for index, condition := range conditions {
		switch cond := condition.(type) {
		case *Predicate:
			cond.model = s.model

			// 在build之前先做一些处理
			// 如果左边或者右边有FromTable的column的话，先给它注入模型信息
			// 其实这个逻辑也可以放到build里面，但是我不想把db注入到model，感觉很奇怪
			if leftCol, ok := cond.left.(*Column); ok {
				if leftCol.tableStruct != nil {
					var err error
					leftCol.fromModel, err = s.layer.getModel(leftCol.tableStruct)
					if err != nil {
						panic(err)
					}
					leftCol.table = leftCol.fromModel.table
				}
			}

			if rightCol, ok := cond.right.(*Column); ok {
				if rightCol.tableStruct != nil {
					var err error
					rightCol.fromModel, err = s.layer.getModel(rightCol.tableStruct)
					if err != nil {
						panic(err)
					}
					rightCol.table = rightCol.fromModel.table
				}
			}
			cond.Build(s.builder, &s.args)
			if index != len(conditions)-1 {
				s.builder.WriteString(" AND ")
			}
		default:
			panic(ferr.ErrInvalidJoinCondition(cond))
		}
	}

	return s
}

func (s *Selector[T]) Using(cols ...string) *Selector[T] {
	s.builder.WriteString(" USING (")
	for i, col := range cols {
		s.builder.WriteString(col)
		if i != len(cols)-1 {
			s.builder.WriteString(", ")
		}
	}
	s.builder.WriteByte(')')
	return s
}

func (s *Selector[T]) AsSubQuery(alias string) *SubQuery[T] {
	return &SubQuery[T]{
		selector: s,
		alias:    alias,
	}
}

func (s *Selector[T]) Build() (*Query, error) {
	// 在build前先检查延迟处理的列
	for _, col := range s.delayCols {
		mp := *s.subqueryCache
		c, ok := mp[col.table]
		if !ok {
			return nil, ferr.ErrInvalidSubqueryColumn(col.table + "." + col.name)
		}

		_, ok = c[col.name]
		if !ok {
			return nil, ferr.ErrInvalidSubqueryColumn(col.table + "." + col.name)
		}
	}

	s.builder.WriteByte(';')
	return &Query{
		SQL:  s.builder.String(),
		Args: s.args,
	}, nil
}

// scanRow 将一行数据扫描到结构体中
// reflect version
//func (s *Selector[T]) scanRow(rows *sql.Rows) (*T, error) {
//	cols, err := rows.Columns()
//	if err != nil {
//		return nil, err
//	}
//
//	t := new(T)
//	vals := make([]any, len(cols))
//	// new返回的是指针
//	valElem := reflect.ValueOf(t).Elem()
//
//	for i, col := range cols {
//		if fieldName, ok := s.model.colNameMap[col]; ok {
//			field := valElem.FieldByName(fieldName)
//			vals[i] = field.Addr().Interface()
//		} else {
//			var v any
//			vals[i] = &v
//		}
//	}
//
//	if err = rows.Scan(vals...); err != nil {
//		return nil, err
//	}
//
//	return t, nil
//}

// scanRow 将一行数据扫描到结构体中
func (s *Selector[T]) scanRow(rows *sql.Rows) (*T, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	t := new(T)
	vals := make([]any, len(cols))

	// 获取结构体的值和类型
	value := reflect.ValueOf(t).Elem()
	typ := value.Type()
	// 获取结构体最初的地址
	baseAddr := unsafe.Pointer(reflect.ValueOf(t).Pointer())

	// 储存字段的地址与类型
	fieldAddrs := make(map[string]unsafe.Pointer)
	fieldTypes := make(map[string]reflect.Type)

	// 预先计算字段的地址
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldName := field.Name

		if field.PkgPath != "" {
			continue
		}

		// 计算地址
		fieldAddr := unsafe.Add(baseAddr, field.Offset)
		// 直接存储字段名
		//fieldAddrs[fieldName] = fieldAddr
		//fieldTypes[fieldName] = field.Type

		// 存储列名的相关信息
		if s.model != nil && s.model.fieldsMap != nil {
			if fieldMeta, ok := s.model.fieldsMap[fieldName]; ok {
				fieldAddrs[fieldMeta.colName] = fieldAddr
				fieldTypes[fieldMeta.colName] = field.Type
			}
		}
	}

	// 创建scan列表
	for i, col := range cols {
		if addr, ok := fieldAddrs[col]; ok {
			vals[i] = reflect.NewAt(fieldTypes[col], addr).Interface()
			continue
		}

		// 通过字段名找到对应的模型的列名
		//if s.model != nil && s.model.colNameMap != nil {
		//	if fieldName, ok := s.model.colNameMap[col]; ok {
		//		if addr, ok := fieldAddrs[fieldName]; ok {
		//			vals[i] = reflect.NewAt(fieldTypes[fieldName], addr).Interface()
		//			continue
		//		}
		//	}
		//}

		// 没找到匹配的列，返回一个dummy
		var dummy any
		vals[i] = &dummy
	}

	if err := rows.Scan(vals...); err != nil {
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

	// 构建查询上下文
	qc := &QueryContext{
		QueryType: "query",
		Query:     q,
		Model:     s.model,
		Builder:   s,
	}

	// 确保 layer 初始化了 handler
	if s.layer.getHandler() == nil {
		return nil, fmt.Errorf("handler not initialized")
	}

	res, err := s.layer.HandleQuery(ctx, qc)
	if err != nil {
		return nil, err
	}

	rows := res.Rows
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

	qc := &QueryContext{
		QueryType: "query",
		Query:     q,
		Model:     s.model,
		Builder:   s,
	}

	res, err := s.layer.HandleQuery(ctx, qc)
	if err != nil {
		return nil, err
	}

	rows := res.Rows
	defer rows.Close()

	result := make([]*T, 0)
	for rows.Next() {
		t, err := s.scanRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, t)
	}

	return result, rows.Err()
}
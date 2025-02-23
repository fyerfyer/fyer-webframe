package orm

import "strings"

type SubQuery[T any] struct {
	selector *Selector[T]
	alias    string
	model    *model
}

func (sq *SubQuery[T]) tableReference() string {
	return sq.alias
}


func (sq *SubQuery[T]) Build(builder *strings.Builder, args *[]any) any {
	builder.WriteString("(")
	query, err := sq.selector.Build()
	if err != nil {
		panic(err)
	}

	sqlString := query.SQL[:len(query.SQL)-1]
	builder.WriteString( sqlString + ")")
	*args = append(*args, query.Args...)

	if sq.alias != "" {
		builder.WriteString(" AS ")
		builder.WriteString(sq.alias)
	}

	mp := make(map[string]map[string]bool, 4)
	mp[sq.alias] = make(map[string]bool, 4)
	alias := mp[sq.alias]
	// 构建子查询缓存
	for _, col := range sq.selector.cols {
		if _, ok := alias[col]; !ok {
			alias[col] = true
		}
	}

	return &mp
}

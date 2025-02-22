package orm

import "strings"

// Expression 表达式接口
type Expression interface {
	expr()
}

// Selectable 选择接口
type Selectable interface {
	selectable()
}

// TableReference 表引用接口
type TableReference interface {
	tableReference() string
	Build (builder *strings.Builder, args *[]any) any
}

// As 别名接口
type As interface {
	As(alias string) Selectable
}

// Value 用于封装基础类型，使其满足Expression接口
type Value struct {
	val any
}

func (v *Value) expr() {}

func (v *Value) selectable() {}

func (v *Value) tableReference() string {
	return v.val.(string)
}

func (v *Value) Build(builder *strings.Builder, args *[]any) any {
	builder.WriteString(v.val.(string))
	return nil
}

// valueOf 将基础类型封装为Value
func valueOf(val any) *Value {
	return &Value{val: val}
}

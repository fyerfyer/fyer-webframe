package orm

// Expression 表达式接口
type Expression interface {
	expr()
}

// Selectable 选择接口
type Selectable interface {
	selectable()
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

// valueOf 将基础类型封装为Value
func valueOf(val any) *Value {
	return &Value{val: val}
}

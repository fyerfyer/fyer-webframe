package orm

type Expression interface {
	expr()
}

type selectable interface {
	selectable()
}

type As interface {
	As(alias string) selectable
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

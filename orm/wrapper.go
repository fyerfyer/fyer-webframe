package orm

type Expression interface {
	expr()
}

// Value 用于封装基础类型，使其满足Expression接口
type Value struct {
	val any
}

func (v *Value) expr() {}

// valueOf 将基础类型封装为Value
func valueOf(val any) *Value {
	return &Value{val: val}
}

package orm

// OrderBy 定义排序方向
type OrderBy struct {
	expr Expression
	desc bool
}

func Asc(expr Expression) OrderBy {
	return OrderBy{
		expr: expr,
		desc: false,
	}
}

func Desc(expr Expression) OrderBy {
	return OrderBy{
		expr: expr,
		desc: true,
	}
}

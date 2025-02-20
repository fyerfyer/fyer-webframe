package orm

import (
	"github.com/fyerfyer/fyer-webframe/orm/internal/utils"
	"strings"
)

type Aggregate struct {
	fn       string
	arg      string
	alias    string
	distinct bool
}

func (a Aggregate) selectable() {}

func (a Aggregate) As(alias string) Aggregate {
	return Aggregate{
		fn:       a.fn,
		arg:      a.arg,
		alias:    alias,
		distinct: a.distinct,
	}
}

func (a Aggregate) Build(builder *strings.Builder) {
	builder.WriteString(a.fn)
	builder.WriteString("(")
	if a.distinct {
		builder.WriteString("DISTINCT ")
	}
	if a.arg == "" {
		builder.WriteString("*")
	} else {
		builder.WriteString("`")
		// 添加列名转换
		builder.WriteString(utils.CamelToSnake(a.arg))
		builder.WriteString("`")
	}
	builder.WriteString(")")
	if a.alias != "" {
		builder.WriteString(" AS `")
		builder.WriteString(a.alias)
		builder.WriteString("`")
	}
}

// 聚合函数构造器
func Count(col string) Aggregate {
	return Aggregate{
		fn:  "COUNT",
		arg: col,
	}
}

func CountDistinct(col string) Aggregate {
	return Aggregate{
		fn:       "COUNT",
		arg:      col,
		distinct: true,
	}
}

func Sum(col string) Aggregate {
	return Aggregate{
		fn:  "SUM",
		arg: col,
	}
}

func Avg(col string) Aggregate {
	return Aggregate{
		fn:  "AVG",
		arg: col,
	}
}

func Max(col string) Aggregate {
	return Aggregate{
		fn:  "MAX",
		arg: col,
	}
}

func Min(col string) Aggregate {
	return Aggregate{
		fn:  "MIN",
		arg: col,
	}
}

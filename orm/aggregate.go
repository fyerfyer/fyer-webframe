package orm

import (
	"github.com/fyerfyer/fyer-webframe/orm/internal/ferr"
	"github.com/fyerfyer/fyer-webframe/orm/internal/utils"
	"strings"
)

type Aggregate struct {
	fn       string
	arg      string
	alias    string
	distinct bool
	model    *model
}

// Count 计算数量
func Count(col string) *Aggregate {
	return &Aggregate{
		fn:  "COUNT",
		arg: col,
	}
}

func CountDistinct(col string) *Aggregate {
	return &Aggregate{
		fn:       "COUNT",
		arg:      col,
		distinct: true,
	}
}

func Sum(col string) *Aggregate {
	return &Aggregate{
		fn:  "SUM",
		arg: col,
	}
}

func Avg(col string) *Aggregate {
	return &Aggregate{
		fn:  "AVG",
		arg: col,
	}
}

func Max(col string) *Aggregate {
	return &Aggregate{
		fn:  "MAX",
		arg: col,
	}
}

func Min(col string) *Aggregate {
	return &Aggregate{
		fn:  "MIN",
		arg: col,
	}
}

// 修改方法接收者为指针
func (a *Aggregate) expr() {}

func (a *Aggregate) selectable() {}

func (a *Aggregate) As(alias string) *Aggregate {
	return &Aggregate{
		fn:       a.fn,
		arg:      a.arg,
		alias:    alias,
		distinct: a.distinct,
		model:    a.model,
	}
}

// getDialect 获取当前模型对应的方言
func (a *Aggregate) getDialect() Dialect {
	if a.model != nil && a.model.dialect != nil {
		return a.model.dialect
	}
	// 默认使用MySQL方言
	return &Mysql{}
}

// Build 构建聚合函数
// 这里有一个细节：对于传入的字符串，如果在模型中找到了它对应的数据库字段的话，就会自动选择使用这个字段
// 否则就会将传入的字符串转换为蛇形命名法，然后使用这个字符串
// todo：固定下聚合函数内部的参数的类型
func (a *Aggregate) Build(builder *strings.Builder) {
	if a.model == nil {
		panic(ferr.ErrInvalidColumn(a.arg))
	}

	dialect := a.getDialect()

	builder.WriteString(a.fn)
	builder.WriteString("(")
	if a.distinct {
		builder.WriteString("DISTINCT ")
	}
	if a.arg == "" {
		builder.WriteString("*")
	} else if col, ok := a.model.fieldsMap[a.arg]; ok {
		builder.WriteString(dialect.Quote(col.colName))
	} else {
		builder.WriteString(dialect.Quote(utils.CamelToSnake(a.arg)))
	}

	builder.WriteString(")")
	if a.alias != "" {
		a.model.colAliasMap[a.alias] = true
		builder.WriteString(" AS ")
		builder.WriteString(dialect.Quote(a.alias))
	}
}

func (a *Aggregate) Eq(arg any) *Predicate {
	return &Predicate{
		left:  a,
		op:    opEQ,
		right: valueOf(arg),
	}
}

func (a *Aggregate) Gt(arg any) *Predicate {
	return &Predicate{
		left:  a,
		op:    opGT,
		right: valueOf(arg),
	}
}
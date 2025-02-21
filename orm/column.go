package orm

import (
	"strings"

	"github.com/fyerfyer/fyer-webframe/orm/internal/ferr"
)

type Column struct {
	name       string
	alias      string
	model      *model
	allowAlias bool
}

func Col(name string) *Column {
	return &Column{name: name}
}

func (c *Column) expr() {}

func (c *Column) selectable() {}

func (c *Column) As(alias string) *Column {
	return &Column{
		name:  c.name,
		alias: alias,
		model: c.model,
	}
}

func (c *Column) Build(builder *strings.Builder) {
	if c.model == nil {
		panic(ferr.ErrInvalidColumn(c.name))
	}

	// 先尝试从字段映射中查找
	if col, ok := c.model.fieldsMap[c.name]; ok {
		builder.WriteString("`" + col.colName + "`")
		if c.alias != "" {
			builder.WriteString(" AS `")
			builder.WriteString(c.alias)
			builder.WriteString("`")
			// 把别名加入到模型中（移到这里）
			c.model.aliasMap[c.alias] = true
		}
		return
	}

	// 如果允许使用别名，则尝试从别名映射中查找
	if c.allowAlias {
		if ok := c.model.aliasMap[c.name]; ok {
			builder.WriteString("`" + c.name + "`")
			return
		}
	}

	panic(ferr.ErrInvalidColumn(c.name))
}

func (c *Column) BuildWithoutQuote(builder *strings.Builder) {
	if c.model == nil {
		panic(ferr.ErrInvalidColumn(c.name))
	}

	col, ok := c.model.fieldsMap[c.name]
	if !ok {
		panic(ferr.ErrInvalidColumn(c.name))
	}

	builder.WriteString(col.colName)
}

func (c *Column) Eq(arg any) Predicate {
	return Predicate{
		left:  c,
		op:    opEQ,
		right: valueOf(arg),
	}
}

func (c *Column) Gt(arg any) Predicate {
	return Predicate{
		left:  c,
		op:    opGT,
		right: valueOf(arg),
	}
}

func (c *Column) IsNull() Predicate {
	return Predicate{
		left: c,
		op:   opISNULL,
	}
}

func (c *Column) NotNull() Predicate {
	return Predicate{
		left: c,
		op:   opNOTNULL,
	}
}

func NOT(pred Predicate) Predicate {
	return Predicate{
		op:    opNOT,
		right: pred,
	}
}

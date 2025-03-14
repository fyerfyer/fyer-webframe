package orm

import (
	"strings"

	"github.com/fyerfyer/fyer-webframe/orm/internal/ferr"
)

type Column struct {
	name        string
	alias       string
	table       string // 存储表名
	tableStruct any
	model       *model // 当前字段所属的model
	fromModel   *model // FromTable设置的model
	allowAlias  bool
	shouldDelay bool
}

func Col(name string) *Column {
	return &Column{name: name}
}

func (c *Column) expr() {}

func (c *Column) selectable() {}

func (c *Column) As(alias string) *Column {
	c.alias = alias
	return c
}

// FromTable 接收任意结构体，解析并缓存其模型信息
func FromTable(tableStruct any, col *Column) *Column {
	if tableName, ok := tableStruct.(string); ok {
		col.table = tableName
		return col
	}

	col.tableStruct = tableStruct
	return col
}

// getDialect 获取当前模型对应的方言
func (c *Column) getDialect() Dialect {
	if c.model != nil && c.model.dialect != nil {
		return c.model.dialect
	}
	// 默认使用MySQL方言
	return &Mysql{}
}

func (c *Column) Build(builder *strings.Builder) {
	dialect := c.getDialect()

	// 这里只有两种情况：join已有的表或者join一个子查询
	// 只有在这里，找不到对应的列的话才设置成延迟解析
	// 其他情况直接panic
	if c.table != "" {
		builder.WriteString(dialect.Quote(c.table) + ".")
		if c.fromModel == nil && c.model == nil {
			// 先不panic，而是把这个列标为延迟解析
			// 因为可能是在子查询中使用的列，而子查询的模型信息在后面才能获取到
			// 所以先把它写上，等之后发现这个列是不合法的的时候再panic
			builder.WriteString(dialect.Quote(c.name))
			return
		}
		if c.fromModel != nil {
			if alias, ok := c.fromModel.tableAliasMap[c.table]; ok {
				builder.WriteString(dialect.Quote(alias) + ".")
			}
		} else if alias, ok := c.model.tableAliasMap[c.table]; ok {
			builder.WriteString(dialect.Quote(alias) + ".")
		}
	}

	// 优先使用fromModel查找字段
	if c.fromModel != nil {
		if col, ok := c.fromModel.fieldsMap[c.name]; ok {
			builder.WriteString(dialect.Quote(col.colName))
			if c.alias != "" {
				builder.WriteString(" AS " + dialect.Quote(c.alias))
				c.fromModel.colAliasMap[c.alias] = true
			}
			return
		}

		if c.allowAlias {
			if c.alias != "" {
				builder.WriteString(dialect.Quote(c.alias))
				c.fromModel.colAliasMap[c.alias] = true
				return
			}
		}
	}

	// 回退到原有的查找逻辑
	if col, ok := c.model.fieldsMap[c.name]; ok {
		builder.WriteString(dialect.Quote(col.colName))
		if c.alias != "" {
			builder.WriteString(" AS " + dialect.Quote(c.alias))
			c.model.colAliasMap[c.alias] = true
		}
		return
	}

	if c.allowAlias {
		if c.alias != "" {
			builder.WriteString(dialect.Quote(c.alias))
			c.model.colAliasMap[c.alias] = true
			return
		} else if _, ok := c.model.colAliasMap[c.name]; ok {
			builder.WriteString(dialect.Quote(c.name))
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

func (c *Column) Eq(arg any) *Predicate {
	p := &Predicate{
		left: c,
		op:   opEQ,
	}

	switch arg := arg.(type) {
	case Expression:
		p.right = arg
	default:
		p.right = valueOf(arg)
	}

	return p
}

func (c *Column) Gt(arg any) *Predicate {
	return &Predicate{
		left:  c,
		op:    opGT,
		right: valueOf(arg),
	}
}

func (c *Column) IsNull() *Predicate {
	return &Predicate{
		left: c,
		op:   opISNULL,
	}
}

func (c *Column) NotNull() *Predicate {
	return &Predicate{
		left: c,
		op:   opNOTNULL,
	}
}

func (c *Column) Gte(arg any) *Predicate {
	return &Predicate{
		left:  c,
		op:    opGTE,
		right: valueOf(arg),
	}
}

func (c *Column) Lt(arg any) *Predicate {
	return &Predicate{
		left:  c,
		op:    opLT,
		right: valueOf(arg),
	}
}

func (c *Column) Lte(arg any) *Predicate {
	return &Predicate{
		left:  c,
		op:    opLTE,
		right: valueOf(arg),
	}
}

func (c *Column) Like(pattern string) *Predicate {
	return &Predicate{
		left:  c,
		op:    opLIKE,
		right: valueOf(pattern),
	}
}

func (c *Column) NotLike(pattern string) *Predicate {
	return &Predicate{
		left:  c,
		op:    opNOTLIKE,
		right: valueOf(pattern),
	}
}

func (c *Column) In(vals ...any) *Predicate {
	return &Predicate{
		left:  c,
		op:    opIN,
		right: valueOf(vals),
	}
}

func (c *Column) NotIn(vals ...any) *Predicate {
	return &Predicate{
		left:  c,
		op:    opNOTIN,
		right: valueOf(vals),
	}
}

func (c *Column) Between(start, end any) *Predicate {
	return &Predicate{
		left:  c,
		op:    opBETWEEN,
		right: valueOf([]any{start, end}),
	}
}

func (c *Column) NotBetween(start, end any) *Predicate {
	return &Predicate{
		left:  c,
		op:    opNOTBETWEEN,
		right: valueOf([]any{start, end}),
	}
}

func NOT(pred *Predicate) *Predicate {
	return &Predicate{
		op:    opNOT,
		right: pred,
	}
}
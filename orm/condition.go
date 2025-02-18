package orm

import "strings"

type Condition interface {
	Build(builder *strings.Builder, args *[]any)
}

// Predicate 需要同时实现 Expression 和 Condition 接口
type Predicate struct {
	left  Expression
	op    Op
	right Expression
}

func (p Predicate) expr() {}

func (p Predicate) Build(builder *strings.Builder, args *[]any) {
	switch p.op.Type {
	case OpUnary:
		// 一元运算符
		if col, ok := p.left.(Column); ok {
			builder.WriteString("`" + col.name + "` ")
		}
		builder.WriteString(p.op.Keyword)
		if pred, ok := p.right.(Condition); ok {
			builder.WriteString(" ")
			pred.Build(builder, args)
		}
	case OpBinary:
		// 二元运算符
		if c, ok := p.left.(Column); ok {
			builder.WriteString("`")
			builder.WriteString(c.name)
			builder.WriteString("`")
		}
		builder.WriteString(" ")
		builder.WriteString(p.op.Keyword)
		builder.WriteString(" ")
		if pred, ok := p.right.(Condition); ok {
			pred.Build(builder, args)
		} else {
			builder.WriteString("?")
			*args = append(*args, p.right)
		}
	}
}

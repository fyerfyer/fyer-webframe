package orm

import (
	"strings"
)

type Condition interface {
	Build(builder *strings.Builder, args *[]any)
}

// Predicate 需要同时实现 Expression 和 Condition 接口
type Predicate struct {
	left  Expression
	op    Op
	right Expression
	model *model
}

// buildExpr 构建表达式,处理不同类型的表达式构建
func (p *Predicate) buildExpr(expr Expression, builder *strings.Builder, args *[]any) {
	switch e := expr.(type) {
	case *Column:
		e.model = p.model
		e.Build(builder)
	case *Aggregate:
		e.model = p.model
		e.Build(builder)
	case *Value:
		builder.WriteByte('?')
		*args = append(*args, e.val)
	case *Predicate:
		e.model = p.model
		builder.WriteByte('(')
		e.Build(builder, args)
		builder.WriteByte(')')
	default:
		builder.WriteByte('?')
		*args = append(*args, expr)
	}
}

func (p *Predicate) expr() {}

func (p *Predicate) Build(builder *strings.Builder, args *[]any) {
	switch p.op.Type {
	case OpUnary:
		// 一元运算符: NOT, IS NULL 等
		if p.left == nil && p.right == nil {
			panic("left and right expressions cannot be nil for unary operator")
		}
		if p.left != nil {
			p.buildExpr(p.left, builder, args)
			builder.WriteByte(' ')
		}
		builder.WriteString(p.op.Keyword)
		if p.right != nil {
			builder.WriteByte(' ')
			p.buildExpr(p.right, builder, args)
		}

	case OpBinary:
		// 二元运算符: =, >, < 等
		if p.left == nil {
			panic("left expression cannot be nil for binary operator")
		}

		// 处理左表达式
		p.buildExpr(p.left, builder, args)

		// 添加操作符
		builder.WriteByte(' ')
		builder.WriteString(p.op.Keyword)
		builder.WriteByte(' ')

		// 处理右表达式
		p.buildExpr(p.right, builder, args)

	default:
		panic("invalid operator type")
	}
}

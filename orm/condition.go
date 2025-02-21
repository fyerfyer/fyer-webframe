package orm

import (
	"fmt"
	"github.com/fyerfyer/fyer-webframe/orm/internal/utils"
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
}

func (p Predicate) expr() {}

func (p Predicate) Build(builder *strings.Builder, args *[]any) {
	switch p.op.Type {
	case OpUnary:
		// 一元运算符
		if col, ok := p.left.(*Column); ok {
			builder.WriteString("`")
			// 优先使用模型中的列名
			if col.model != nil {
				if fd, ok := col.model.fieldsMap[col.name]; ok {
					builder.WriteString(fd.colName)
				} else {
					builder.WriteString(utils.CamelToSnake(col.name))
				}
			} else {
				builder.WriteString(utils.CamelToSnake(col.name))
			}
			builder.WriteString("`")
			builder.WriteString(" ")
		}

		builder.WriteString(p.op.Keyword)
		if pred, ok := p.right.(Condition); ok {
			builder.WriteString(" ")
			pred.Build(builder, args)
		}
	case OpBinary:
		// 处理左侧表达式
		switch left := p.left.(type) {
		case *Column:
			left.Build(builder)
		case *Aggregate: // 修改类型断言
			left.Build(builder)
		default:
			panic(fmt.Errorf("invalid left expression: %v", left))
		}

		builder.WriteString(" ")
		builder.WriteString(p.op.Keyword)
		builder.WriteString(" ")

		// 处理右侧表达式
		if pred, ok := p.right.(Condition); ok {
			pred.Build(builder, args)
		} else {
			builder.WriteString("?")
			if v, ok := p.right.(*Value); ok {
				*args = append(*args, v.val)
			} else {
				*args = append(*args, p.right)
			}
		}
	}
}

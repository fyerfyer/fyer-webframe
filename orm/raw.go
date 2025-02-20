package orm

import "strings"

type RawExpr struct {
	raw  string
	args []any
}

func (r RawExpr) expr() {}

func (r RawExpr) selectable() {}

func (r RawExpr) Build(builder *strings.Builder) {
	builder.WriteString(r.raw)
}

func Raw(raw string, args ...any) RawExpr {
	return RawExpr{
		raw:  raw,
		args: args,
	}
}

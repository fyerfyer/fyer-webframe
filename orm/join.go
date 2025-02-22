package orm

import "strings"

type JoinType string

const (
	InnerJoin JoinType = "INNER JOIN"
	LeftJoin  JoinType = "LEFT JOIN"
	RightJoin JoinType = "RIGHT JOIN"
	CrossJoin JoinType = "CROSS JOIN"
)

type Join struct {
	JoinType string
	Target TableReference
}

func (j *Join) tableReference() string {
	return j.Target.tableReference()
}

func (j *Join) Build(builder *strings.Builder, args *[]any) any {
	builder.WriteString(" " + j.JoinType)
	builder.WriteString(" ")
    return j.Target.Build(builder, args)
}

type Table_ struct {
	name string
}

func (t *Table_) tableReference() string {
	return t.name
}

func (t *Table_) Build(builder *strings.Builder, args *[]any) any {
	builder.WriteString("`" + t.name + "`")
	return nil
}

func Table(name string) *Table_ {
	return &Table_{name: name}
}
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
	Target   TableReference
	dialect  Dialect
}

func (j *Join) getDialect() Dialect {
	if j.dialect != nil {
		return j.dialect
	}

	return &Mysql{}
}

func (j *Join) SetDialect(dialect Dialect) {
	j.dialect = dialect
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
	name    string
	dialect Dialect
}

func (t *Table_) getDialect() Dialect {
	if t.dialect != nil {
		return t.dialect
	}

	return &Mysql{}
}

func (t *Table_) SetDialect(dialect Dialect) {
	t.dialect = dialect
}

func (t *Table_) tableReference() string {
	return t.name
}

func (t *Table_) Build(builder *strings.Builder, args *[]any) any {
	dialect := t.getDialect()
	builder.WriteString(dialect.Quote(t.name))
	return nil
}

func Table(name string) *Table_ {
	return &Table_{name: name}
}

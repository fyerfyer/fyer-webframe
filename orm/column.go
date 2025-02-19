package orm

type Column struct {
	name  string
	model *model
}

func Col(name string) *Column {
	return &Column{name: name}
}

func (c *Column) expr() {}

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

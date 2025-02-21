package orm

import (
	"errors"
	"github.com/fyerfyer/fyer-webframe/orm/internal/ferr"
	"strings"
)

type Sqlite struct{
	BaseDialect
}

func (s Sqlite) BuildUpsert(builder *strings.Builder, conflictCols []*Column, cols []*Column) {
	if conflictCols == nil || len(conflictCols) == 0 {
		panic(errors.New("sqlite must have conflict columns"))
	}
	if len(cols) == 0 {
		panic(ferr.ErrUpsertRowNotFound)
	}

	builder.WriteString(" ON CONFLICT(")
	for index, col := range cols {
		col.model = s.model
		col.Build(builder)
		if index != len(cols)-1 {
			builder.WriteString(", ")
		}
	}

	builder.WriteString(") DO UPDATE SET ")

	for index, col := range cols {
		col.BuildWithoutQuote(builder)
		builder.WriteString(" = EXCLUDED.")
		col.BuildWithoutQuote(builder)
		if index != len(cols)-1 {
			builder.WriteString(", ")
		}
	}
}

func init() {
	RegisterDialect("sqlite", &Sqlite{})
}
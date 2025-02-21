package orm

import (
	"errors"
	"github.com/fyerfyer/fyer-webframe/orm/internal/ferr"
	"strings"
)

type Mysql struct {
	BaseDialect
}

func (m Mysql) BuildUpsert(builder *strings.Builder, conflictCols []*Column, cols []*Column) {
	if conflictCols != nil {
		panic(errors.New("mysql does not support conflict columns"))
	}
	m.buildUpsert(builder, cols)
}

func (m Mysql) buildUpsert(builder *strings.Builder, cols []*Column) {
	if len(cols) == 0 {
		panic(ferr.ErrUpsertRowNotFound)
	}

	builder.WriteString(" ON DUPLICATE KEY UPDATE ")

	for index, col := range cols {
		if index > 0 {
			builder.WriteString(", ")
		}

		// 注入模型信息
		col.model = m.model
		col.Build(builder)
		builder.WriteString(" = VALUES(")
		col.BuildWithoutQuote(builder)
		builder.WriteByte(')')
	}
}

func init() {
	RegisterDialect("mysql", &Mysql{})
}

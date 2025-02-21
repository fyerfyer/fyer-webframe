package orm

import (
	"strings"
)

type Dialect interface {
	BuildUpsert(builder *strings.Builder, conflictCols []*Column, cols []*Column)
}

var (
	dialects = make(map[string]Dialect)
)

func RegisterDialect(name string, dialect Dialect) {
	dialects[name] = dialect
}

func Get(name string) Dialect {
	return dialects[name]
}

type BaseDialect struct {
	model *model
}

func (b *BaseDialect) setModel(m *model) {
	b.model = m
}

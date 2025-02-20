package orm

import "strings"

type Inserter[T any] struct {
	builder *strings.Builder
	values  []any
	db      *DB
}

func RegisterInserter[T any](db *DB) *Inserter[T] {
	return &Inserter[T]{
		db: db,
	}
}

//func (s *Inserter[T]) Insert(table string,) (int64, error) {
//
//}

package sql

import "database/sql"

type RowScanner[T any] interface {
	ScanRow(rows *sql.Rows) (*T, error)
}



package ferr

import "fmt"

var (
	ErrNoRows            = fmt.Errorf("data not found")
	ErrTooManyRows       = fmt.Errorf("too many rows")
	ErrInsertRowNotFound = fmt.Errorf("insert row not found")
	ErrUpsertRowNotFound = fmt.Errorf("upsert row not found")
)

func ErrInvalidColumn(col string) error {
	return fmt.Errorf("invalid column name: %s", col)
}

func ErrInvalidTag(tag string) error {
	return fmt.Errorf("invalid tag: %s", tag)
}

func ErrInvalidSelectable(col any) error {
	return fmt.Errorf("invalid selectable column: %v", col)
}

func ErrInvalidSubqueryColumn(col any) error {
	return fmt.Errorf("invalid subquery column: %v", col)
}

func ErrInvalidJoinCondition(cond any) error {
	return fmt.Errorf("invalid join condition: %v", cond)
}

func ErrInvalidTableReference(table any) error {
	return fmt.Errorf("invalid table reference: %v", table)
}

func ErrInvalidInsertValue(v any) error {
	return fmt.Errorf("invalid insert value: %v", v)
}

func ErrInvalidDialect(v any) error {
	return fmt.Errorf("invalid dialect: %v", v)
}

func ErrInvalidOrderBy(v any) error {
	return fmt.Errorf("invalid order by column: %v", v)
}
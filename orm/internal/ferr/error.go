package ferr

import "fmt"

var (
	ErrNoRows      = fmt.Errorf("data not found")
	ErrTooManyRows = fmt.Errorf("too many rows")
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
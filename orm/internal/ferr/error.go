package ferr

import "fmt"

func ErrInvalidColumn(col string) error {
	return fmt.Errorf("invalid column name: %s", col)
}

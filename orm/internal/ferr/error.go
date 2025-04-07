package ferr

import (
	"errors"
	"fmt"
)

var (
	ErrNoRows            = fmt.Errorf("data not found")
	ErrTooManyRows       = fmt.Errorf("too many rows")
	ErrInsertRowNotFound = fmt.Errorf("insert row not found")
	ErrUpsertRowNotFound = fmt.Errorf("upsert row not found")
	ErrPointerOnly       = errors.New("orm: only supports pointers to structs, e.g., *User")
)

var (
	ErrPoolClosed              = errors.New("orm: connection pool is closed")
	ErrPoolFull                = errors.New("orm: connection pool is full")
	ErrPoolExhausted           = errors.New("orm: connection pool resources exhausted")
	ErrConnTimeout             = errors.New("orm: connection timeout")
	ErrInvalidConnection       = errors.New("orm: invalid database connection")
	ErrTransactionOnBrokenConn = errors.New("orm: cannot create transaction on a broken connection")
	ErrTooManyClients          = errors.New("orm: too many clients waiting for connection")
	ErrDBClosed                = errors.New("orm: operation on a closed database")
)

func ErrInvalidColumn(col string) error {
	return fmt.Errorf("invalid column name: %s", col)
}

func ErrInvalidTag(tag string) error {
	return fmt.Errorf("orm: invalid tag %s", tag)
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

func ErrDialTimeout(duration string) error {
	return fmt.Errorf("orm: database connection timeout, duration %s", duration)
}

func ErrHealthCheckFailed(reason string) error {
	return fmt.Errorf("orm: connection health check failed: %s", reason)
}

func ErrCreateConnectionFailed(err error) error {
	return fmt.Errorf("orm: failed to create database connection: %w", err)
}
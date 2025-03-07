package orm

import (
	"context"
	"database/sql"
	"errors"
)

type Tx struct {
	db *DB
	tx *sql.Tx
}

func (t *Tx) getModel(val any) (*model, error) {
	m, err := t.db.getModel(val)
	if err != nil {
		return nil, err
	}
	// 确保从事务中获取的模型也设置了方言
	m.SetDialect(t.db.dialect)
	return m, nil
}

func (t *Tx) getDB() *DB {
	return t.db
}

func (t *Tx) queryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return t.tx.QueryContext(ctx, query, args...)
}

func (t *Tx) execContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

func (t *Tx) getHandler() Handler {
	return t.db.handler
}

func (t *Tx) HandleQuery(ctx context.Context, qc *QueryContext) (*QueryResult, error) {
	return t.db.handler.QueryHandler(ctx, qc)
}

func (t *Tx) Commit() error {
	return t.tx.Commit()
}

func (t *Tx) RollBack() error {
	return t.tx.Rollback()
}

func (t *Tx) RollbackIfNotCommitted() error {
	if t.tx != nil {
		err := t.tx.Rollback()
		if err == nil || errors.Is(err, sql.ErrTxDone) {
			return nil
		}

		return err
	}

	return errors.New("transaction is not started")
}
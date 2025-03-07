package orm

import (
	"context"
	"database/sql"
	"errors"
	"github.com/fyerfyer/fyer-webframe/orm/internal/ferr"

	"github.com/fyerfyer/fyer-kit/pool"
)

type Tx struct {
	db       *DB
	tx       *sql.Tx
	poolConn pool.Connection // 来自连接池的连接
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
	err := t.tx.Commit()

	// 如果是连接池模式，归还连接
	if t.poolConn != nil {
		t.db.pooledDB.PutConn(t.poolConn, err)
		t.poolConn = nil
	}

	return err
}

func (t *Tx) RollBack() error {
	err := t.tx.Rollback()

	// 如果是连接池模式，归还连接
	if t.poolConn != nil {
		t.db.pooledDB.PutConn(t.poolConn, err)
		t.poolConn = nil
	}

	return err
}

func (t *Tx) RollbackIfNotCommitted() error {
	if t.tx != nil {
		err := t.tx.Rollback()

		// 如果是连接池模式，归还连接
		if t.poolConn != nil {
			t.db.pooledDB.PutConn(t.poolConn, err)
			t.poolConn = nil
		}

		if err == nil || errors.Is(err, sql.ErrTxDone) {
			return nil
		}

		return err
	}

	return errors.New("transaction is not started")
}

// 实现 Layer 接口的方法
func (t *Tx) getConn(ctx context.Context) (*sql.DB, pool.Connection, error) {
	// 事务已经绑定了一个连接，不应该再获取新连接
	return nil, nil, ferr.ErrTransactionOnBrokenConn
}

func (t *Tx) putConn(conn pool.Connection, err error) {
	// 事务中不应该直接归还连接，只有在提交或回滚时才释放
}
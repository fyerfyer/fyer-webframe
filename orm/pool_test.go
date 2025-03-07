package orm

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectionPool_Basic(t *testing.T) {
	// 创建模拟数据库连接
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置预期查询
	mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

	// 创建ORM实例并启用连接池
	db, err := Open(mockDB, "mysql", WithPoolSize(5, 10))
	require.NoError(t, err)
	defer db.Close()

	// 执行简单查询
	rows, err := db.queryContext(context.Background(), "SELECT 1")
	require.NoError(t, err)
	defer rows.Close()

	// 验证查询执行正确
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestConnectionPool_Transaction(t *testing.T) {
	// 创建模拟数据库连接
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置预期事务操作
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO test_model").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// 创建ORM实例并启用连接池
	db, err := Open(mockDB, "mysql", WithPoolSize(5, 10))
	require.NoError(t, err)
	defer db.Close()

	// 执行事务
	err = db.Tx(context.Background(), func(tx *Tx) error {
		_, err := tx.execContext(context.Background(), "INSERT INTO test_model VALUES(1, 'test')")
		return err
	}, nil)
	require.NoError(t, err)

	// 验证事务执行正确
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestConnectionPool_MultipleQueries(t *testing.T) {
	// 创建模拟数据库连接
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置预期查询
	for i := 0; i < 5; i++ {
		mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(i))
	}

	// 创建ORM实例并启用连接池
	db, err := Open(mockDB, "mysql", WithConnectionPool(
		WithPoolMaxIdle(5),
		WithPoolMaxActive(10),
	))
	require.NoError(t, err)
	defer db.Close()

	// 执行多个查询
	for i := 0; i < 5; i++ {
		rows, err := db.queryContext(context.Background(), "SELECT")
		require.NoError(t, err)
		rows.Close()
	}

	// 验证所有查询执行正确
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestConnectionPool_QueryWithinTransaction(t *testing.T) {
	// 创建模拟数据库连接
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置预期事务和查询
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	// 创建ORM实例并启用连接池
	db, err := Open(mockDB, "mysql", WithPoolSize(5, 10))
	require.NoError(t, err)
	defer db.Close()

	// 执行事务内的查询和更新
	err = db.Tx(context.Background(), func(tx *Tx) error {
		rows, err := tx.queryContext(context.Background(), "SELECT")
		if err != nil {
			return err
		}
		defer rows.Close()

		_, err = tx.execContext(context.Background(), "UPDATE")
		return err
	}, nil)
	require.NoError(t, err)

	// 验证事务内的操作执行正确
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestConnectionPool_RollbackTransaction(t *testing.T) {
	// 创建模拟数据库连接
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置预期事务和回滚
	mock.ExpectBegin()
	mock.ExpectExec("INSERT").WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	// 创建ORM实例并启用连接池
	db, err := Open(mockDB, "mysql", WithPoolSize(5, 10))
	require.NoError(t, err)
	defer db.Close()

	// 执行会失败的事务
	err = db.Tx(context.Background(), func(tx *Tx) error {
		_, err := tx.execContext(context.Background(), "INSERT")
		return err
	}, nil)
	require.Error(t, err)

	// 验证事务回滚正确执行
	assert.NoError(t, mock.ExpectationsWereMet())
}
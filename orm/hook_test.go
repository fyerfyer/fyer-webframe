package orm

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectionPool_WithHooks(t *testing.T) {
	// 创建模拟数据库连接
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置测试预期
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(1, 1))

	// 跟踪钩子调用
	var getHookCalled bool
	var putHookCalled bool
	var healthCheckCalled bool
	var closeHookCalled bool

	// 创建测试钩子
	hooks := &ConnHooks{
		OnGet: func(ctx context.Context, conn *sql.DB) error {
			getHookCalled = true
			return nil
		},
		OnPut: func(conn *sql.DB, err error) error {
			putHookCalled = true
			return nil
		},
		OnCheckHealth: func(conn *sql.DB) bool {
			healthCheckCalled = true
			return true
		},
		OnClose: func(conn *sql.DB) error {
			closeHookCalled = true
			return nil
		},
	}

	// 创建ORM实例并配置连接池和钩子
	db, err := Open(mockDB, "mysql",
		WithConnectionPool(
			WithPoolMaxIdle(5),
			WithPoolMaxActive(10),
		),
		WithConnHooks(hooks),
	)
	require.NoError(t, err)

	// 执行一个查询，这应该调用Get和Put钩子
	rows, err := db.queryContext(context.Background(), "SELECT")
	require.NoError(t, err)
	defer rows.Close()

	// 执行一个更新操作，这也应该调用Get和Put钩子
	_, err = db.execContext(context.Background(), "UPDATE")
	require.NoError(t, err)

	// 关闭数据库，这应该调用Close钩子
	err = db.Close()
	require.NoError(t, err)

	// 验证钩子是否被调用
	assert.True(t, getHookCalled, "OnGet hook should be called")
	assert.True(t, putHookCalled, "OnPut hook should be called")
	assert.True(t, healthCheckCalled, "OnCheckHealth hook should be called")
	assert.True(t, closeHookCalled, "OnClose hook should be called")

	// 验证所有预期的SQL语句都已执行
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestConnectionPool_HooksWithError(t *testing.T) {
	// 创建模拟数据库连接
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置测试预期
	expectedError := errors.New("get hook error")

	// 注意：我们不再期望SQL查询，因为钩子会阻止它执行

	// 创建会返回错误的钩子
	hooks := &ConnHooks{
		OnGet: func(ctx context.Context, conn *sql.DB) error {
			return expectedError
		},
	}

	// 创建ORM实例并配置连接池和钩子
	db, err := Open(mockDB, "mysql",
		WithConnectionPool(
			WithPoolMaxIdle(5),
			WithPoolMaxActive(10),
		),
		WithConnHooks(hooks),
	)
	require.NoError(t, err)
	defer db.Close()

	// 执行查询，应该返回OnGet钩子的错误
	_, err = db.queryContext(context.Background(), "SELECT")
	assert.Equal(t, expectedError, err, "Should return error from OnGet hook")

	// 验证所有预期的SQL语句都已执行
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestConnectionPool_HooksInTransaction(t *testing.T) {
	// 创建模拟数据库连接
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置测试预期
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	// 跟踪钩子调用
	var getHookCalled bool
	var putHookCalled bool

	// 创建测试钩子
	hooks := &ConnHooks{
		OnGet: func(ctx context.Context, conn *sql.DB) error {
			getHookCalled = true
			return nil
		},
		OnPut: func(conn *sql.DB, err error) error {
			putHookCalled = true
			return nil
		},
	}

	// 创建ORM实例并配置连接池和钩子
	db, err := Open(mockDB, "mysql",
		WithConnectionPool(
			WithPoolMaxIdle(5),
			WithPoolMaxActive(10),
		),
		WithConnHooks(hooks),
	)
	require.NoError(t, err)
	defer db.Close()

	// 执行事务
	err = db.Tx(context.Background(), func(tx *Tx) error {
		// 执行查询
		rows, err := tx.queryContext(context.Background(), "SELECT")
		if err != nil {
			return err
		}
		defer rows.Close()
		return nil
	}, nil)
	require.NoError(t, err)

	// 验证钩子是否被调用
	assert.True(t, getHookCalled, "OnGet hook should be called")
	assert.True(t, putHookCalled, "OnPut hook should be called")

	// 验证所有预期的SQL语句都已执行
	assert.NoError(t, mock.ExpectationsWereMet())
}

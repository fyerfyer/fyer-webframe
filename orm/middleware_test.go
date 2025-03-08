package orm

import (
	"context"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// TestMiddleware 测试中间件功能
func TestMiddleware(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)

	// 记录执行顺序的切片
	var order []string

	// 模拟日志中间件
	logMiddleware := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, qc *QueryContext) (*QueryResult, error) {
			order = append(order, "log start")
			res, err := next.QueryHandler(ctx, qc)
			order = append(order, "log end")
			return res, err
		})
	}

	// 模拟耗时统计中间件
	metricMiddleware := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, qc *QueryContext) (*QueryResult, error) {
			start := time.Now()
			order = append(order, "metric start")
			res, err := next.QueryHandler(ctx, qc)
			order = append(order, "metric end")
			// 记录执行时间
			_ = time.Since(start)
			return res, err
		})
	}

	// 注册中间件
	db.Use(logMiddleware, metricMiddleware)

	// 准备测试数据
	mock.ExpectQuery("SELECT *").
		WithArgs(12).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow(12, "Tom"))

	// 执行查询
	selector := RegisterSelector[TestModel](db).
		Select().
		Where(Col("ID").Eq(12))

	_, err = selector.Get(context.Background())
	require.NoError(t, err)

	// 验证中间件执行顺序
	// 期望顺序: log start -> metric start -> metric end -> log end
	expected := []string{
		"log start",
		"metric start",
		"metric end",
		"log end",
	}
	assert.Equal(t, expected, order)
}

// TestMiddlewareChain 测试中间件链的构建
func TestMiddlewareChain(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)

	// 测试用例
	testCases := []struct {
		name            string
		middlewares    []Middleware
		wantInitErr    error
		wantHandlerNil bool
	}{
		{
			name: "empty middleware",
			middlewares: []Middleware{},
			wantHandlerNil: false,
		},
		{
			name: "single middleware",
			middlewares: []Middleware{
				func(next Handler) Handler {
					return HandlerFunc(func(ctx context.Context, qc *QueryContext) (*QueryResult, error) {
						return next.QueryHandler(ctx, qc)
					})
				},
			},
			wantHandlerNil: false,
		},
		{
			name: "multiple middlewares",
			middlewares: []Middleware{
				func(next Handler) Handler {
					return HandlerFunc(func(ctx context.Context, qc *QueryContext) (*QueryResult, error) {
						return next.QueryHandler(ctx, qc)
					})
				},
				func(next Handler) Handler {
					return HandlerFunc(func(ctx context.Context, qc *QueryContext) (*QueryResult, error) {
						return next.QueryHandler(ctx, qc)
					})
				},
			},
			wantHandlerNil: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db.Use(tc.middlewares...)
			assert.Equal(t, tc.wantHandlerNil, db.handler == nil)
		})
	}
}

// TestMiddlewareContext 测试中间件上下文传递
func TestMiddlewareContext(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)

	// 上下文键
	type ctxKey struct{}

	// 创建一个在上下文中传递值的中间件
	contextMiddleware := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, qc *QueryContext) (*QueryResult, error) {
			// 在上下文中设置值
			ctx = context.WithValue(ctx, ctxKey{}, "test-value")
			return next.QueryHandler(ctx, qc)
		})
	}

	// 创建一个检查上下文值的中间件
	checkMiddleware := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, qc *QueryContext) (*QueryResult, error) {
			// 获取上下文中的值
			val := ctx.Value(ctxKey{}).(string)
			assert.Equal(t, "test-value", val)
			return next.QueryHandler(ctx, qc)
		})
	}

	// 注册中间件
	db.Use(contextMiddleware, checkMiddleware)

	// 准备测试数据
	mock.ExpectQuery("SELECT *").
		WithArgs(12).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(12))

	// 执行查询
	selector := RegisterSelector[TestModel](db).
		Select().
		Where(Col("ID").Eq(12))

	_, err = selector.Get(context.Background())
	require.NoError(t, err)
}
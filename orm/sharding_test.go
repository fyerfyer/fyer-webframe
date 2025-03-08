package orm

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShardingStrategy 测试分片策略
func TestShardingStrategy(t *testing.T) {
	// 测试哈希策略
	hashStrategy := WithHashStrategy("user_db_", 4, "user_", 8, "UserID")

	// 测试路由
	dbIndex, tableIndex, err := hashStrategy.Route(1001)
	require.NoError(t, err)
	dbName, tableName, err := hashStrategy.GetShardName(dbIndex, tableIndex)
	require.NoError(t, err)

	// 哈希结果可能因哈希算法而异，但应该是确定的
	t.Logf("Hash Strategy Result - dbIndex: %d, tableIndex: %d, dbName: %s, tableName: %s",
		dbIndex, tableIndex, dbName, tableName)

	// 测试取模策略
	modStrategy := WithModStrategy("order_db_", 3, "order_", 5, "OrderID")

	// 测试路由
	dbIndex, tableIndex, err = modStrategy.Route(1001)
	require.NoError(t, err)

	// 取模的结果是确定的: 1001 % 3 = 2, (1001 / 3) % 5 = 3
	assert.Equal(t, 2, dbIndex)
	assert.Equal(t, 3, tableIndex)

	dbName, tableName, err = modStrategy.GetShardName(dbIndex, tableIndex)
	require.NoError(t, err)
	assert.Equal(t, "order_db_2", dbName)
	assert.Equal(t, "order_3", tableName)
}

// ShardingUser 分片用户模型
type ShardingUser struct {
	UserID   int64     `orm:"primary_key"`
	Username string    `orm:"size:255;unique"`
	Email    string    `orm:"size:255"`
	RegTime  time.Time
	Status   int
}

// ShardingOrder 分片订单模型
type ShardingOrder struct {
	OrderID  int64     `orm:"primary_key"`
	UserID   int64
	Amount   float64
	Status   int
	CreateAt time.Time
}

// TestShardingDB_Basic 测试基本的分片数据库功能
func TestShardingDB_Basic(t *testing.T) {
	// 创建模拟数据库
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置模拟预期 - 检查表是否存在
	mock.ExpectQuery("SELECT 1 FROM information_schema.tables WHERE table_name = 'sharding_user'").
		WillReturnRows(sqlmock.NewRows([]string{"1"}))

	// 创建ORM实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)
	defer db.Close()

	// 创建路由器
	router := NewShardingRouter()

	// 创建分片数据库
	shardDB := NewShardingDB(db, router)

	// 注册分片策略
	shardDB.RegisterShardStrategy("ShardingUser",
		WithHashStrategy("user_db_", 4, "user_", 8, "UserID"), "user_db_0")

	// 注册分片配置
	err = shardDB.ConfigureShards(map[string]ShardConfig{
		"user_db_0": {Driver: "mysql", DSN: "user:pass@tcp(localhost:3306)/user_db_0"},
		"user_db_1": {Driver: "mysql", DSN: "user:pass@tcp(localhost:3306)/user_db_1"},
	})
	// 这里预期会失败，因为我们使用的是mock数据库
	assert.Error(t, err)
}

// TestShardingRouter 测试分片路由
func TestShardingRouter(t *testing.T) {
	// 创建路由器
	router := NewShardingRouter()

	// 注册分片策略
	hashStrategy := WithHashStrategy("user_db_", 4, "user_", 8, "UserID")
	router.RegisterStrategy("ShardingUser", hashStrategy)

	modStrategy := WithModStrategy("order_db_", 3, "order_", 5, "OrderID")
	router.RegisterStrategy("ShardingOrder", modStrategy)

	// 测试用户路由
	userValues := map[string]interface{}{
		"UserID": 1001,
	}
	dbName, tableName, err := router.CalculateRoute(context.Background(), &ShardingUser{}, userValues)
	require.NoError(t, err)
	t.Logf("UserID 1001 routes to: %s.%s", dbName, tableName)

	// 测试订单路由
	orderValues := map[string]interface{}{
		"OrderID": 1001,
	}
	dbName, tableName, err = router.CalculateRoute(context.Background(), &ShardingOrder{}, orderValues)
	require.NoError(t, err)
	assert.Equal(t, "order_db_2", dbName)  // 1001 % 3 = 2
	assert.Equal(t, "order_3", tableName)  // (1001 / 3) % 5 = 3
}

// TestShardingMiddleware 测试分片中间件
func TestShardingMiddleware(t *testing.T) {
	// 创建模拟数据库
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 创建ORM实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)
	defer db.Close()

	// 创建路由器
	router := NewShardingRouter()

	// 注册分片策略
	router.RegisterStrategy("ShardingUser", WithHashStrategy("user_db_", 4, "user_", 8, "UserID"))

	// 创建分片管理器
	manager := NewShardingManager(db, router)

	// 注册中间件
	db.Use(ShardingMiddleware(manager))

	// 设置模拟预期 - 查询
	mock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "username"}).AddRow(1001, "test_user"))

	// 创建查询上下文
	qc := &QueryContext{
		QueryType:  "query",
		Query:      &Query{SQL: "SELECT user_id, username FROM sharding_user WHERE user_id = ?", Args: []interface{}{1001}},
		TableName:  "sharding_user",
		ShardKey:   "UserID",
		ShardValue: 1001,
	}

	// 执行查询
	result, err := db.HandleQuery(context.Background(), qc)
	require.NoError(t, err)
	assert.NotNil(t, result.Rows)
}

// TestShardingClient 测试分片客户端
func TestShardingClient(t *testing.T) {
	// 创建模拟数据库
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 创建ORM实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)
	defer db.Close()

	// 创建路由器
	router := NewShardingRouter()

	// 注册分片策略
	router.RegisterStrategy("ShardingUser", WithHashStrategy("user_db_", 4, "user_", 8, "UserID"))

	// 创建分片数据库
	shardDB := NewShardingDB(db, router)

	// 创建分片客户端
	client := shardDB.AsShardingClient()

	// 测试分片键查询上下文
	shardCtx := client.WithShardKey("ShardingUser", "UserID", 1001)
	assert.NotNil(t, shardCtx)

	// 设置模拟预期 - 原始SQL执行
	mock.ExpectExec("INSERT INTO").
		WithArgs(1001, "test_user", "test@example.com").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 测试SQL执行
	_, err = shardCtx.Exec(context.Background(), "INSERT INTO sharding_user (user_id, username, email) VALUES (?, ?, ?)", 1001, "test_user", "test@example.com")
	require.NoError(t, err)
}
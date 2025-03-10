package orm

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/fyerfyer/fyer-webframe/orm/internal/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCacheInterface 测试缓存接口的内存实现
func TestCacheInterface(t *testing.T) {
	memCache := NewMemoryCache()
	ctx := context.Background()

	// 测试缓存不存在
	var result string
	err := memCache.Get(ctx, "test_key", &result)
	assert.Equal(t, ErrCacheMiss, err)

	// 测试缓存设置
	err = memCache.Set(ctx, "test_key", "test_value", 5*time.Second)
	assert.NoError(t, err)

	// 测试缓存命中
	err = memCache.Get(ctx, "test_key", &result)
	assert.NoError(t, err)
	assert.Equal(t, "test_value", result)

	// 测试标签系统
	err = memCache.SetWithTags(ctx, "tagged_key", "tagged_value", 5*time.Second, "tag1", "tag2")
	assert.NoError(t, err)

	var taggedResult string
	err = memCache.Get(ctx, "tagged_key", &taggedResult)
	assert.NoError(t, err)
	assert.Equal(t, "tagged_value", taggedResult)

	// 按标签删除
	err = memCache.DeleteByTags(ctx, "tag1")
	assert.NoError(t, err)

	// 验证已删除
	err = memCache.Get(ctx, "tagged_key", &taggedResult)
	assert.Equal(t, ErrCacheMiss, err)

	// 清空缓存
	err = memCache.Clear(ctx)
	assert.NoError(t, err)

	// 验证已清空
	err = memCache.Get(ctx, "test_key", &result)
	assert.Equal(t, ErrCacheMiss, err)
}

// TestCacheKeyGeneration 测试缓存键生成
func TestCacheKeyGeneration(t *testing.T) {
	keyGen := cache.NewDefaultKeyGenerator("test_prefix")

	// 测试基本键生成
	key1 := keyGen.Generate("User", "query", "SELECT * FROM users WHERE id = ?", []interface{}{1})
	key2 := keyGen.Generate("User", "query", "SELECT * FROM users WHERE id = ?", []interface{}{2})
	assert.NotEqual(t, key1, key2, "不同参数应生成不同的键")

	// 测试相同的参数应该生成相同的键
	key1Again := keyGen.Generate("User", "query", "SELECT * FROM users WHERE id = ?", []interface{}{1})
	assert.Equal(t, key1, key1Again, "相同参数应生成相同的键")

	// 测试带标签的键生成
	taggedKey := keyGen.GenerateWithTags("User", "query", "SELECT * FROM users", nil, "list", "all")
	assert.Contains(t, taggedKey, "tags:all,list", "标签应该包含在键中")

	// 测试标签键生成
	tagKey := keyGen.BuildTagKey("list")
	assert.Equal(t, "tag:list", tagKey)
}

// TestCacheManager 测试缓存管理器
func TestCacheManager(t *testing.T) {
	memCache := NewMemoryCache()
	cm := NewCacheManager(memCache)

	// 测试默认设置
	assert.Equal(t, 5*time.Minute, cm.defaultTTL)
	assert.True(t, cm.enabled)

	// 测试模型配置
	cm.SetModelCacheConfig("User", &ModelCacheConfig{
		Enabled: true,
		TTL:     10 * time.Minute,
		Tags:    []string{"user", "auth"},
	})

	config, ok := cm.GetModelCacheConfig("User")
	assert.True(t, ok)
	assert.Equal(t, 10*time.Minute, config.TTL)
	assert.Equal(t, []string{"user", "auth"}, config.Tags)

	// 测试是否应该缓存
	ctx := context.Background()
	qc := &QueryContext{
		QueryType: "query",
		Model:     &model{table: "User"},
	}

	assert.True(t, cm.ShouldCache(ctx, qc))

	// 禁用缓存
	cm.Disable()
	assert.False(t, cm.ShouldCache(ctx, qc))

	// 重新启用
	cm.Enable()
	assert.True(t, cm.ShouldCache(ctx, qc))

	// 测试自定义条件
	cm.SetModelCacheConfig("User", &ModelCacheConfig{
		Enabled: true,
		Conditions: []CacheCondition{
			func(ctx context.Context, qc *QueryContext) bool {
				// 只有 query 类型的查询才缓存
				return qc.QueryType == "query"
			},
		},
	})

	assert.True(t, cm.ShouldCache(ctx, qc))

	qc.QueryType = "exec"
	assert.False(t, cm.ShouldCache(ctx, qc))
}

// TestCacheWithSelector 测试缓存与选择器的集成
func TestCacheWithSelector(t *testing.T) {
	// 创建模拟数据库
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// 设置预期查询 - 只期望查询一次，因为第二次会命中缓存
	mock.ExpectQuery("SELECT .*").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "job"}).
			AddRow(1, "Test User", sql.NullString{String: "Developer", Valid: true}))

	// 创建ORM实例并配置缓存
	ormDB, err := Open(db, "mysql")
	require.NoError(t, err)
	defer ormDB.Close()

	memCache := NewMemoryCache()
	ormDB.SetCacheManager(NewCacheManager(memCache))

	// 配置User模型的缓存
	ormDB.SetModelCacheConfig("test_model", &ModelCacheConfig{
		Enabled: true,
		TTL:     1 * time.Minute,
		Tags:    []string{"test"},
	})

	ctx := context.Background()

	// 执行第一次查询，这应该会缓存结果
	selector := RegisterSelector[TestModel](ormDB).
		Select().
		Where(Col("ID").Eq(1)).
		WithCache()

	t.Log(selector.Build())
	t.Logf("Selector enable cache: %v", selector.useCache)

	result, err := selector.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ID)
	assert.Equal(t, "Test User", result.Name)

	// 验证SQL查询已执行
	err = mock.ExpectationsWereMet()
	require.NoError(t, err)

	// 不需要清空模拟预期，因为我们不期望有第二次实际查询

	// 再次执行相同的查询，应从缓存获取结果
	result2, err := selector.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, result.ID, result2.ID)
	assert.Equal(t, result.Name, result2.Name)

	// 验证没有额外的SQL查询执行 (所有预期都已满足)
	err = mock.ExpectationsWereMet()
	require.NoError(t, err)
}

// TestCacheInvalidation 测试缓存失效机制
func TestCacheInvalidation(t *testing.T) {
	// 创建模拟数据库
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// 设置预期查询
	mock.ExpectQuery("SELECT \\*").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "job"}).
			AddRow(1, "Test User", sql.NullString{String: "Developer", Valid: true}))

	// 设置预期更新
	mock.ExpectExec("UPDATE test_model").
		WithArgs("New Name", 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 设置第二次查询的预期
	mock.ExpectQuery("SELECT \\*").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "job"}).
			AddRow(1, "New Name", sql.NullString{String: "Developer", Valid: true}))

	// 创建ORM实例并配置缓存
	ormDB, err := Open(db, "mysql")
	require.NoError(t, err)
	defer ormDB.Close()

	memCache := NewMemoryCache()
	cm := NewCacheManager(memCache)
	ormDB.SetCacheManager(cm)

	// 配置User模型的缓存
	ormDB.SetModelCacheConfig("test_model", &ModelCacheConfig{
		Enabled: true,
		TTL:     1 * time.Minute,
		Tags:    []string{"test"},
	})

	ctx := context.Background()

	// 执行第一次查询，这将缓存结果
	selector := RegisterSelector[TestModel](ormDB).
		Select().
		Where(Col("ID").Eq(1)).
		WithCache()

	result, err := selector.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Test User", result.Name)

	// 执行更新，并使缓存失效
	_, err = ormDB.execContext(ctx, "UPDATE test_model SET name = ? WHERE id = ?", "New Name", 1)
	require.NoError(t, err)

	// 使缓存失效 - 此行可能存在问题
	err = ormDB.InvalidateCache(ctx, "test_model", "test")
	require.NoError(t, err)

	// 添加调试日志，检查缓存是否真的被清除
	t.Logf("Cache invalidated, checking if key is now missing...")
	var testResult TestModel
	cacheErr := memCache.Get(ctx, "test_model:query:SELECT * FROM `test_model` WHERE `id` = ?;", &testResult)
	t.Logf("Cache check result: %v (expected ErrCacheMiss)", cacheErr)

	// 再次执行相同的查询，应该从数据库获取新结果
	result2, err := selector.Get(ctx)
	require.NoError(t, err)
	t.Logf("Second query result: %+v", result2)
	assert.Equal(t, "New Name", result2.Name)

	// 验证所有预期都已满足
	err = mock.ExpectationsWereMet()
	require.NoError(t, err)
}

// TestCacheTTL 测试缓存过期
func TestCacheTTL(t *testing.T) {
	// 创建模拟数据库
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// 设置两次查询的预期
	mock.ExpectQuery("SELECT .*").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "job"}).
			AddRow(1, "Test User", sql.NullString{String: "Developer", Valid: true}))

	mock.ExpectQuery("SELECT .*").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "job"}).
			AddRow(1, "Test User", sql.NullString{String: "Developer", Valid: true}))

	// 创建ORM实例并配置缓存
	ormDB, err := Open(db, "mysql")
	require.NoError(t, err)
	defer ormDB.Close()

	memCache := NewMemoryCache()
	ormDB.SetCacheManager(NewCacheManager(memCache))

	// 配置User模型的缓存，设置非常短的TTL
	ormDB.SetModelCacheConfig("test_model", &ModelCacheConfig{
		Enabled: true,
		TTL:     50 * time.Millisecond, // 非常短的过期时间
		Tags:    []string{"test"},
	})

	ctx := context.Background()

	// 执行第一次查询，这将缓存结果
	selector := RegisterSelector[TestModel](ormDB).
		Select().
		Where(Col("ID").Eq(1)).
		WithCache()

	_, err = selector.Get(ctx)
	require.NoError(t, err)

	// 等待缓存过期
	time.Sleep(100 * time.Millisecond)

	// 再次执行相同的查询，应该再次访问数据库
	_, err = selector.Get(ctx)
	require.NoError(t, err)

	// 验证所有预期都已满足
	err = mock.ExpectationsWereMet()
	require.NoError(t, err)
}

// TestCacheMiddleware 测试缓存中间件
func TestCacheMiddleware(t *testing.T) {
	// 创建模拟数据库
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	defer db.Close()

	// 设置预期查询 - 打印实际预期的SQL以便调试
	expectedSQL := "SELECT * FROM `test_model` WHERE `id` = ?;"
	t.Logf("Setting mock expectation for SQL: %s with arg: %v", expectedSQL, 1)

	mock.ExpectQuery(expectedSQL).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "job"}).
			AddRow(1, "Test User", sql.NullString{String: "Developer", Valid: true}))

	// 创建ORM实例
	ormDB, err := Open(db, "mysql")
	require.NoError(t, err)
	defer ormDB.Close()

	// 创建缓存和缓存管理器
	memCache := NewMemoryCache()
	cm := NewCacheManager(memCache)
	cm.Enable()

	// 在 ORM 实例中设置缓存管理器
	ormDB.SetCacheManager(cm)

	// 注册缓存中间件
	ormDB.Use(CacheMiddleware(cm))

	// 配置模型缓存 - 确保正确设置表名和启用状态
	ormDB.SetModelCacheConfig("test_model", &ModelCacheConfig{
		Enabled: true,
		TTL:     1 * time.Minute,
		Tags:    []string{"test"},
	})

	ctx := context.Background()

	// 执行第一次查询，应该访问数据库
	selector := RegisterSelector[TestModel](ormDB).
		Select().
		Where(Col("ID").Eq(1)).
		WithCache() // 显式启用缓存

	// 打印构建的SQL和参数
	q, err := selector.Build()
	t.Logf("Built SQL: %s with args: %v", q.SQL, q.Args)

	result1, err := selector.Get(ctx)
	if err != nil {
		t.Logf("First query error: %v", err)
	} else {
		t.Logf("First query result: %+v", result1)
	}
	require.NoError(t, err)

	// 验证SQL查询已执行
	err = mock.ExpectationsWereMet()
	if err != nil {
		t.Logf("Mock expectations error: %v", err)
	}
	require.NoError(t, err)

	// 再次执行相同的查询，应从缓存获取结果
	result2, err := selector.Get(ctx)
	if err != nil {
		t.Logf("Second query error: %v", err)
	} else {
		t.Logf("Second query result: %+v", result2)
	}
	require.NoError(t, err)

	// 验证结果正确
	assert.Equal(t, result1.ID, result2.ID)
	assert.Equal(t, result1.Name, result2.Name)

	// 验证没有额外的SQL查询执行
	err = mock.ExpectationsWereMet()
	require.NoError(t, err)
}

// TestClientCacheInterface 测试客户端缓存接口
func TestClientCacheInterface(t *testing.T) {
	// 创建模拟数据库
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// 设置预期查询
	mock.ExpectQuery("SELECT .*").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow(1, "Test Model"))

	// 创建ORM实例和客户端
	ormDB, err := Open(db, "mysql")
	require.NoError(t, err)
	defer ormDB.Close()

	// 配置缓存
	memCache := NewMemoryCache()
	ormDB.SetCacheManager(NewCacheManager(memCache))

	// 创建客户端
	client := New(ormDB)

	// 配置模型缓存
	client.SetModelCacheConfig("TestModel", &ModelCacheConfig{
		Enabled: true,
		TTL:     1 * time.Minute,
		Tags:    []string{"test_tag"},
	})

	// 无缓存查询
	collection := client.Collection(&TestModel{})
	_, err = collection.Find(context.Background())
	require.NoError(t, err)

	// 验证SQL查询已执行
	err = mock.ExpectationsWereMet()
	require.NoError(t, err)

	// 使缓存失效
	err = client.InvalidateCache(context.Background(), "TestModel", "test_tag")
	require.NoError(t, err)

	// 启用缓存的客户端
	cachedClient := client.WithCache()
	assert.NotNil(t, cachedClient)

	// 禁用缓存的客户端
	noCacheClient := client.WithoutCache()
	assert.NotNil(t, noCacheClient)
}

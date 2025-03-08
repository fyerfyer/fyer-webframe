package orm

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MigrationTestModel struct {
	ID        int       `orm:"primary_key;auto_increment"`
	Name      string    `orm:"size:255;index"`
	Email     string    `orm:"size:255;unique"`
	Age       int       `orm:"nullable:false;default:18"`
	CreatedAt time.Time `orm:"nullable:false"`
	UpdatedAt time.Time
	DeletedAt sql.NullTime
}

// MigrationTestModelChanged 表示模式变更后的模型
type MigrationTestModelChanged struct {
	ID        int       `orm:"primary_key;auto_increment"`
	Name      string    `orm:"size:255;index"`
	Email     string    `orm:"size:255;unique"`
	Age       int       `orm:"nullable:false;default:18"`
	CreatedAt time.Time `orm:"nullable:false"`
	UpdatedAt time.Time
	DeletedAt sql.NullTime
	// 新增字段
	Status int    `orm:"nullable:false;default:0"`
	Remark string `orm:"size:500"`
}

func TestAutoMigrate_CreateTable(t *testing.T) {
	// 创建mock数据库
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置表不存在的预期 - 需要返回空结果集表示表不存在
	mock.ExpectQuery("SELECT 1 FROM information_schema.tables WHERE table_name = 'migration_test_model'").
		WillReturnRows(sqlmock.NewRows([]string{"1"})) // 空结果集表示表不存在

	// 设置创建表的预期
	mock.ExpectExec("CREATE TABLE").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// 创建ORM实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)

	// 执行自动迁移，禁用迁移日志
	err = db.MigrateModel(context.Background(), &MigrationTestModel{}, WithMigrationLog(false))
	assert.NoError(t, err)

	// 验证预期
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAutoMigrate_TableExists(t *testing.T) {
	// 创建mock数据库
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置表已存在的预期 - 注意这里使用正确的表名
	mock.ExpectQuery("SELECT 1 FROM information_schema.tables WHERE table_name = 'migration_test_model_changed'").
		WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

	// 设置获取现有表结构的预期
	mock.ExpectQuery(".*FROM.*INFORMATION_SCHEMA.COLUMNS.*").
		WillReturnRows(sqlmock.NewRows([]string{
			"COLUMN_NAME", "DATA_TYPE", "IS_NULLABLE", "COLUMN_DEFAULT",
			"CHARACTER_MAXIMUM_LENGTH", "NUMERIC_PRECISION", "NUMERIC_SCALE", "COLUMN_KEY", "EXTRA"}).
			AddRow("id", "int", "NO", nil, nil, 10, 0, "PRI", "auto_increment").
			AddRow("name", "varchar", "YES", nil, 255, nil, nil, "MUL", "").
			AddRow("email", "varchar", "YES", nil, 255, nil, nil, "UNI", "").
			AddRow("age", "int", "NO", "18", nil, 10, 0, "", "").
			AddRow("created_at", "datetime", "NO", nil, nil, nil, nil, "", "").
			AddRow("updated_at", "datetime", "YES", nil, nil, nil, nil, "", "").
			AddRow("deleted_at", "datetime", "YES", nil, nil, nil, nil, "", ""))

	// 设置修改表的预期
	mock.ExpectExec("ALTER TABLE").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// 创建ORM实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)

	// 执行自动迁移，使用AlterIfNeeded策略
	err = db.MigrateModel(context.Background(), &MigrationTestModelChanged{},
		WithStrategy(AlterIfNeeded),
		WithMigrationLog(false))
	assert.NoError(t, err)

	// 验证预期
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAutoMigrate_DryRun(t *testing.T) {
	// 创建mock数据库
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置表不存在的预期
	mock.ExpectQuery(regexp.QuoteMeta("SELECT 1 FROM information_schema.tables WHERE table_name = 'migration_test_model'")).
		WillReturnRows(sqlmock.NewRows([]string{"1"}))

	// 注意：在DryRun模式下不应该执行任何DDL语句

	// 创建ORM实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)

	// 执行自动迁移，使用DryRun模式
	err = db.MigrateModel(context.Background(), &MigrationTestModel{}, WithDryRun(true))
	assert.NoError(t, err)

	// 验证预期
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAutoMigrate_Callback(t *testing.T) {
	// 创建mock数据库
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置表不存在的预期
	mock.ExpectQuery(regexp.QuoteMeta("SELECT 1 FROM information_schema.tables WHERE table_name = 'migration_test_model'")).
		WillReturnRows(sqlmock.NewRows([]string{"1"}))

	// 设置创建表的预期
	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE")).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// 创建ORM实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)

	var callbackCalled bool
	var callbackMigration *Migration

	// 执行自动迁移，使用回调函数
	err = db.MigrateModel(context.Background(), &MigrationTestModel{}, WithMigrationCallback(func(m *Migration) {
		callbackCalled = true
		callbackMigration = m
	}), WithMigrationLog(false))
	assert.NoError(t, err)

	// 验证回调被调用
	assert.True(t, callbackCalled, "Migration callback should be called")
	assert.NotNil(t, callbackMigration, "Migration object should be passed to callback")
	assert.Equal(t, "migration_test_model", callbackMigration.TableName)

	// 验证预期
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAutoMigrate_ForceRecreate(t *testing.T) {
	// 创建mock数据库
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置表已存在的预期
	mock.ExpectQuery(regexp.QuoteMeta("SELECT 1 FROM information_schema.tables WHERE table_name = 'migration_test_model'")).
		WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

	// 设置删除和创建表的预期
	mock.ExpectExec(regexp.QuoteMeta("DROP TABLE")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE")).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// 创建ORM实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)

	// 执行自动迁移，使用ForceRecreate策略
	err = db.MigrateModel(context.Background(), &MigrationTestModel{},
		WithStrategy(ForceRecreate),
		WithMigrationLog(false))
	assert.NoError(t, err)

	// 验证预期
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAutoMigrate_CreateMigrationLog(t *testing.T) {
	// 创建mock数据库
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置表不存在的预期
	mock.ExpectQuery(regexp.QuoteMeta("SELECT 1 FROM information_schema.tables WHERE table_name = 'migration_test_model'")).
		WillReturnRows(sqlmock.NewRows([]string{"1"}))

	// 设置创建表的预期
	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE")).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// 设置迁移日志表预期
	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE IF NOT EXISTS orm_migration_log")).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// 设置检查迁移记录预期
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM orm_migration_log")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	// 设置插入迁移记录预期
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO orm_migration_log")).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 创建ORM实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)

	// 执行自动迁移，启用迁移日志
	err = db.MigrateModel(context.Background(), &MigrationTestModel{}, WithMigrationLog(true))
	assert.NoError(t, err)

	// 验证预期
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAutoMigrate_MultipleModels(t *testing.T) {
	// 创建mock数据库
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 为第一个模型设置预期
	mock.ExpectQuery(regexp.QuoteMeta("SELECT 1 FROM information_schema.tables WHERE table_name = 'migration_test_model'")).
		WillReturnRows(sqlmock.NewRows([]string{"1"}))
	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE")).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// 为第二个模型设置预期
	mock.ExpectQuery(regexp.QuoteMeta("SELECT 1 FROM information_schema.tables WHERE table_name = 'migration_test_model_changed'")).
		WillReturnRows(sqlmock.NewRows([]string{"1"}))
	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE")).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// 创建ORM实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)

	// 执行多模型自动迁移
	err = db.AutoMigrateWithOptions(context.Background(), []MigrateOption{
		WithMigrationLog(false)},
		&MigrationTestModel{}, &MigrationTestModelChanged{})
	assert.NoError(t, err)

	// 验证预期
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDifferentDialects(t *testing.T) {
	testCases := []struct {
		name           string
		dialectName    string
		expectedCreate string
	}{
		{
			name:           "mysql",
			dialectName:    "mysql",
			expectedCreate: "CREATE TABLE `migration_test_model` .*ENGINE=InnoDB DEFAULT CHARSET=utf8mb4.*",
		},
		{
			name:           "postgresql",
			dialectName:    "postgresql",
			expectedCreate: "CREATE TABLE \"migration_test_model\".*",
		},
		{
			name:           "sqlite",
			dialectName:    "sqlite",
			expectedCreate: "CREATE TABLE \"migration_test_model\".*",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建mock数据库
			mockDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer mockDB.Close()

			// 设置表不存在的预期
			mock.ExpectQuery("SELECT 1 FROM").
				WillReturnRows(sqlmock.NewRows([]string{"1"}))

			// 设置创建表的预期，使用正则匹配不同方言生成的SQL
			mock.ExpectExec(tc.expectedCreate).
				WillReturnResult(sqlmock.NewResult(0, 0))

			// 创建ORM实例
			db, err := Open(mockDB, tc.dialectName)
			require.NoError(t, err)

			// 执行自动迁移
			err = db.AutoMigrateWithOptions(context.Background(),
				[]MigrateOption{WithMigrationLog(false)},
				&MigrationTestModel{})
			assert.NoError(t, err)

			// 验证预期
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRegisterModel_WithAutoMigrate(t *testing.T) {
	// 创建mock数据库
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置表不存在的预期
	mock.ExpectQuery(regexp.QuoteMeta("SELECT 1 FROM information_schema.tables WHERE table_name = 'migration_test_model'")).
		WillReturnRows(sqlmock.NewRows([]string{"1"}))

	// 设置创建表的预期
	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE")).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// 创建ORM实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)

	// 注册模型并自动迁移
	err = db.RegisterModel("test_model", &MigrationTestModel{}, true,
		WithMigrationLog(false))
	assert.NoError(t, err)

	// 验证模型是否已注册
	model, ok := DefaultModelRegistry.Get("test_model")
	assert.True(t, ok, "Model should be registered")
	assert.NotNil(t, model, "Model should not be nil")

	// 验证预期
	assert.NoError(t, mock.ExpectationsWereMet())
}

func Test_isTableChanged(t *testing.T) {
	// 创建mock数据库
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 创建ORM实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)

	// 获取原始模型和变更后模型的元数据
	model1, err := db.getModel(&MigrationTestModel{})
	require.NoError(t, err)

	model2, err := db.getModel(&MigrationTestModelChanged{})
	require.NoError(t, err)

	// 检测表结构是否变化
	changed := db.schemaManager.isTableChanged(model2, model1)
	assert.True(t, changed, "Table structure should be detected as changed")

	// 相同模型应该不被检测为变化
	changed = db.schemaManager.isTableChanged(model1, model1)
	assert.False(t, changed, "Same models should not be detected as changed")
}

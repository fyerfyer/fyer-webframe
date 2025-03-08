package orm

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Collection(t *testing.T) {
	// 创建 mock 数据库和连接
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 创建 ORM DB 实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)
	defer db.Close()

	// 创建客户端实例
	client := New(db)

	// 创建一个测试模型的集合
	collection := client.Collection(&TestModel{})
	assert.NotNil(t, collection)
	assert.Equal(t, "TestModel", collection.modelName)
}

func TestCollection_Find(t *testing.T) {
	// 创建 mock 数据库和连接
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置预期查询
	mock.ExpectQuery("SELECT (.+) FROM `test_model` WHERE `id` = ?").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "job"}).
			AddRow(1, "Test User", sql.NullString{String: "Developer", Valid: true}))

	// 创建 ORM DB 实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)
	defer db.Close()

	// 创建客户端实例
	client := New(db)

	// 创建一个测试模型的集合
	collection := client.Collection(&TestModel{})

	// 执行查找操作
	result, err := collection.Find(context.Background(), Col("ID").Eq(1))
	require.NoError(t, err)

	// 验证结果
	testModel, ok := result.(*TestModel)
	require.True(t, ok)
	assert.Equal(t, 1, testModel.ID)
	assert.Equal(t, "Test User", testModel.Name)
	assert.Equal(t, "Developer", testModel.Job.String)
	assert.True(t, testModel.Job.Valid)

	// 验证所有预期的SQL语句都已执行
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCollection_FindAll(t *testing.T) {
	// 创建 mock 数据库和连接
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置预期查询
	mock.ExpectQuery("SELECT (.+) FROM `test_model` WHERE `id` > ?").
		WithArgs(0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "job"}).
			AddRow(1, "User 1", sql.NullString{String: "Developer", Valid: true}).
			AddRow(2, "User 2", sql.NullString{String: "Designer", Valid: true}))

	// 创建 ORM DB 实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)
	defer db.Close()

	// 创建客户端实例
	client := New(db)

	// 创建一个测试模型的集合
	collection := client.Collection(&TestModel{})

	// 执行查找操作
	results, err := collection.FindAll(context.Background(), Col("ID").Gt(0))
	require.NoError(t, err)

	// 验证结果
	require.Len(t, results, 2)

	// 验证第一个结果
	testModel1, ok := results[0].(*TestModel)
	require.True(t, ok)
	assert.Equal(t, 1, testModel1.ID)
	assert.Equal(t, "User 1", testModel1.Name)
	assert.Equal(t, "Developer", testModel1.Job.String)

	// 验证第二个结果
	testModel2, ok := results[1].(*TestModel)
	require.True(t, ok)
	assert.Equal(t, 2, testModel2.ID)
	assert.Equal(t, "User 2", testModel2.Name)
	assert.Equal(t, "Designer", testModel2.Job.String)

	// 验证所有预期的SQL语句都已执行
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCollection_FindWithOptions(t *testing.T) {
	// 创建 mock 数据库和连接
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置预期查询，包含 ORDER BY, LIMIT, OFFSET 子句
	mock.ExpectQuery("^SELECT \\* FROM `test_model` WHERE `id` > \\? ORDER BY `name` DESC LIMIT 10 OFFSET 20;$  \n").
		WithArgs(0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "job"}).
			AddRow(2, "User 2", sql.NullString{String: "Designer", Valid: true}).
			AddRow(1, "User 1", sql.NullString{String: "Developer", Valid: true}))

	// 创建 ORM DB 实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)
	defer db.Close()

	// 创建客户端实例
	client := New(db)

	// 创建一个测试模型的集合
	collection := client.Collection(&TestModel{})

	// 创建查询选项
	options := FindOptions{
		Limit:   10,
		Offset:  20,
		OrderBy: []OrderBy{Desc(Col("Name"))},
	}

	// 执行带选项的查找操作
	results, err := collection.FindWithOptions(context.Background(), options, Col("ID").Gt(0))
	require.NoError(t, err)

	// 验证结果
	require.Len(t, results, 2)
	assert.Equal(t, "User 2", results[0].(*TestModel).Name) // 确认按名称降序排序
	assert.Equal(t, "User 1", results[1].(*TestModel).Name)

	// 验证所有预期的SQL语句都已执行
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCollection_Insert(t *testing.T) {
	// 创建 mock 数据库和连接
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置预期查询
	mock.ExpectExec("^INSERT INTO `test_model` \\(`id`, `name`, `job`\\) VALUES \\(\\?, \\?, \\?\\);$").
		WithArgs(3, "New User", sql.NullString{String: "Manager", Valid: true}).
		WillReturnResult(sqlmock.NewResult(3, 1))

	// 创建 ORM DB 实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)
	defer db.Close()

	// 创建客户端实例
	client := New(db)

	// 创建一个测试模型的集合
	collection := client.Collection(&TestModel{})

	// 创建要插入的模型
	newModel := &TestModel{
		ID:   3,
		Name: "New User",
		Job:  sql.NullString{String: "Manager", Valid: true},
	}

	// 执行插入操作
	result, err := collection.Insert(context.Background(), newModel)
	require.NoError(t, err)

	// 验证受影响的行数
	rowsAffected, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)

	// 验证所有预期的SQL语句都已执行
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCollection_Update(t *testing.T) {
	// 创建 mock 数据库和连接
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置预期查询
	mock.ExpectExec("UPDATE `test_model` SET `name` = \\?, `job` = \\? WHERE `id` = \\?").
		WithArgs("Updated User", sql.NullString{String: "Senior Developer", Valid: true}, 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 创建 ORM DB 实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)
	defer db.Close()

	// 创建客户端实例
	client := New(db)

	// 创建一个测试模型的集合
	collection := client.Collection(&TestModel{})

	// 准备更新数据
	updates := map[string]interface{}{
		"name": "Updated User",
		"job":  sql.NullString{String: "Senior Developer", Valid: true},
	}

	// 执行更新操作
	result, err := collection.Update(context.Background(), updates, Col("ID").Eq(1))
	require.NoError(t, err)

	// 验证受影响的行数
	rowsAffected, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)

	// 验证所有预期的SQL语句都已执行
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCollection_Delete(t *testing.T) {
	// 创建 mock 数据库和连接
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置预期查询
	mock.ExpectExec("DELETE FROM `test_model` WHERE `id` = \\?").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// 创建 ORM DB 实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)
	defer db.Close()

	// 创建客户端实例
	client := New(db)

	// 创建一个测试模型的集合
	collection := client.Collection(&TestModel{})

	// 执行删除操作
	result, err := collection.Delete(context.Background(), Col("ID").Eq(1))
	require.NoError(t, err)

	// 验证受影响的行数
	rowsAffected, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)

	// 验证所有预期的SQL语句都已执行
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestClient_Transaction(t *testing.T) {
	// 创建 mock 数据库和连接
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置事务预期
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `test_model` \\(`id`, `name`, `job`\\) VALUES \\(\\?, \\?, \\?\\)").
		WithArgs(4, "Transaction User", sql.NullString{String: "Tester", Valid: true}).
		WillReturnResult(sqlmock.NewResult(4, 1))
	mock.ExpectCommit()

	// 创建 ORM DB 实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)
	defer db.Close()

	// 创建客户端实例
	client := New(db)

	// 执行事务
	err = client.Transaction(context.Background(), func(tc *Client) error {
		// 在事务中创建集合
		collection := tc.Collection(&TestModel{})

		// 创建要插入的模型
		newModel := &TestModel{
			ID:   4,
			Name: "Transaction User",
			Job:  sql.NullString{String: "Tester", Valid: true},
		}

		// 执行插入操作
		_, err := collection.Insert(context.Background(), newModel)
		return err
	})

	require.NoError(t, err)

	// 验证所有预期的SQL语句都已执行
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestClient_TransactionRollback(t *testing.T) {
	// 创建 mock 数据库和连接
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置事务预期，包括回滚
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `test_model` \\(`id`, `name`, `job`\\) VALUES \\(\\?, \\?, \\?\\)").
		WithArgs(5, "Rollback User", sql.NullString{String: "Failed", Valid: true}).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	// 创建 ORM DB 实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)
	defer db.Close()

	// 创建客户端实例
	client := New(db)

	// 执行事务，预期失败并回滚
	err = client.Transaction(context.Background(), func(tc *Client) error {
		// 在事务中创建集合
		collection := tc.Collection(&TestModel{})

		// 创建要插入的模型
		newModel := &TestModel{
			ID:   5,
			Name: "Rollback User",
			Job:  sql.NullString{String: "Failed", Valid: true},
		}

		// 执行插入操作，预期失败
		_, err := collection.Insert(context.Background(), newModel)
		return err
	})

	// 应该返回错误
	require.Error(t, err)
	assert.Equal(t, sql.ErrNoRows, err)

	// 验证所有预期的SQL语句都已执行
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestClient_Count(t *testing.T) {
	// 创建 mock 数据库和连接
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置预期查询
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM `test_model` WHERE `id` > \\?").
		WithArgs(0).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(5))

	// 创建 ORM DB 实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)
	defer db.Close()

	// 创建客户端实例
	client := New(db)

	// 执行计数查询
	count, err := client.Count(context.Background(), &TestModel{}, Col("ID").Gt(0))
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)

	// 验证所有预期的SQL语句都已执行
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestClient_RegisterModel(t *testing.T) {
	// 创建 mock 数据库和连接
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 创建 ORM DB 实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)
	defer db.Close()

	// 创建客户端实例
	client := New(db)

	// 注册模型
	client.RegisterModel("test_model", &TestModel{})

	// 获取已注册的模型
	model, ok := client.GetRegisteredModel("test_model")
	require.True(t, ok)

	// 验证模型类型
	testModel, ok := model.(*TestModel)
	require.True(t, ok)
	assert.IsType(t, &TestModel{}, testModel)
}

func TestClient_Exec(t *testing.T) {
	// 创建 mock 数据库和连接
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置预期执行
	mock.ExpectExec("UPDATE `test_model` SET `status` = \\? WHERE `id` > \\?").
		WithArgs("active", 10).
		WillReturnResult(sqlmock.NewResult(0, 5))

	// 创建 ORM DB 实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)
	defer db.Close()

	// 创建客户端实例
	client := New(db)

	// 执行自定义SQL
	result, err := client.Exec(context.Background(), "UPDATE `test_model` SET `status` = ? WHERE `id` > ?", "active", 10)
	require.NoError(t, err)

	// 验证受影响的行数
	rowsAffected, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(5), rowsAffected)

	// 验证所有预期的SQL语句都已执行
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestClient_Raw(t *testing.T) {
	// 创建 mock 数据库和连接
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	// 设置预期查询
	mock.ExpectQuery("SELECT \\* FROM `test_model` WHERE `id` = \\?").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow(1, "Raw Query"))

	// 创建 ORM DB 实例
	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)
	defer db.Close()

	// 创建客户端实例
	client := New(db)

	// 执行原始SQL查询
	rows, err := client.Raw(context.Background(), "SELECT * FROM `test_model` WHERE `id` = ?", 1)
	require.NoError(t, err)
	defer rows.Close()

	// 验证结果
	assert.True(t, rows.Next())
	var id int
	var name string
	err = rows.Scan(&id, &name)
	require.NoError(t, err)
	assert.Equal(t, 1, id)
	assert.Equal(t, "Raw Query", name)

	// 验证所有预期的SQL语句都已执行
	assert.NoError(t, mock.ExpectationsWereMet())
}
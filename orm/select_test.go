package orm

import (
	"context"
	"database/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/fyerfyer/fyer-webframe/orm/internal/ferr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

type TestModel struct {
	ID   int
	Name string
	Job  sql.NullString
}

type TestModelWithTableNameInterface struct {
	ID int
}

func (t TestModelWithTableNameInterface) TableName() string {
	return "test_model"
}

type TestModelWithTableNameInterfacePtr struct {
	ID int
}

func (t *TestModelWithTableNameInterfacePtr) TableName() string {
	return "test_model"
}

type TestModelWithTag struct {
	ID   int    `orm:"column_name:testid"`
	Name string `orm:"column_name:testname"`
	Job  sql.NullString
}

func TestSelector_Build(t *testing.T) {
	// 使用 sqlmock
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		q         *Selector[TestModel]
		wantQuery *Query
		wantErr   error
	}{
		{
			name: "simple from",
			q:    RegisterSelector[TestModel](db).Select(),
			wantQuery: &Query{
				SQL:  "SELECT * FROM `test_model`;",
				Args: nil,
			},
		},
		{
			// 指定列
			name: "with columns",
			q:    RegisterSelector[TestModel](db).Select("ID", "Name"),
			wantQuery: &Query{
				SQL:  "SELECT `id`, `name` FROM `test_model`;",
				Args: nil,
			},
		},
		{
			name: "with simple where",
			q: RegisterSelector[TestModel](db).Select().
				Where(Col("ID").Eq(12)),
			wantQuery: &Query{
				SQL:  "SELECT * FROM `test_model` WHERE `id` = ?;",
				Args: []any{&Value{val: 12}}, // 修改这里，使用指针类型
			},
		},
		{
			name: "with complex where",
			q: RegisterSelector[TestModel](db).Select().
				Where(NOT(Col("ID").Eq(12))),
			wantQuery: &Query{
				SQL:  "SELECT * FROM `test_model` WHERE NOT `id` = ?;",
				Args: []any{&Value{val: 12}}, // 修改这里，使用指针类型
			},
		},
		{
			name: "with multiple where",
			q: RegisterSelector[TestModel](db).Select("ID").
				Where(Col("ID").Eq(12), Col("Job").IsNull()),
			wantQuery: &Query{
				SQL:  "SELECT `id` FROM `test_model` WHERE `id` = ? AND `job` IS NULL;",
				Args: []any{&Value{val: 12}}, // 修改这里，使用指针类型
			},
		},
		{
			name: "with nonexist column",
			q:    RegisterSelector[TestModel](db),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "with nonexist column" {
				assert.Panics(t, func() {
					tc.q.Select("ID", "nonexist").Build()
				})
				return
			}
			query, err := tc.q.Build()
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			assert.Equal(t, tc.wantQuery, query)
		})
	}
}

func TestTableNameInterface(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		q         *Selector[TestModelWithTableNameInterface]
		wantQuery *Query
		wantErr   error
	}{
		{
			name: "simple from",
			q:    RegisterSelector[TestModelWithTableNameInterface](db).Select(),
			wantQuery: &Query{
				SQL:  "SELECT * FROM `test_model`;",
				Args: nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			query, err := tc.q.Build()
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			assert.Equal(t, tc.wantQuery, query)
		})
	}

	ptrTestCases := []struct {
		name      string
		q         *Selector[TestModelWithTableNameInterfacePtr]
		wantQuery *Query
		wantErr   error
	}{
		{
			name: "simple from with ptr",
			q:    RegisterSelector[TestModelWithTableNameInterfacePtr](db).Select(),
			wantQuery: &Query{
				SQL:  "SELECT * FROM `test_model`;",
				Args: nil,
			},
		},
	}

	for _, tc := range ptrTestCases {
		t.Run(tc.name, func(t *testing.T) {
			query, err := tc.q.Build()
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			assert.Equal(t, tc.wantQuery, query)
		})
	}
}

func TestSelectorTag(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		q         *Selector[TestModelWithTag]
		wantQuery *Query
		wantErr   error
	}{
		{
			// 指定列
			name: "with columns",
			q:    RegisterSelector[TestModelWithTag](db).Select("ID", "Name"),
			wantQuery: &Query{
				SQL:  "SELECT `testid`, `testname` FROM `test_model_with_tag`;",
				Args: nil,
			},
		},
		{
			name: "with simple where",
			q: RegisterSelector[TestModelWithTag](db).Select().
				Where(Col("ID").Eq(12)),
			wantQuery: &Query{
				SQL:  "SELECT * FROM `test_model_with_tag` WHERE `testid` = ?;",
				Args: []any{&Value{val: 12}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			query, err := tc.q.Build()
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			assert.Equal(t, tc.wantQuery, query)
		})
	}
}

func TestSelector_Get(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB)
	require.NoError(t, err)

	testCases := []struct {
		name     string
		query    string
		mockRows *sqlmock.Rows
		wantErr  error
		wantRes  *TestModel
	}{
		{
			name:  "single row",
			query: "SELECT \\* FROM `test_model` WHERE `id` = \\?;",
			mockRows: sqlmock.NewRows([]string{"id", "name", "job"}).
				AddRow(1, "Tom", sql.NullString{String: "programmer", Valid: true}),
			wantRes: &TestModel{
				ID:   1,
				Name: "Tom",
				Job:  sql.NullString{String: "programmer", Valid: true},
			},
		},
		{
			name:     "no rows",
			query:    "SELECT \\* FROM `test_model` WHERE `id` = \\?;",
			mockRows: sqlmock.NewRows([]string{"id", "name", "job"}),
			wantErr:  ferr.ErrNoRows,
		},
		{
			name:  "multiple rows",
			query: "SELECT \\* FROM `test_model` WHERE `id` = \\?;",
			mockRows: sqlmock.NewRows([]string{"id", "name", "job"}).
				AddRow(1, "Tom", sql.NullString{String: "programmer", Valid: true}).
				AddRow(2, "Jerry", sql.NullString{String: "teacher", Valid: true}),
			wantErr: ferr.ErrTooManyRows,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock.ExpectQuery(tc.query).
				WithArgs(1).
				WillReturnRows(tc.mockRows)

			res, err := RegisterSelector[TestModel](db).
				Select().
				Where(Col("ID").Eq(1)).
				Get(context.Background())

			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			assert.Equal(t, tc.wantRes, res)
		})
	}
}

func TestSelector_GetMulti(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB)
	require.NoError(t, err)

	testCases := []struct {
		name     string
		query    string
		mockRows *sqlmock.Rows
		wantErr  error
		wantRes  []*TestModel
	}{
		{
			name:  "multiple rows",
			query: "SELECT \\* FROM `test_model` WHERE `age` > \\?;",
			mockRows: sqlmock.NewRows([]string{"id", "name", "job"}).
				AddRow(1, "Tom", sql.NullString{String: "programmer", Valid: true}).
				AddRow(2, "Jerry", sql.NullString{String: "teacher", Valid: true}),
			wantRes: []*TestModel{
				{
					ID:   1,
					Name: "Tom",
					Job:  sql.NullString{String: "programmer", Valid: true},
				},
				{
					ID:   2,
					Name: "Jerry",
					Job:  sql.NullString{String: "teacher", Valid: true},
				},
			},
		},
		{
			name:     "no rows",
			query:    "SELECT \\* FROM `test_model` WHERE `age` > \\?;",
			mockRows: sqlmock.NewRows([]string{"id", "name", "job"}),
			wantRes:  []*TestModel(nil),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock.ExpectQuery(tc.query).
				WithArgs(18).
				WillReturnRows(tc.mockRows)

			res, err := RegisterSelector[TestModel](db).
				Select().
				Where(Col("Age").Gt(18)).
				GetMulti(context.Background())

			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			assert.Equal(t, tc.wantRes, res)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

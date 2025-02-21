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

type TestModel2 struct {
	ID   int
	Name string
	Age  int
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

	db, err := Open(mockDB, "mysql")
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
			q:    RegisterSelector[TestModel](db).Select(Col("ID"), Col("Name")),
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
				Args: []any{12},
			},
		},
		{
			name: "with complex where",
			q: RegisterSelector[TestModel](db).Select().
				Where(NOT(Col("ID").Eq(12))),
			wantQuery: &Query{
				SQL:  "SELECT * FROM `test_model` WHERE NOT `id` = ?;",
				Args: []any{12},
			},
		},
		{
			name: "with multiple where",
			q: RegisterSelector[TestModel](db).Select(Col("ID")).
				Where(Col("ID").Eq(12), Col("Job").IsNull()),
			wantQuery: &Query{
				SQL:  "SELECT `id` FROM `test_model` WHERE `id` = ? AND `job` IS NULL;",
				Args: []any{12},
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
					tc.q.Select(Col("ID"), Col("nonexist")).Build()
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

	db, err := Open(mockDB, "mysql")
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

	db, err := Open(mockDB, "mysql")
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
			q:    RegisterSelector[TestModelWithTag](db).Select(Col("ID"), Col("Name")),
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

	db, err := Open(mockDB, "mysql")
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

	db, err := Open(mockDB, "mysql")
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

func TestSelector_Aggregate(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)

	testCases := []struct {
		name      string
		q         *Selector[TestModel]
		wantQuery *Query
		wantErr   error
	}{
		{
			name: "count",
			q: RegisterSelector[TestModel](db).
				Select(Count("Age")),
			wantQuery: &Query{
				SQL:  "SELECT COUNT(`age`) FROM `test_model`;",
				Args: nil,
			},
		},
		{
			name: "count distinct",
			q: RegisterSelector[TestModel](db).
				Select(CountDistinct("Age")),
			wantQuery: &Query{
				SQL:  "SELECT COUNT(DISTINCT `age`) FROM `test_model`;",
				Args: nil,
			},
		},
		{
			name: "sum",
			q: RegisterSelector[TestModel](db).
				Select(Sum("Age")),
			wantQuery: &Query{
				SQL:  "SELECT SUM(`age`) FROM `test_model`;",
				Args: nil,
			},
		},
		{
			name: "avg",
			q: RegisterSelector[TestModel](db).
				Select(Avg("Age")),
			wantQuery: &Query{
				SQL:  "SELECT AVG(`age`) FROM `test_model`;",
				Args: nil,
			},
		},
		{
			name: "max",
			q: RegisterSelector[TestModel](db).
				Select(Max("Age")),
			wantQuery: &Query{
				SQL:  "SELECT MAX(`age`) FROM `test_model`;",
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
}

func TestSelector_As(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)

	testCases := []struct {
		name      string
		q         *Selector[TestModel]
		wantQuery *Query
		wantErr   error
	}{
		{
			name: "column with alias",
			q: RegisterSelector[TestModel](db).
				Select(Col("ID").As("user_id")),
			wantQuery: &Query{
				SQL:  "SELECT `id` AS `user_id` FROM `test_model`;",
				Args: nil,
			},
		},
		{
			name: "aggregate with alias",
			q: RegisterSelector[TestModel](db).
				Select(Count("Age").As("total_age")),
			wantQuery: &Query{
				SQL:  "SELECT COUNT(`age`) AS `total_age` FROM `test_model`;",
				Args: nil,
			},
		},
		{
			name: "mixed columns with alias",
			q: RegisterSelector[TestModel](db).
				Select(
					Col("ID").As("user_id"),
					Count("Age").As("total_age"),
					Avg("Age").As("avg_age"),
				),
			wantQuery: &Query{
				SQL:  "SELECT `id` AS `user_id`, COUNT(`age`) AS `total_age`, AVG(`age`) AS `avg_age` FROM `test_model`;",
				Args: nil,
			},
		},
		{
			name: "where with alias",
			q : RegisterSelector[TestModel](db),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "where with alias" {
				assert.Panics(t, func() {
					tc.q.Select(Col("ID").As("user_id")).Where(Col("user_id").Eq(1)).Build()
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

func TestSelector_Raw(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)

	testCases := []struct {
		name      string
		q         *Selector[TestModel]
		wantQuery *Query
		wantErr   error
	}{
		{
			name: "raw expression",
			q: RegisterSelector[TestModel](db).
				Select(Raw("CASE WHEN `age` > 18 THEN 'adult' ELSE 'minor' END as age_group")),
			wantQuery: &Query{
				SQL:  "SELECT CASE WHEN `age` > 18 THEN 'adult' ELSE 'minor' END as age_group FROM `test_model`;",
				Args: nil,
			},
		},
		{
			name: "raw with args",
			q: RegisterSelector[TestModel](db).
				Select(Raw("CASE WHEN `age` > ? THEN ? ELSE ? END as age_group", 18, "adult", "minor")),
			wantQuery: &Query{
				SQL:  "SELECT CASE WHEN `age` > ? THEN ? ELSE ? END as age_group FROM `test_model`;",
				Args: []any{18, "adult", "minor"},
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

func TestSelector_GroupBy(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)

	testCases := []struct {
		name      string
		q         *Selector[TestModel]
		wantQuery *Query
		wantErr   error
	}{
		{
			name: "group by",
			q: RegisterSelector[TestModel](db).
				Select(Col("ID")).
				GroupBy(Col("Name")),
			wantQuery: &Query{
				SQL:  "SELECT `id` FROM `test_model` GROUP BY `name`;",
				Args: nil,
			},
		},
		{
			name: "group by multiple columns",
			q: RegisterSelector[TestModel](db).
				Select(Col("ID")).
				GroupBy(Col("Job"), Col("Name")),
			wantQuery: &Query{
				SQL:  "SELECT `id` FROM `test_model` GROUP BY (`job`, `name`);",
				Args: nil,
			},
		},
		{
			name: "group by with aggregate",
			q: RegisterSelector[TestModel](db).
				Select(Count("ID")).
				GroupBy(Count("Name")),
			wantQuery: &Query{
				SQL: "SELECT COUNT(`id`) FROM `test_model` GROUP BY COUNT(`name`);",
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
}

func TestSelector_OrderBy(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)

	testCases := []struct {
		name      string
		q         *Selector[TestModel]
		wantQuery *Query
		wantErr   error
	}{
		{
			name: "order by column",
			q: RegisterSelector[TestModel](db).
				Select(Col("ID")).
				OrderBy(Asc(Col("ID"))),
			wantQuery: &Query{
				SQL:  "SELECT `id` FROM `test_model` ORDER BY `id`;",
				Args: nil,
			},
		},
		{
			name: "order by alias",
			q: RegisterSelector[TestModel](db).
				Select(Col("ID").As("user_id")).
				OrderBy(Desc(Col("user_id"))),
			wantQuery: &Query{
				SQL:  "SELECT `id` AS `user_id` FROM `test_model` ORDER BY `user_id` DESC;",
				Args: nil,
			},
		},
		{
			name: "aggregate order by",
			q: RegisterSelector[TestModel](db).
				Select(Count("ID").As("count")).
				GroupBy(Col("Job")).
				OrderBy(Desc(Count("ID"))),
			wantQuery: &Query{
				SQL:  "SELECT COUNT(`id`) AS `count` FROM `test_model` GROUP BY `job` ORDER BY COUNT(`id`) DESC;",
				Args: nil,
			},
		},
		{
			name: "multiple order by",
			q: RegisterSelector[TestModel](db).
				Select(Col("Name"), Count("ID").As("count")).
				GroupBy(Col("Name")).
				OrderBy(Asc(Col("Name")), Desc(Count("ID"))),
			wantQuery: &Query{
				SQL:  "SELECT `name`, COUNT(`id`) AS `count` FROM `test_model` GROUP BY `name` ORDER BY `name`, COUNT(`id`) DESC;",
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
}

func TestSelector_Having(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)

	testCases := []struct {
		name      string
		q         *Selector[TestModel2]
		wantQuery *Query
		wantErr   error
	}{
		{
			name: "having with aggregate",
			q: RegisterSelector[TestModel2](db).
				Select(Col("Name"), Count("ID").As("count")).
				GroupBy(Col("Name")).
				Having(Count("ID").Gt(100)),
			wantQuery: &Query{
				SQL:  "SELECT `name`, COUNT(`id`) AS `count` FROM `test_model` GROUP BY `name` HAVING COUNT(`id`) > ?;",
				Args: []any{100},
			},
		},
		{
			name: "having with alias",
			q: RegisterSelector[TestModel2](db).
				Select(Col("Name"), Count("ID").As("count")).
				GroupBy(Col("Name")).
				Having(Col("count").Gt(100)),
			wantQuery: &Query{
				SQL:  "SELECT `name`, COUNT(`id`) AS `count` FROM `test_model` GROUP BY `name` HAVING `count` > ?;",
				Args: []any{100},
			},
		},
		{
			name: "having with multiple conditions",
			q: RegisterSelector[TestModel2](db).
				Select(Col("Name"), Count("ID").As("count"), Avg("Age").As("avg_age")).
				GroupBy(Col("Name")).
				Having(Count("ID").Gt(100), Avg("Age").Gt(18)),
			wantQuery: &Query{
				SQL:  "SELECT `name`, COUNT(`id`) AS `count`, AVG(`age`) AS `avg_age` FROM `test_model` GROUP BY `name` HAVING COUNT(`id`) > ? AND AVG(`age`) > ?;",
				Args: []any{100, 18},
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
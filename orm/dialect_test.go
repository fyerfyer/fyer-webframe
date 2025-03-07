package orm

import (
	"context"
	"database/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDialect_Quote(t *testing.T) {
	testCases := []struct {
		name     string
		dialect  Dialect
		input    string
		expected string
	}{
		{
			name:     "mysql",
			dialect:  &Mysql{},
			input:    "user_name",
			expected: "`user_name`",
		},
		{
			name:     "postgresql",
			dialect:  &Postgresql{},
			input:    "user_name",
			expected: "\"user_name\"",
		},
		{
			name:     "sqlite",
			dialect:  &Sqlite{},
			input:    "user_name",
			expected: "\"user_name\"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.dialect.Quote(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDialect_Placeholder(t *testing.T) {
	testCases := []struct {
		name     string
		dialect  Dialect
		index    int
		expected string
	}{
		{
			name:     "mysql",
			dialect:  &Mysql{},
			index:    1,
			expected: "?",
		},
		{
			name:     "postgresql",
			dialect:  &Postgresql{},
			index:    1,
			expected: "$1",
		},
		{
			name:     "postgresql multiple",
			dialect:  &Postgresql{},
			index:    3,
			expected: "$3",
		},
		{
			name:     "sqlite",
			dialect:  &Sqlite{},
			index:    1,
			expected: "?",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.dialect.Placeholder(tc.index)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDialect_Concat(t *testing.T) {
	testCases := []struct {
		name     string
		dialect  Dialect
		items    []string
		expected string
	}{
		{
			name:     "mysql",
			dialect:  &Mysql{},
			items:    []string{"first_name", "last_name"},
			expected: "CONCAT(first_name, last_name)",
		},
		{
			name:     "postgresql",
			dialect:  &Postgresql{},
			items:    []string{"first_name", "last_name"},
			expected: "first_name || last_name",
		},
		{
			name:     "sqlite",
			dialect:  &Sqlite{},
			items:    []string{"first_name", "last_name"},
			expected: "first_name || last_name",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.dialect.Concat(tc.items...)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDialect_IfNull(t *testing.T) {
	testCases := []struct {
		name       string
		dialect    Dialect
		expr       string
		defaultVal string
		expected   string
	}{
		{
			name:       "mysql",
			dialect:    &Mysql{},
			expr:       "user_name",
			defaultVal: "'Unknown'",
			expected:   "IFNULL(user_name, 'Unknown')",
		},
		{
			name:       "postgresql",
			dialect:    &Postgresql{},
			expr:       "user_name",
			defaultVal: "'Unknown'",
			expected:   "COALESCE(user_name, 'Unknown')",
		},
		{
			name:       "sqlite",
			dialect:    &Sqlite{},
			expr:       "user_name",
			defaultVal: "'Unknown'",
			expected:   "IFNULL(user_name, 'Unknown')",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.dialect.IfNull(tc.expr, tc.defaultVal)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDialect_RealQueries(t *testing.T) {
	// Test MySQL dialect with a real query
	t.Run("mysql_query", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db, err := Open(mockDB, "mysql")
		require.NoError(t, err)

		// Setup expectations
		mock.ExpectQuery("SELECT `id`, `name` FROM `test_model` WHERE `id` = ?").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Test"))

		// Execute query
		selector := RegisterSelector[TestModel](db).
			Select(Col("ID"), Col("Name")).
			Where(Col("ID").Eq(1))

		result, err := selector.Get(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 1, result.ID)
		assert.Equal(t, "Test", result.Name)
	})

	// Test PostgreSQL dialect with a real query
	t.Run("postgresql_query", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db, err := Open(mockDB, "postgresql")
		require.NoError(t, err)

		// Setup expectations for PostgreSQL style
		mock.ExpectQuery(`SELECT "id", "name" FROM "test_model" WHERE "id" = \$1`).
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Test"))

		// Execute query
		selector := RegisterSelector[TestModel](db).
			Select(Col("ID"), Col("Name")).
			Where(Col("ID").Eq(1))

		result, err := selector.Get(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 1, result.ID)
		assert.Equal(t, "Test", result.Name)
	})

	// Test SQLite dialect with a real query
	t.Run("sqlite_query", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db, err := Open(mockDB, "sqlite")
		require.NoError(t, err)

		// Setup expectations for SQLite style
		mock.ExpectQuery(`SELECT "id", "name" FROM "test_model" WHERE "id" = ?`).
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Test"))

		// Execute query
		selector := RegisterSelector[TestModel](db).
			Select(Col("ID"), Col("Name")).
			Where(Col("ID").Eq(1))

		result, err := selector.Get(context.Background())
		require.NoError(t, err)
		assert.Equal(t, 1, result.ID)
		assert.Equal(t, "Test", result.Name)
	})
}

func TestDialect_Insert(t *testing.T) {
	// Test PostgreSQL placeholder incrementing
	t.Run("postgresql_insert", func(t *testing.T) {
		mockDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db, err := Open(mockDB, "postgresql")
		require.NoError(t, err)

		testModel := TestModel{
			ID:   1,
			Name: "Test",
			Job:  sql.NullString{String: "Engineer", Valid: true},
		}

		// Setup expectations for PostgreSQL style placeholders
		mock.ExpectExec(`INSERT INTO "test_model" \("id", "name", "job"\) VALUES \(\$1, \$2, \$3\)`).
			WithArgs(1, "Test", sql.NullString{String: "Engineer", Valid: true}).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Execute query
		inserter := RegisterInserter[TestModel](db).Insert(nil, &testModel)
		_, err = inserter.Exec(context.Background())
		require.NoError(t, err)
	})
}

func TestDialect_DateFunction(t *testing.T) {
	// Using type assertion to test dialect-specific functions

	mysql := &Mysql{}
	pgSQL := &Postgresql{}
	sqlite := &Sqlite{}

	// Test DateFormat function
	if mysqlWithDate, ok := interface{}(mysql).(interface{ DateFormat(string, string) string }); ok {
		result := mysqlWithDate.DateFormat("created_at", "%Y-%m-%d")
		assert.Equal(t, "DATE_FORMAT(created_at, '%Y-%m-%d')", result)
	}

	if pgWithDate, ok := interface{}(pgSQL).(interface{ DateFormat(string, string) string }); ok {
		result := pgWithDate.DateFormat("created_at", "YYYY-MM-DD")
		assert.Equal(t, "TO_CHAR(created_at, 'YYYY-MM-DD')", result)
	}

	if sqliteWithDate, ok := interface{}(sqlite).(interface{ DateFormat(string, string) string }); ok {
		result := sqliteWithDate.DateFormat("created_at", "%Y-%m-%d")
		assert.Equal(t, "strftime('%Y-%m-%d', created_at)", result)
	}

	// Test SQLite specific JulianDay function
	if sqliteWithJulian, ok := interface{}(sqlite).(interface{ JulianDay(string) string }); ok {
		result := sqliteWithJulian.JulianDay("created_at")
		assert.Equal(t, "julianday(created_at)", result)
	}
}
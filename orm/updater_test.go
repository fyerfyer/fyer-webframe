package orm

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdater_Build(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)

	testCases := []struct {
		name      string
		q         QueryBuilder
		wantQuery *Query
		wantErr   error
	}{
		{
			name: "simple update",
			q: RegisterUpdater[TestModel](db).Update().
				Set(Col("Name"), "Tom").
				Where(Col("ID").Eq(12)),
			wantQuery: &Query{
				SQL:  "UPDATE `test_model` SET `name` = ? WHERE `id` = ?;",
				Args: []any{"Tom", 12},
			},
		},
		{
			name: "update multiple columns",
			q: RegisterUpdater[TestModel](db).Update().
				Set(Col("Name"), "Jerry").
				Set(Col("Job"), sql.NullString{String: "Engineer", Valid: true}).
				Where(Col("ID").Eq(12)),
			wantQuery: &Query{
				SQL:  "UPDATE `test_model` SET `name` = ?, `job` = ? WHERE `id` = ?;",
				Args: []any{"Jerry", sql.NullString{String: "Engineer", Valid: true}, 12},
			},
		},
		{
			name: "update with SetMulti",
			q: RegisterUpdater[TestModel](db).Update().
				SetMulti(map[string]any{
					"Name": "Bob",
					"Job":  sql.NullString{String: "Doctor", Valid: true},
				}).
				Where(Col("ID").Eq(12)),
			wantQuery: &Query{
				SQL:  "UPDATE `test_model` SET `name` = ?, `job` = ? WHERE `id` = ?;",
				Args: []any{"Bob", sql.NullString{String: "Doctor", Valid: true}, 12},
			},
		},
		{
			name: "update with multiple conditions",
			q: RegisterUpdater[TestModel](db).Update().
				Set(Col("Name"), "Alice").
				Where(Col("ID").Gt(10), Col("Name").Eq("Tom")),
			wantQuery: &Query{
				SQL:  "UPDATE `test_model` SET `name` = ? WHERE `id` > ? AND `name` = ?;",
				Args: []any{"Alice", 10, "Tom"},
			},
		},
		{
			name: "update with limit",
			q: RegisterUpdater[TestModel](db).Update().
				Set(Col("Name"), "Jim").
				Where(Col("ID").Gt(10)).
				Limit(5),
			wantQuery: &Query{
				SQL:  "UPDATE `test_model` SET `name` = ? WHERE `id` > ? LIMIT 5;",
				Args: []any{"Jim", 10},
			},
		},
		{
			name: "update with expression value",
			q: RegisterUpdater[TestModel](db).Update().
				Set(Col("Name"), Raw("CONCAT(name, '_updated')")).
				Where(Col("ID").Eq(12)),
			wantQuery: &Query{
				SQL:  "UPDATE `test_model` SET `name` = CONCAT(name, '_updated') WHERE `id` = ?;",
				Args: []any{12},
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

func TestUpdater_Exec(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)

	testCases := []struct {
		name      string
		updater   *Updater[TestModel]
		affected  int64
		expectErr bool
		setup     func()
	}{
		{
			name: "update success",
			updater: RegisterUpdater[TestModel](db).Update().
				Set(Col("Name"), "Tom").
				Where(Col("ID").Eq(12)),
			affected: 1,
			setup: func() {
				mock.ExpectExec("UPDATE `test_model` SET `name` = \\? WHERE `id` = \\?").
					WithArgs("Tom", 12).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name: "update multiple rows",
			updater: RegisterUpdater[TestModel](db).Update().
				Set(Col("Name"), "Jerry").
				Where(Col("ID").Gt(10)),
			affected: 5,
			setup: func() {
				mock.ExpectExec("UPDATE `test_model` SET `name` = \\? WHERE `id` > \\?").
					WithArgs("Jerry", 10).
					WillReturnResult(sqlmock.NewResult(0, 5))
			},
		},
		{
			name: "update with error",
			updater: RegisterUpdater[TestModel](db).Update().
				Set(Col("Name"), "Bob").
				Where(Col("ID").Eq(999)),
			expectErr: true,
			setup: func() {
				mock.ExpectExec("UPDATE `test_model` SET `name` = \\? WHERE `id` = \\?").
					WithArgs("Bob", 999).
					WillReturnError(sql.ErrNoRows)
			},
		},
		{
			name: "update with limit",
			updater: RegisterUpdater[TestModel](db).Update().
				Set(Col("Name"), "Alice").
				Where(Col("ID").Gt(5)).
				Limit(3),
			affected: 3,
			setup: func() {
				mock.ExpectExec("UPDATE `test_model` SET `name` = \\? WHERE `id` > \\? LIMIT 3").
					WithArgs("Alice", 5).
					WillReturnResult(sqlmock.NewResult(0, 3))
			},
		},
		{
			name: "update with SetMulti",
			updater: RegisterUpdater[TestModel](db).Update().
				SetMulti(map[string]any{
					"Name": "Mark",
					"Job":  sql.NullString{String: "Engineer", Valid: true},
				}).
				Where(Col("ID").Eq(42)),
			affected: 1,
			setup: func() {
				// Since map iteration order is non-deterministic, we need to use a pattern that doesn't
				// depend on the order of the SET clauses
				mock.ExpectExec("UPDATE `test_model` SET .* WHERE `id` = \\?").
					WithArgs("Mark", sql.NullString{String: "Engineer", Valid: true}, 42).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			result, err := tc.updater.Exec(context.Background())
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			affected, err := result.RowsAffected()
			require.NoError(t, err)
			assert.Equal(t, tc.affected, affected)
		})
	}

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdaterTableNameInterface(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)

	updater := RegisterUpdater[TestModelWithTableNameInterface](db).
		Update().
		Set(Col("ID"), 42).
		Where(Col("ID").Eq(1))

	mock.ExpectExec("UPDATE `test_model` SET `id` = \\? WHERE `id` = \\?").
		WithArgs(42, 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	result, err := updater.Exec(context.Background())
	require.NoError(t, err)

	affected, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdater_EmptyBuild(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)

	updater := RegisterUpdater[TestModel](db).
		Update().
		Where(Col("ID").Eq(1))

	assert.Panics(t, func() {
		updater.Build()
	})
}

func TestUpdater_WithTag(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)

	updater := RegisterUpdater[TestModelWithTag](db).
		Update().
		Set(Col("ID"), 100).
		Set(Col("Name"), "TaggedUpdate").
		Where(Col("ID").Eq(1))

	query, err := updater.Build()
	require.NoError(t, err)

	assert.Equal(t, "UPDATE `test_model_with_tag` SET `testid` = ?, `testname` = ? WHERE `testid` = ?;", query.SQL)
	assert.Equal(t, []any{100, "TaggedUpdate", 1}, query.Args)
}
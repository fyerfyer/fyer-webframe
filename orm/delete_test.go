package orm

import (
	"context"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDeleter_Build(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		q         *Deleter[TestModel]
		wantQuery *Query
		wantErr   error
	}{
		{
			name: "simple delete",
			q:    RegisterDeleter[TestModel](db).Delete(),
			wantQuery: &Query{
				SQL:  "DELETE FROM `test_model`;",
				Args: nil,
			},
		},
		{
			name: "with where",
			q: RegisterDeleter[TestModel](db).Delete().
				Where(Col("ID").Eq(12)),
			wantQuery: &Query{
				SQL:  "DELETE FROM `test_model` WHERE `id` = ?;",
				Args: []any{12},
			},
		},
		{
			name: "with where multiple conditions",
			q: RegisterDeleter[TestModel](db).Delete().
				Where(Col("ID").Eq(12), Col("Name").Eq("Tom")),
			wantQuery: &Query{
				SQL:  "DELETE FROM `test_model` WHERE `id` = ? AND `name` = ?;",
				Args: []any{12, "Tom"},
			},
		},
		{
			name: "with limit",
			q: RegisterDeleter[TestModel](db).Delete().
				Where(Col("ID").Gt(12)).
				Limit(10),
			wantQuery: &Query{
				SQL:  "DELETE FROM `test_model` WHERE `id` > ? LIMIT 10;",
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

func TestDeleter_Exec(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB)
	require.NoError(t, err)

	testCases := []struct {
		name         string
		q           *Deleter[TestModel]
		query       string
		wantErr     error
		affected    int64
	}{
		{
			name: "delete success",
			q: RegisterDeleter[TestModel](db).Delete().
				Where(Col("ID").Eq(12)),
			query: "DELETE FROM `test_model` WHERE `id` = ?",
			affected: 1,
		},
		{
			name: "delete with limit",
			q: RegisterDeleter[TestModel](db).Delete().
				Where(Col("Age").Gt(18)).
				Limit(10),
			query: "DELETE FROM `test_model` WHERE `age` > \\? LIMIT 10",
			affected: 10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock.ExpectExec(tc.query).
				WillReturnResult(sqlmock.NewResult(0, tc.affected))

			res, err := tc.q.Exec(context.Background())
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}

			affected, err := res.RowsAffected()
			require.NoError(t, err)
			assert.Equal(t, tc.affected, affected)
		})
	}
}

func TestDeleterWithTag(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB)
	require.NoError(t, err)

	testCases := []struct {
		name      string
		q         *Deleter[TestModelWithTag]
		wantQuery *Query
		wantErr   error
	}{
		{
			name: "delete with tag",
			q: RegisterDeleter[TestModelWithTag](db).Delete().
				Where(Col("ID").Eq(12)),
			wantQuery: &Query{
				SQL:  "DELETE FROM `test_model_with_tag` WHERE `testid` = ?;",
				Args: []any{12},
			},
		},
		{
			name: "delete with multiple tag columns",
			q: RegisterDeleter[TestModelWithTag](db).Delete().
				Where(Col("ID").Eq(12), Col("Name").Eq("Tom")),
			wantQuery: &Query{
				SQL:  "DELETE FROM `test_model_with_tag` WHERE `testid` = ? AND `testname` = ?;",
				Args: []any{12, "Tom"},
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
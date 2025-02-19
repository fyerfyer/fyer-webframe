package orm

import (
	"database/sql"
	"github.com/stretchr/testify/assert"
	"testing"
)

type TestModel struct {
	ID   int
	Name string
	Job  sql.NullString
}

func TestSelector_Build(t *testing.T) {
	db, err := NewDB()
	assert.NoError(t, err)

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
				Args: []any{Value{12}},
			},
		},
		{
			name: "with complex where",
			q: RegisterSelector[TestModel](db).Select().
				Where(NOT(Col("ID").Eq(12))),
			wantQuery: &Query{
				SQL:  "SELECT * FROM `test_model` WHERE NOT `id` = ?;",
				Args: []any{Value{12}},
			},
		},
		{
			name: "with multiple where",
			q: RegisterSelector[TestModel](db).Select("ID").
				Where(Col("ID").Eq(12), Col("Job").IsNull()),
			wantQuery: &Query{
				SQL:  "SELECT `id` FROM `test_model` WHERE `id` = ? AND `job` IS NULL;",
				Args: []any{Value{12}},
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

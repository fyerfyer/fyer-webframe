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
	testCases := []struct {
		name      string
		q         *Selector[TestModel]
		wantQuery *Query
		wantErr   error
	}{
		{
			name: "simple from",
			q:    NewSelector[TestModel]().Select(),
			wantQuery: &Query{
				SQL:  "SELECT * FROM `test_model`;",
				Args: nil,
			},
		},
		{
			// 指定列
			name: "with columns",
			q:    NewSelector[TestModel]().Select("id", "name"),
			wantQuery: &Query{
				SQL:  "SELECT `id`, `name` FROM `test_model`;",
				Args: nil,
			},
		},
		{
			name: "with simple where",
			q: NewSelector[TestModel]().Select().
				Where(Col("id").Eq(12)),
			wantQuery: &Query{
				SQL:  "SELECT * FROM `test_model` WHERE `id` = ?;",
				Args: []any{Value{12}},
			},
		},
		{
			name: "with complex where",
			q: NewSelector[TestModel]().Select().
				Where(NOT(Col("id").Eq(12))),
			wantQuery: &Query{
				SQL:  "SELECT * FROM `test_model` WHERE NOT `id` = ?;",
				Args: []any{Value{12}},
			},
		},
		{
			name: "with multiple where",
			q: NewSelector[TestModel]().Select("id").
				Where(Col("id").Eq(12), Col("job").IsNull()),
			wantQuery: &Query{
				SQL:  "SELECT `id` FROM `test_model` WHERE `id` = ? AND `job` IS NULL;",
				Args: []any{Value{12}},
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

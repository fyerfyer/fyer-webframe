package orm

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeleter_Build(t *testing.T) {
	testCases := []struct {
		name      string
		q         *Deleter[TestModel]
		wantQuery *Query
		wantErr   error
	}{
		{
			name: "simple from",
			q:    NewDeleter[TestModel]().Delete(),
			wantQuery: &Query{
				SQL:  "DELETE * FROM `test_model`;",
				Args: nil,
			},
		},
		{
			// 指定列
			name: "with columns",
			q:    NewDeleter[TestModel]().Delete("id", "name"),
			wantQuery: &Query{
				SQL:  "DELETE `id`, `name` FROM `test_model`;",
				Args: nil,
			},
		},
		{
			name: "with simple where",
			q: NewDeleter[TestModel]().Delete().
				Where(Col("id").Eq(12)),
			wantQuery: &Query{
				SQL:  "DELETE * FROM `test_model` WHERE `id` = ?;",
				Args: []any{Value{12}},
			},
		},
		{
			name: "with complex where",
			q: NewDeleter[TestModel]().Delete().
				Where(NOT(Col("id").Eq(12))),
			wantQuery: &Query{
				SQL:  "DELETE * FROM `test_model` WHERE NOT `id` = ?;",
				Args: []any{Value{12}},
			},
		},
		{
			name: "with multiple where",
			q: NewDeleter[TestModel]().Delete("id").
				Where(Col("id").Eq(12), Col("job").IsNull()),
			wantQuery: &Query{
				SQL:  "DELETE `id` FROM `test_model` WHERE `id` = ? AND `job` IS NULL;",
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

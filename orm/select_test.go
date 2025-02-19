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
	db, err := NewDB()
	assert.NoError(t, err)

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
	db, err := NewDB()
	assert.NoError(t, err)

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

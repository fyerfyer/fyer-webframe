package orm

import (
	"database/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInserter_Build(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)
	testModel := TestModel{
		ID:   1,
		Name: "Tom",
		Job: sql.NullString{
			String: "Engineer",
			Valid:  true,
		},
	}
	testModel2 := TestModel{
		ID:   2,
		Name: "Jerry",
		Job: sql.NullString{
			String: "Doctor",
			Valid:  true,
		},
	}

	testCases := []struct {
		name      string
		q         *Inserter[TestModel]
		wantQuery *Query
		wantErr   error
	}{
		{
			name: "simple insert",
			q:    RegisterInserter[TestModel](db).Insert(nil, &testModel),
			wantQuery: &Query{
				SQL:  "INSERT INTO `test_model` (`id`, `name`, `job`) VALUES (?, ?, ?);",
				Args: []any{1, "Tom", sql.NullString{String: "Engineer", Valid: true}},
			},
		},
		{
			name: "multiple insert",
			q:    RegisterInserter[TestModel](db).Insert(nil, &testModel, &testModel2),
			wantQuery: &Query{
				SQL: "INSERT INTO `test_model` (`id`, `name`, `job`) VALUES (?, ?, ?), (?, ?, ?);",
				Args: []any{1, "Tom", sql.NullString{String: "Engineer", Valid: true},
					2, "Jerry", sql.NullString{String: "Doctor", Valid: true}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			q, err := tc.q.Build()
			require.NoError(t, err)
			require.Equal(t, tc.wantQuery, q)
		})
	}
}

func TestInserterTableNameInterface(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)
	testModel := TestModelWithTableNameInterface{
		ID: 1,
	}

	testCases := []struct {
		name      string
		q         *Inserter[TestModelWithTableNameInterface]
		wantQuery *Query
		wantErr   error
	}{
		{
			name: "simple delete",
			q: RegisterInserter[TestModelWithTableNameInterface](db).
				Insert(nil, &testModel),
			wantQuery: &Query{
				SQL:  "INSERT INTO `test_model` (`id`) VALUES (?);",
				Args: []any{1},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			q, err := tc.q.Build()
			require.NoError(t, err)
			require.Equal(t, tc.wantQuery, q)
		})
	}
}

func TestInserter_PartialColumns(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db, err := Open(mockDB, "mysql")
	require.NoError(t, err)

	testModel := &TestModel{
		ID:   1,
		Name: "Tom",
		Job:  sql.NullString{String: "Engineer", Valid: true},
	}

	testCases := []struct {
		name      string
		q         *Inserter[TestModel]
		wantQuery *Query
		wantErr   error
	}{
		{
			name: "specified columns",
			q: RegisterInserter[TestModel](db).
				Insert([]string{"ID", "Name"}, testModel),
			wantQuery: &Query{
				SQL:  "INSERT INTO `test_model` (`id`, `name`) VALUES (?, ?);",
				Args: []any{1, "Tom"},
			},
		},
		{
			name: "all columns",
			q: RegisterInserter[TestModel](db).
				Insert(nil, testModel),
			wantQuery: &Query{
				SQL:  "INSERT INTO `test_model` (`id`, `name`, `job`) VALUES (?, ?, ?);",
				Args: []any{1, "Tom", sql.NullString{String: "Engineer", Valid: true}},
			},
		},
		{
			name: "multiple rows with specified columns",
			q: RegisterInserter[TestModel](db).
				Insert([]string{"Name", "Job"},
					&TestModel{Name: "Tom", Job: sql.NullString{String: "Engineer", Valid: true}},
					&TestModel{Name: "Jerry", Job: sql.NullString{String: "Teacher", Valid: true}},
				),
			wantQuery: &Query{
				SQL: "INSERT INTO `test_model` (`name`, `job`) VALUES (?, ?), (?, ?);",
				Args: []any{"Tom", sql.NullString{String: "Engineer", Valid: true},
					"Jerry", sql.NullString{String: "Teacher", Valid: true}},
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

func TestInserter_Upsert(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	mysqlDB, err := Open(mockDB, "mysql")
	require.NoError(t, err)
	sqliteDB, err := Open(mockDB, "sqlite")
	require.NoError(t, err)

	testModel := TestModel{
		ID:   1,
		Name: "Tom",
		Job: sql.NullString{
			String: "Engineer",
			Valid:  true,
		},
	}

	testCases := []struct {
		name      string
		q         *Inserter[TestModel]
		wantQuery *Query
		wantErr   error
	}{
		{
			name: "mysql upsert",
			q: RegisterInserter[TestModel](mysqlDB).
				Insert(nil, &testModel).
				Upsert(nil, []*Column{Col("ID"), Col("Name")}),
			wantQuery: &Query{
				SQL:  "INSERT INTO `test_model` (`id`, `name`, `job`) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE `id` = VALUES(id), `name` = VALUES(name);",
				Args: []any{1, "Tom", sql.NullString{String: "Engineer", Valid: true}},
			},
		},
		{
			name: "sqlite upsert",
			q: RegisterInserter[TestModel](sqliteDB).
				Insert(nil, &testModel).
				Upsert([]*Column{Col("ID"), Col("Name")}, []*Column{Col("ID"), Col("Name")}),
			wantQuery: &Query{
				SQL:  "INSERT INTO `test_model` (`id`, `name`, `job`) VALUES (?, ?, ?) ON CONFLICT(`id`, `name`) DO UPDATE SET id = EXCLUDED.id, name = EXCLUDED.name;",
				Args: []any{1, "Tom", sql.NullString{String: "Engineer", Valid: true}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			q, err := tc.q.Build()
			require.NoError(t, err)
			require.Equal(t, tc.wantQuery, q)
		})
	}
}

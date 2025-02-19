package orm

import "testing"

func TestMeta(t *testing.T) {
	type User struct {
		ID   int
		Name string
		Age  int
	}

	m, err := parseModel(User{})
	if err != nil {
		t.Fatal(err)
	}

	if m.table != "user" {
		t.Fatalf("expected table name is User, but got %s", m.table)
	}

	if len(m.fieldsMap) != 3 {
		t.Fatalf("expected 3 fields, but got %d", len(m.fieldsMap))
	}

	for _, f := range []string{"ID", "Name", "Age"} {
		if _, ok := m.fieldsMap[f]; !ok {
			t.Fatalf("expected field %s, but not found", f)
		}
	}
}

package orm

import (
	"context"
	"strconv"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func prepareQueryTestData(b *testing.B, count int) {
	ctx := context.Background()

	_, err := benchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatal(err)
	}

	for i := 1; i <= count; i++ {
		user := &BenchmarkUser{
			ID:        i, // Explicitly set the ID to ensure IDs start from 1
			Name:      "user_" + strconv.Itoa(i),
			Email:     "user" + strconv.Itoa(i) + "@example.com",
			Age:       20 + (i % 50),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    i%2 == 0,
		}

		inserter := RegisterInserter[BenchmarkUser](benchDB)

		_, err := inserter.Insert([]string{"ID", "Name", "Email", "Age", "CreatedAt", "UpdatedAt", "Active"}, user).Exec(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func cleanupQueryTestData(b *testing.B) {
	ctx := context.Background()
	_, err := benchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatal(err)
	}
}

func BenchmarkQueryById(b *testing.B) {
	prepareQueryTestData(b, 100)
	defer cleanupQueryTestData(b)

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		selector := RegisterSelector[BenchmarkUser](benchDB)

		_, err := selector.Select().Where(Col("ID").Eq(1)).Get(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQueryWithCondition(b *testing.B) {
	prepareQueryTestData(b, 100)
	defer cleanupQueryTestData(b)

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		selector := RegisterSelector[BenchmarkUser](benchDB)

		_, err := selector.Select().Where(Col("Age").Gt(30), Col("Active").Eq(true)).GetMulti(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQueryMultiple(b *testing.B) {
	prepareQueryTestData(b, 100)
	defer cleanupQueryTestData(b)

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		selector := RegisterSelector[BenchmarkUser](benchDB)
		_, err := selector.Select().GetMulti(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQueryWithOrderBy(b *testing.B) {
	prepareQueryTestData(b, 100)
	defer cleanupQueryTestData(b)

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		selector := RegisterSelector[BenchmarkUser](benchDB)
		_, err := selector.Select().OrderBy(Desc(Col("Age"))).GetMulti(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQueryWithLimitOffset(b *testing.B) {
	prepareQueryTestData(b, 100)
	defer cleanupQueryTestData(b)

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		selector := RegisterSelector[BenchmarkUser](benchDB)
		_, err := selector.Select().Limit(10).Offset(i % 10 * 10).GetMulti(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQueryWithGroupBy(b *testing.B) {
	prepareQueryTestData(b, 100)
	defer cleanupQueryTestData(b)

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		selector := RegisterSelector[BenchmarkUser](benchDB)
		_, err := selector.Select(Col("Age"), Count("ID").As("UserCount")).
			GroupBy(Col("Age")).
			GetMulti(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQueryWithAggregation(b *testing.B) {
	prepareQueryTestData(b, 100)
	defer cleanupQueryTestData(b)

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		selector := RegisterSelector[BenchmarkUser](benchDB)

		_, err := selector.Select(Max("Age"), Min("Age"), Avg("Age")).
			Get(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQueryWithComplexCondition(b *testing.B) {
	prepareQueryTestData(b, 100)
	defer cleanupQueryTestData(b)

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		selector := RegisterSelector[BenchmarkUser](benchDB)

		_, err := selector.Select().
			Where(
				Col("Age").Gt(30),
				Col("Active").Eq(true),
				Col("Name").Like("user_%"),
			).
			OrderBy(Desc(Col("Age"))).
			Limit(5).
			GetMulti(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
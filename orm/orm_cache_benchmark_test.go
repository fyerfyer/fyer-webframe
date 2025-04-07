package orm

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func prepareBenchmarkData(b *testing.B) {
	ctx := context.Background()

	_, err := benchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < 100; i++ {
		user := &BenchmarkUser{
			Name:      fmt.Sprintf("User%d", i),
			Email:     fmt.Sprintf("user%d@example.com", i),
			Age:       20 + i%30,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    i%2 == 0,
		}

		inserter := RegisterInserter[BenchmarkUser](benchDB).Insert(nil, user)
		_, err := inserter.Exec(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func setupBenchmarkCache(b *testing.B) *MemoryCache {
	memCache := NewMemoryCache()
	err := WithDBMiddlewareCache(memCache)(benchDB)
	if err != nil {
		b.Fatal(err)
	}

	benchDB.SetModelCacheConfig("benchmark_user", &ModelCacheConfig{
		Enabled: true,
		TTL:     time.Minute,
		Tags:    []string{"benchmark_user"},
	})

	return memCache
}

func BenchmarkSelectNoCache(b *testing.B) {
	prepareBenchmarkData(b)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		selector := RegisterSelector[BenchmarkUser](benchDB).Select().Where(Col("ID").Gt(0))
		_, err := selector.GetMulti(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSelectWithCache(b *testing.B) {
	prepareBenchmarkData(b)
	ctx := context.Background()

	memCache := setupBenchmarkCache(b)
	defer memCache.Clear(ctx)

	selector := RegisterSelector[BenchmarkUser](benchDB).Select().Where(Col("ID").Gt(0)).WithCache()
	_, err := selector.GetMulti(ctx)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		selector := RegisterSelector[BenchmarkUser](benchDB).Select().Where(Col("ID").Gt(0)).WithCache()
		_, err := selector.GetMulti(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCacheInvalidation(b *testing.B) {
	prepareBenchmarkData(b)
	ctx := context.Background()

	memCache := setupBenchmarkCache(b)
	defer memCache.Clear(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		selector := RegisterSelector[BenchmarkUser](benchDB).Select().Where(Col("ID").Gt(0)).WithCache()
		_, err := selector.GetMulti(ctx)
		if err != nil {
			b.Fatal(err)
		}

		updater := RegisterUpdater[BenchmarkUser](benchDB).Update().
			Set(Col("Name"), "Updated").
			Where(Col("ID").Eq(1)).
			WithInvalidateCache().
			WithInvalidateTags("benchmark_user")
		_, err = updater.Exec(ctx)
		if err != nil {
			b.Fatal(err)
		}

		selector = RegisterSelector[BenchmarkUser](benchDB).Select().Where(Col("ID").Gt(0)).WithCache()
		_, err = selector.GetMulti(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCacheTTL(b *testing.B) {
	prepareBenchmarkData(b)
	ctx := context.Background()

	memCache := setupBenchmarkCache(b)
	defer memCache.Clear(ctx)

	shortTTL := 10 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		selector := RegisterSelector[BenchmarkUser](benchDB).Select().
			Where(Col("ID").Gt(0)).
			WithCache().
			WithSelectorCacheTTL(shortTTL)
		_, err := selector.GetMulti(ctx)
		if err != nil {
			b.Fatal(err)
		}

		time.Sleep(shortTTL + 5*time.Millisecond)

		selector = RegisterSelector[BenchmarkUser](benchDB).Select().
			Where(Col("ID").Gt(0)).
			WithCache().
			WithSelectorCacheTTL(shortTTL)
		_, err = selector.GetMulti(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCacheWithTags(b *testing.B) {
	prepareBenchmarkData(b)
	ctx := context.Background()

	memCache := setupBenchmarkCache(b)
	defer memCache.Clear(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tags := []string{"tag1", "tag2"}
		selector := RegisterSelector[BenchmarkUser](benchDB).Select().
			Where(Col("ID").Gt(0)).
			WithCache().
			WithCacheTags(tags...)
		_, err := selector.GetMulti(ctx)
		if err != nil {
			b.Fatal(err)
		}

		err = benchDB.InvalidateCache(ctx, "benchmark_user", "tag1")
		if err != nil {
			b.Fatal(err)
		}

		selector = RegisterSelector[BenchmarkUser](benchDB).Select().
			Where(Col("ID").Gt(0)).
			WithCache().
			WithCacheTags(tags...)
		_, err = selector.GetMulti(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLargeDatasetCache(b *testing.B) {
	ctx := context.Background()

	_, err := benchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < 1000; i++ {
		user := &BenchmarkUser{
			Name:      fmt.Sprintf("User%d", i),
			Email:     fmt.Sprintf("user%d@example.com", i),
			Age:       20 + i%30,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    i%2 == 0,
		}

		inserter := RegisterInserter[BenchmarkUser](benchDB).Insert(nil, user)
		_, err := inserter.Exec(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}

	memCache := setupBenchmarkCache(b)
	defer memCache.Clear(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		selector := RegisterSelector[BenchmarkUser](benchDB).Select().WithCache()
		_, err := selector.GetMulti(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
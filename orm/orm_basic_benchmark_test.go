package orm

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type BenchmarkUser struct {
	ID        int       `orm:"primary_key;auto_increment"`
	Name      string    `orm:"size:255"`
	Email     string    `orm:"size:255;unique"`
	Age       int
	CreatedAt time.Time
	UpdatedAt time.Time
	Active    bool
}

var benchDB *DB

func TestMain(m *testing.M) {
	DisableCacheDebugLog()

	setupBenchmarkDB()
	defer cleanupBenchmarkDB()
	m.Run()
}

func setupBenchmarkDB() {
	var err error
	sqlDB, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/")
	if err != nil {
		panic(err)
	}

	_, err = sqlDB.Exec("CREATE DATABASE IF NOT EXISTS orm_benchmark")
	if err != nil {
		panic(err)
	}

	_, err = sqlDB.Exec("USE orm_benchmark")
	if err != nil {
		panic(err)
	}

	_, err = sqlDB.Exec("DROP TABLE IF EXISTS benchmark_user")
	if err != nil {
		panic(err)
	}

	sqlDB.Close()

	sqlDB, err = sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/orm_benchmark?parseTime=true")
	if err != nil {
		panic(err)
	}

	benchDB, err = Open(sqlDB, "mysql")
	if err != nil {
		panic(err)
	}

	err = benchDB.MigrateModel(context.Background(), &BenchmarkUser{},
		WithMigrationLog(false),
		WithStrategy(ForceRecreate))
	if err != nil {
		panic(err)
	}
}

func cleanupBenchmarkDB() {
	if benchDB != nil {
		_, err := benchDB.execContext(context.Background(), "DROP TABLE IF EXISTS benchmark_user")
		if err != nil {
			panic(err)
		}
		_, err = benchDB.execContext(context.Background(), "DROP DATABASE IF EXISTS orm_benchmark")
		if err != nil {
			panic(err)
		}
		benchDB.Close()
	}
}

func BenchmarkInsert(b *testing.B) {
	ctx := context.Background()

	_, err := benchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatalf("Failed to clean before insert: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		user := &BenchmarkUser{
			Name:      fmt.Sprintf("User %d", i),
			Email:     fmt.Sprintf("user%d_%d@example.com", i, time.Now().UnixNano()), // Make email unique
			Age:       25 + (i % 40),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    true,
		}

		inserter := RegisterInserter[BenchmarkUser](benchDB)
		_, err := inserter.Insert(nil, user).Exec(ctx)
		if err != nil {
			b.Fatalf("Insert failed: %v", err)
		}
	}
}

func BenchmarkBatchInsert(b *testing.B) {
	ctx := context.Background()
	batchSize := 100
	batches := b.N / batchSize
	if batches == 0 {
		batches = 1
	}

	_, err := benchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatalf("Failed to clean before batch insert: %v", err)
	}

	b.ResetTimer()
	for batch := 0; batch < batches; batch++ {
		users := make([]*BenchmarkUser, batchSize)
		for i := 0; i < batchSize; i++ {
			idx := batch*batchSize + i
			users[i] = &BenchmarkUser{
				Name:      fmt.Sprintf("BatchUser %d", idx),
				Email:     fmt.Sprintf("batch%d_%d@example.com", idx, time.Now().UnixNano()), // Make email unique
				Age:       25 + (idx % 40),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Active:    true,
			}
		}

		for _, user := range users {
			inserter := RegisterInserter[BenchmarkUser](benchDB)
			_, err := inserter.Insert(nil, user).Exec(ctx)
			if err != nil {
				b.Fatalf("Batch insert failed: %v", err)
			}
		}
	}
}

func BenchmarkSelect(b *testing.B) {
	ctx := context.Background()

	prepareSelectData(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		selector := RegisterSelector[BenchmarkUser](benchDB)
		_, err := selector.Select().Where(Col("ID").Gt(0)).Limit(10).GetMulti(ctx)
		if err != nil {
			b.Fatalf("Select failed: %v", err)
		}
	}
}

func prepareSelectData(b *testing.B) {
	ctx := context.Background()

	_, err := benchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatalf("Failed to clean before select: %v", err)
	}

	for i := 0; i < 100; i++ {
		user := &BenchmarkUser{
			Name:      fmt.Sprintf("SelectUser %d", i),
			Email:     fmt.Sprintf("select%d@example.com", i),
			Age:       25 + (i % 40),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    true,
		}

		inserter := RegisterInserter[BenchmarkUser](benchDB)
		_, err := inserter.Insert(nil, user).Exec(ctx)
		if err != nil {
			b.Fatalf("Failed to prepare select data: %v", err)
		}
	}
}

func BenchmarkSelectMulti(b *testing.B) {
	ctx := context.Background()

	prepareSelectData(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		selector := RegisterSelector[BenchmarkUser](benchDB)
		_, err := selector.Select().Where(Col("Age").Gt(30)).GetMulti(ctx)
		if err != nil {
			b.Fatalf("SelectMulti failed: %v", err)
		}
	}
}

func BenchmarkUpdate(b *testing.B) {
	ctx := context.Background()

	prepareUpdateData(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := (i % 100) + 1
		updater := RegisterUpdater[BenchmarkUser](benchDB)
		_, err := updater.Update().
			Set(Col("Name"), fmt.Sprintf("Updated %d", i)).
			Set(Col("UpdatedAt"), time.Now()).
			Where(Col("ID").Eq(id)).
			Exec(ctx)
		if err != nil {
			b.Fatalf("Update failed: %v", err)
		}
	}
}

func prepareUpdateData(b *testing.B) {
	ctx := context.Background()

	_, err := benchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatalf("Failed to clean before update: %v", err)
	}

	for i := 0; i < 100; i++ {
		user := &BenchmarkUser{
			Name:      fmt.Sprintf("UpdateUser %d", i),
			Email:     fmt.Sprintf("update%d@example.com", i),
			Age:       25 + (i % 40),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    true,
		}

		inserter := RegisterInserter[BenchmarkUser](benchDB)
		_, err := inserter.Insert(nil, user).Exec(ctx)
		if err != nil {
			b.Fatalf("Failed to prepare update data: %v", err)
		}
	}
}

func BenchmarkDelete(b *testing.B) {
	ctx := context.Background()

	prepareDeleteData(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := (i % 100) + 1
		deleter := RegisterDeleter[BenchmarkUser](benchDB)
		_, err := deleter.Delete().Where(Col("ID").Eq(id)).Exec(ctx)
		if err != nil {
			b.Fatalf("Delete failed: %v", err)
		}
	}
}

func prepareDeleteData(b *testing.B) {
	ctx := context.Background()

	_, err := benchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatalf("Failed to clean before delete: %v", err)
	}

	for i := 0; i < 100; i++ {
		user := &BenchmarkUser{
			Name:      fmt.Sprintf("DeleteUser %d", i),
			Email:     fmt.Sprintf("delete%d@example.com", i),
			Age:       25 + (i % 40),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    true,
		}

		inserter := RegisterInserter[BenchmarkUser](benchDB)
		_, err := inserter.Insert(nil, user).Exec(ctx)
		if err != nil {
			b.Fatalf("Failed to prepare delete data: %v", err)
		}
	}
}

func BenchmarkTransaction(b *testing.B) {
	ctx := context.Background()

	_, err := benchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatalf("Failed to clean before transaction: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := benchDB.Tx(ctx, func(tx *Tx) error {
			user := &BenchmarkUser{
				Name:      fmt.Sprintf("TxUser %d", i),
				Email:     fmt.Sprintf("tx%d_%d@example.com", i, time.Now().UnixNano()), // Make email unique
				Age:       25 + (i % 40),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Active:    true,
			}

			inserter := RegisterInserter[BenchmarkUser](tx)
			result, err := inserter.Insert(nil, user).Exec(ctx)
			if err != nil {
				return err
			}

			id, err := result.LastInsertId()
			if err != nil {
				return err
			}

			// Just to verify the ID was obtained
			if id <= 0 {
				return fmt.Errorf("invalid ID: %d", id)
			}

			return nil
		}, nil)

		if err != nil {
			b.Fatalf("Transaction failed: %v", err)
		}
	}
}

func BenchmarkComplexQuery(b *testing.B) {
	ctx := context.Background()

	prepareSelectData(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		selector := RegisterSelector[BenchmarkUser](benchDB)
		_, err := selector.Select().
			Where(
				Col("Age").Gt(20),
				Col("Name").Like("%User%"),
				NOT(Col("Email").Like("%batch%")),
			).
			OrderBy(Desc(Col("ID"))).
			Limit(20).
			Offset(5).
			GetMulti(ctx)

		if err != nil {
			b.Fatalf("Complex query failed: %v", err)
		}
	}
}
package orm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var concurrentBenchDB *DB

func setupConcurrentBenchDB() {
	var err error
	sqlDB, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/")
	if err != nil {
		panic(err)
	}

	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(20)
	sqlDB.SetConnMaxLifetime(time.Minute * 3)
	sqlDB.SetConnMaxIdleTime(time.Minute)

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

	// 并发量高的话可以调大一点
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(20)
	sqlDB.SetConnMaxLifetime(time.Minute * 3)
	sqlDB.SetConnMaxIdleTime(time.Minute)

	concurrentBenchDB, err = Open(sqlDB, "mysql",
		WithPoolSize(20, 50),
		WithPoolTimeouts(time.Minute, time.Minute*3))
	if err != nil {
		panic(err)
	}

	err = concurrentBenchDB.MigrateModel(context.Background(), &BenchmarkUser{},
		WithMigrationLog(false),
		WithStrategy(ForceRecreate))
	if err != nil {
		panic(err)
	}
}

func cleanupConcurrentBenchDB() {
	if concurrentBenchDB != nil {
		concurrentBenchDB.Close()
	}
}

func runWithWorkerPool(workCount int, maxWorkers int, work func(int)) {
	if workCount <= 0 {
		return
	}

	workers := maxWorkers
	if workCount < workers {
		workers = workCount
	}

	jobs := make(chan int, workCount)
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for jobID := range jobs {
				work(jobID)
			}
		}()
	}

	for i := 0; i < workCount; i++ {
		jobs <- i
	}
	close(jobs)

	wg.Wait()
}

func BenchmarkConcurrentInsert(b *testing.B) {
	setupConcurrentBenchDB()
	defer cleanupConcurrentBenchDB()

	ctx := context.Background()

	_, err := concurrentBenchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	maxWorkers := 20
	runWithWorkerPool(b.N, maxWorkers, func(id int) {
		user := &BenchmarkUser{
			Name:      fmt.Sprintf("User %d", id),
			Email:     fmt.Sprintf("user%d@example.com", id),
			Age:       id % 100,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    id%2 == 0,
		}

		result, err := RegisterInserter[BenchmarkUser](concurrentBenchDB).
			Insert(nil, user).
			Exec(ctx)
		if err != nil {
			b.Logf("Insert error: %v", err)
			return
		}

		_, err = result.LastInsertId()
		if err != nil {
			b.Logf("LastInsertId error: %v", err)
		}
	})
}

func BenchmarkConcurrentBatchInsert(b *testing.B) {
	setupConcurrentBenchDB()
	defer cleanupConcurrentBenchDB()

	ctx := context.Background()
	batchSize := 100
	batches := b.N / batchSize
	if batches == 0 {
		batches = 1
	}

	_, err := concurrentBenchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	maxWorkers := 10
	runWithWorkerPool(batches, maxWorkers, func(batchIndex int) {
		users := make([]*BenchmarkUser, 0, batchSize)
		startID := batchIndex * batchSize

		for i := 0; i < batchSize; i++ {
			user := &BenchmarkUser{
				Name:      fmt.Sprintf("User %d", startID+i),
				Email:     fmt.Sprintf("user%d@example.com", startID+i),
				Age:       (startID+i) % 100,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Active:    (startID+i)%2 == 0,
			}
			users = append(users, user)
		}

		_, err := RegisterInserter[BenchmarkUser](concurrentBenchDB).
			Insert(nil, users...).
			Exec(ctx)
		if err != nil {
			b.Logf("Batch insert error: %v", err)
		}
	})
}

func prepareConcurrentSelectData(b *testing.B) {
	ctx := context.Background()

	_, err := concurrentBenchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatal(err)
	}

	batchSize := 25
	users := make([]*BenchmarkUser, 0, batchSize)

	for i := 0; i < 100; i++ {
		user := &BenchmarkUser{
			Name:      fmt.Sprintf("User %d", i),
			Email:     fmt.Sprintf("user%d@example.com", i),
			Age:       i % 100,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    i%2 == 0,
		}
		users = append(users, user)

		if len(users) == batchSize || i == 99 {
			_, err = RegisterInserter[BenchmarkUser](concurrentBenchDB).
				Insert(nil, users...).
				Exec(ctx)
			if err != nil {
				b.Fatal(err)
			}
			users = users[:0]
		}
	}
}

func BenchmarkConcurrentSelect(b *testing.B) {
	setupConcurrentBenchDB()
	defer cleanupConcurrentBenchDB()

	prepareConcurrentSelectData(b)

	ctx := context.Background()
	b.ResetTimer()

	maxWorkers := 20
	runWithWorkerPool(b.N, maxWorkers, func(id int) {
		_, err := RegisterSelector[BenchmarkUser](concurrentBenchDB).
			Select().
			Where(Col("ID").Eq(id%100 + 1)).
			Get(ctx)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			b.Logf("Select error: %v", err)
		}
	})
}

func BenchmarkConcurrentSelectMulti(b *testing.B) {
	setupConcurrentBenchDB()
	defer cleanupConcurrentBenchDB()

	prepareConcurrentSelectData(b)

	ctx := context.Background()
	b.ResetTimer()

	maxWorkers := 20
	runWithWorkerPool(b.N, maxWorkers, func(id int) {
		_, err := RegisterSelector[BenchmarkUser](concurrentBenchDB).
			Select().
			Where(Col("Age").Gt(id % 50)).
			Limit(10).
			GetMulti(ctx)
		if err != nil {
			b.Logf("SelectMulti error: %v", err)
		}
	})
}

func prepareConcurrentUpdateData(b *testing.B) {
	ctx := context.Background()

	_, err := concurrentBenchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatal(err)
	}

	batchSize := 25
	users := make([]*BenchmarkUser, 0, batchSize)

	for i := 0; i < 100; i++ {
		user := &BenchmarkUser{
			Name:      fmt.Sprintf("User %d", i),
			Email:     fmt.Sprintf("user%d@example.com", i),
			Age:       i % 100,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    i%2 == 0,
		}
		users = append(users, user)

		if len(users) == batchSize || i == 99 {
			_, err = RegisterInserter[BenchmarkUser](concurrentBenchDB).
				Insert(nil, users...).
				Exec(ctx)
			if err != nil {
				b.Fatal(err)
			}
			users = users[:0]
		}
	}
}

func BenchmarkConcurrentUpdate(b *testing.B) {
	setupConcurrentBenchDB()
	defer cleanupConcurrentBenchDB()

	prepareConcurrentUpdateData(b)

	ctx := context.Background()
	b.ResetTimer()

	maxWorkers := 20
	runWithWorkerPool(b.N, maxWorkers, func(id int) {
		_, err := RegisterUpdater[BenchmarkUser](concurrentBenchDB).
			Update().
			Set(Col("Name"), fmt.Sprintf("Updated User %d", id)).
			Set(Col("UpdatedAt"), time.Now()).
			Where(Col("ID").Eq(id%100 + 1)).
			Exec(ctx)
		if err != nil {
			b.Logf("Update error: %v", err)
		}
	})
}

func prepareConcurrentDeleteData(b *testing.B) {
	ctx := context.Background()

	_, err := concurrentBenchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatal(err)
	}

	batchSize := 50
	totalRows := 1000
	batches := totalRows / batchSize

	for batch := 0; batch < batches; batch++ {
		users := make([]*BenchmarkUser, 0, batchSize)
		startID := batch * batchSize

		for i := 0; i < batchSize; i++ {
			user := &BenchmarkUser{
				Name:      fmt.Sprintf("User %d", startID+i),
				Email:     fmt.Sprintf("user%d@example.com", startID+i),
				Age:       (startID+i) % 100,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Active:    (startID+i)%2 == 0,
			}
			users = append(users, user)
		}

		_, err = RegisterInserter[BenchmarkUser](concurrentBenchDB).
			Insert(nil, users...).
			Exec(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConcurrentDelete(b *testing.B) {
	setupConcurrentBenchDB()
	defer cleanupConcurrentBenchDB()

	prepareConcurrentDeleteData(b)

	ctx := context.Background()
	b.ResetTimer()

	maxWorkers := 20
	runWithWorkerPool(b.N, maxWorkers, func(id int) {
		_, err := RegisterDeleter[BenchmarkUser](concurrentBenchDB).
			Delete().
			Where(Col("ID").Eq(id%1000 + 1)).
			Exec(ctx)
		if err != nil {
			b.Logf("Delete error: %v", err)
		}
	})
}

func BenchmarkConcurrentTransaction(b *testing.B) {
	setupConcurrentBenchDB()
	defer cleanupConcurrentBenchDB()

	ctx := context.Background()
	_, err := concurrentBenchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	maxWorkers := 15
	runWithWorkerPool(b.N, maxWorkers, func(id int) {
		err := concurrentBenchDB.Tx(ctx, func(tx *Tx) error {
			// Insert a user
			user := &BenchmarkUser{
				Name:      fmt.Sprintf("TxUser %d", id),
				Email:     fmt.Sprintf("txuser%d@example.com", id),
				Age:       id % 100,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Active:    id%2 == 0,
			}

			result, err := RegisterInserter[BenchmarkUser](tx).
				Insert(nil, user).
				Exec(ctx)
			if err != nil {
				return err
			}

			lastID, err := result.LastInsertId()
			if err != nil {
				return err
			}

			_, err = RegisterUpdater[BenchmarkUser](tx).
				Update().
				Set(Col("Name"), fmt.Sprintf("Updated TxUser %d", id)).
				Where(Col("ID").Eq(lastID)).
				Exec(ctx)

			return err
		}, nil)

		if err != nil {
			b.Logf("Transaction error: %v", err)
		}
	})
}

func BenchmarkConcurrentMixedOperations(b *testing.B) {
	setupConcurrentBenchDB()
	defer cleanupConcurrentBenchDB()

	prepareConcurrentUpdateData(b)

	ctx := context.Background()
	b.ResetTimer()

	operationsPerGoroutine := 5
	maxWorkers := 15

	runWithWorkerPool(b.N, maxWorkers, func(id int) {
		for i := 0; i < operationsPerGoroutine; i++ {
			opType := (id + i) % 5

			switch opType {
			case 0:
				user := &BenchmarkUser{
					Name:      fmt.Sprintf("MixedUser %d-%d", id, i),
					Email:     fmt.Sprintf("mixeduser%d_%d@example.com", id, i),
					Age:       (id + i) % 100,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Active:    (id+i)%2 == 0,
				}

				_, err := RegisterInserter[BenchmarkUser](concurrentBenchDB).
					Insert(nil, user).
					Exec(ctx)
				if err != nil {
					b.Logf("Mixed insert error: %v", err)
				}

			case 1:
				_, err := RegisterSelector[BenchmarkUser](concurrentBenchDB).
					Select().
					Where(Col("ID").Eq((id+i)%100 + 1)).
					Get(ctx)
				if err != nil && !errors.Is(err, sql.ErrNoRows) {
					b.Logf("Mixed select error: %v", err)
				}

			case 2:
				_, err := RegisterSelector[BenchmarkUser](concurrentBenchDB).
					Select().
					Where(Col("Age").Gt((id+i) % 50)).
					Limit(5).
					GetMulti(ctx)
				if err != nil {
					b.Logf("Mixed select multi error: %v", err)
				}

			case 3:
				_, err := RegisterUpdater[BenchmarkUser](concurrentBenchDB).
					Update().
					Set(Col("Name"), fmt.Sprintf("Mixed Updated %d-%d", id, i)).
					Where(Col("ID").Eq((id+i)%100 + 1)).
					Exec(ctx)
				if err != nil {
					b.Logf("Mixed update error: %v", err)
				}

			case 4:
				_, err := RegisterDeleter[BenchmarkUser](concurrentBenchDB).
					Delete().
					Where(Col("ID").Eq((id+i)%100 + 1)).
					Exec(ctx)
				if err != nil {
					b.Logf("Mixed delete error: %v", err)
				}
			}
		}
	})
}
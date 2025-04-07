package orm

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func BenchmarkTransactionSimpleInsert(b *testing.B) {
	ctx := context.Background()

	_, err := benchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := benchDB.Tx(ctx, func(tx *Tx) error {
			user := &BenchmarkUser{
				Name:      fmt.Sprintf("User%d", i),
				Email:     fmt.Sprintf("user%d@example.com", i),
				Age:       25,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Active:    true,
			}

			result, err := RegisterInserter[BenchmarkUser](tx).Insert(nil, user).Exec(ctx)
			if err != nil {
				return err
			}

			_, err = result.LastInsertId()
			return err
		}, nil)

		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTransactionMultipleOperations(b *testing.B) {
	ctx := context.Background()

	if benchDB.cacheManager != nil {
		benchDB.cacheManager.Disable()
		defer benchDB.cacheManager.Enable()
	}

	_, err := benchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatal(err)
	}

	for i := 1; i <= 100; i++ {
		user := &BenchmarkUser{
			ID:        i,
			Name:      fmt.Sprintf("User%d", i),
			Email:     fmt.Sprintf("user%d@example.com", i),
			Age:       25,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    true,
		}

		_, err := RegisterInserter[BenchmarkUser](benchDB).
			Insert([]string{"ID", "Name", "Email", "Age", "CreatedAt", "UpdatedAt", "Active"}, user).
			Exec(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := benchDB.Tx(ctx, func(tx *Tx) error {
			id := i%100 + 1

			_, err := RegisterUpdater[BenchmarkUser](tx).Update().
				Set(Col("Name"), fmt.Sprintf("UpdatedUser%d", i)).
				Where(Col("ID").Eq(id)).
				Exec(ctx)
			if err != nil {
				return err
			}

			// Select operation
			user, err := RegisterSelector[BenchmarkUser](tx).Select().
				Where(Col("ID").Eq(id)).
				Get(ctx)
			if err != nil {
				return err
			}

			if user == nil {
				return fmt.Errorf("user with ID %d not found", id)
			}

			newUser := &BenchmarkUser{
				Name:      fmt.Sprintf("NewUser%d", i+1000),
				Email:     fmt.Sprintf("newuser%d@example.com", i+1000),
				Age:       30,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Active:    true,
			}
			_, err = RegisterInserter[BenchmarkUser](tx).Insert(nil, newUser).Exec(ctx)
			return err
		}, nil)

		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTransactionRollback(b *testing.B) {
	ctx := context.Background()

	_, err := benchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx, err := benchDB.BeginTx(ctx, nil)
		if err != nil {
			b.Fatal(err)
		}

		user := &BenchmarkUser{
			Name:      fmt.Sprintf("User%d", i),
			Email:     fmt.Sprintf("user%d@example.com", i),
			Age:       25,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    true,
		}

		_, err = RegisterInserter[BenchmarkUser](tx).Insert(nil, user).Exec(ctx)
		if err != nil {
			tx.RollBack()
			b.Fatal(err)
		}

		err = tx.RollBack()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTransactionCommit(b *testing.B) {
	ctx := context.Background()

	_, err := benchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx, err := benchDB.BeginTx(ctx, nil)
		if err != nil {
			b.Fatal(err)
		}

		user := &BenchmarkUser{
			Name:      fmt.Sprintf("User%d", i),
			Email:     fmt.Sprintf("user%d@example.com", i),
			Age:       25,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    true,
		}

		_, err = RegisterInserter[BenchmarkUser](tx).Insert(nil, user).Exec(ctx)
		if err != nil {
			tx.RollBack()
			b.Fatal(err)
		}

		err = tx.Commit()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTransactionBatchOperations(b *testing.B) {
	ctx := context.Background()

	_, err := benchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatal(err)
	}

	batchSize := 10
	batches := b.N / batchSize
	if batches == 0 {
		batches = 1
	}

	b.ResetTimer()
	for batch := 0; batch < batches; batch++ {
		err := benchDB.Tx(ctx, func(tx *Tx) error {
			for i := 0; i < batchSize; i++ {
				index := batch*batchSize + i
				user := &BenchmarkUser{
					Name:      fmt.Sprintf("User%d", index),
					Email:     fmt.Sprintf("user%d@example.com", index),
					Age:       25,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Active:    true,
				}

				_, err := RegisterInserter[BenchmarkUser](tx).Insert(nil, user).Exec(ctx)
				if err != nil {
					return err
				}
			}
			return nil
		}, nil)

		if err != nil {
			b.Fatal(err)
		}
	}
}
package orm

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func BenchmarkModelInsert(b *testing.B) {
	ctx := context.Background()

	_, err := benchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		user := &BenchmarkUser{
			Name:      fmt.Sprintf("user_%d", i),
			Email:     fmt.Sprintf("user_%d@example.com", i),
			Age:       20 + i%20,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    true,
		}

		inserter := RegisterInserter[BenchmarkUser](benchDB)
		_, err := inserter.Insert(nil, user).Exec(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkModelBatchInsert(b *testing.B) {
	ctx := context.Background()
	batchSize := 100
	batches := b.N / batchSize
	if batches == 0 {
		batches = 1
	}

	_, err := benchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for batch := 0; batch < batches; batch++ {
		users := make([]*BenchmarkUser, 0, batchSize)
		for i := 0; i < batchSize; i++ {
			user := &BenchmarkUser{
				Name:      fmt.Sprintf("user_%d", batch*batchSize+i),
				Email:     fmt.Sprintf("user_%d@example.com", batch*batchSize+i),
				Age:       20 + i%20,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Active:    true,
			}
			users = append(users, user)
		}

		inserter := RegisterInserter[BenchmarkUser](benchDB)
		_, err := inserter.Insert(nil, users...).Exec(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func prepareModelSelectData(b *testing.B) {
	ctx := context.Background()

	_, err := benchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatal(err)
	}

	users := make([]*BenchmarkUser, 0, 100)
	for i := 0; i < 100; i++ {
		user := &BenchmarkUser{
			ID:        i + 1, // Explicitly set ID to ensure it starts from 1
			Name:      fmt.Sprintf("user_%d", i),
			Email:     fmt.Sprintf("user_%d@example.com", i),
			Age:       20 + i%20,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    true,
		}
		users = append(users, user)
	}

	inserter := RegisterInserter[BenchmarkUser](benchDB)

	_, err = inserter.Insert([]string{"ID", "Name", "Email", "Age", "CreatedAt", "UpdatedAt", "Active"}, users...).Exec(ctx)
	if err != nil {
		b.Fatal(err)
	}
}

func BenchmarkModelSelectById(b *testing.B) {
	ctx := context.Background()
	prepareModelSelectData(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		selector := RegisterSelector[BenchmarkUser](benchDB)
		user, err := selector.Select().Where(Col("ID").Eq((i % 100) + 1)).Get(ctx)
		if err != nil {
			b.Fatal(err)
		}
		if user == nil {
			b.Fatal("user not found")
		}
	}
}

func BenchmarkModelSelectAll(b *testing.B) {
	ctx := context.Background()
	prepareModelSelectData(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		selector := RegisterSelector[BenchmarkUser](benchDB)
		users, err := selector.Select().GetMulti(ctx)
		if err != nil {
			b.Fatal(err)
		}
		if len(users) == 0 {
			b.Fatal("no users found")
		}
	}
}

func BenchmarkModelSelectWithCondition(b *testing.B) {
	ctx := context.Background()
	prepareModelSelectData(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		age := 20 + i%20
		selector := RegisterSelector[BenchmarkUser](benchDB)
		_, err := selector.Select().Where(Col("Age").Eq(age)).GetMulti(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func prepareModelUpdateData(b *testing.B) {
	ctx := context.Background()

	_, err := benchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatal(err)
	}

	users := make([]*BenchmarkUser, 0, 100)
	for i := 0; i < 100; i++ {
		user := &BenchmarkUser{
			ID:        i + 1, // Explicitly set ID
			Name:      fmt.Sprintf("user_%d", i),
			Email:     fmt.Sprintf("user_%d@example.com", i),
			Age:       20 + i%20,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    true,
		}
		users = append(users, user)
	}

	inserter := RegisterInserter[BenchmarkUser](benchDB)

	_, err = inserter.Insert([]string{"ID", "Name", "Email", "Age", "CreatedAt", "UpdatedAt", "Active"}, users...).Exec(ctx)
	if err != nil {
		b.Fatal(err)
	}
}

func BenchmarkModelUpdate(b *testing.B) {
	ctx := context.Background()
	prepareModelUpdateData(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		updater := RegisterUpdater[BenchmarkUser](benchDB)
		_, err := updater.Update().
			Set(Col("Name"), fmt.Sprintf("updated_user_%d", i)).
			Set(Col("UpdatedAt"), time.Now()).
			Where(Col("ID").Eq((i % 100) + 1)).
			Exec(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkModelBatchUpdate(b *testing.B) {
	ctx := context.Background()
	prepareModelUpdateData(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		updater := RegisterUpdater[BenchmarkUser](benchDB)
		_, err := updater.Update().
			SetMulti(map[string]any{
				"Name":      fmt.Sprintf("batch_updated_%d", i),
				"UpdatedAt": time.Now(),
				"Active":    false,
			}).
			Where(Col("Age").Eq(20 + i%20)).
			Exec(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func prepareModelDeleteData(b *testing.B) {
	ctx := context.Background()

	_, err := benchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatal(err)
	}

	users := make([]*BenchmarkUser, 0, 100)
	for i := 0; i < 100; i++ {
		user := &BenchmarkUser{
			ID:        i + 1, // Explicitly set ID
			Name:      fmt.Sprintf("user_%d", i),
			Email:     fmt.Sprintf("user_%d@example.com", i),
			Age:       20 + i%20,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    true,
		}
		users = append(users, user)
	}

	inserter := RegisterInserter[BenchmarkUser](benchDB)

	_, err = inserter.Insert([]string{"ID", "Name", "Email", "Age", "CreatedAt", "UpdatedAt", "Active"}, users...).Exec(ctx)
	if err != nil {
		b.Fatal(err)
	}
}

func BenchmarkModelDelete(b *testing.B) {
	ctx := context.Background()
	prepareModelDeleteData(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		deleter := RegisterDeleter[BenchmarkUser](benchDB)
		_, err := deleter.Delete().Where(Col("ID").Eq((i % 100) + 1)).Exec(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkModelTransaction(b *testing.B) {
	ctx := context.Background()

	_, err := benchDB.execContext(ctx, "DELETE FROM benchmark_user")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := benchDB.Tx(ctx, func(tx *Tx) error {
			user := &BenchmarkUser{
				ID:        i + 1, // Explicitly set ID
				Name:      fmt.Sprintf("tx_user_%d", i),
				Email:     fmt.Sprintf("tx_user_%d@example.com", i),
				Age:       20 + i%20,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Active:    true,
			}

			inserter := RegisterInserter[BenchmarkUser](tx)
			_, err := inserter.Insert([]string{"ID", "Name", "Email", "Age", "CreatedAt", "UpdatedAt", "Active"}, user).Exec(ctx)
			if err != nil {
				return err
			}

			updater := RegisterUpdater[BenchmarkUser](tx)
			_, err = updater.Update().
				Set(Col("Name"), fmt.Sprintf("tx_updated_%d", i)).
				Where(Col("Email").Eq(fmt.Sprintf("tx_user_%d@example.com", i))).
				Exec(ctx)

			return err
		}, nil)

		if err != nil {
			b.Fatal(err)
		}
	}
}
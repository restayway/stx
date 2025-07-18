package stx

import (
	"context"
	"errors"
	"sync"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type TestModel struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"not null"`
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(&TestModel{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

func TestNew(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	newCtx := New(ctx, db)
	if newCtx == nil {
		t.Fatal("expected non-nil context")
	}

	retrievedDB := GetCurrent(newCtx)
	if retrievedDB == nil {
		t.Fatal("expected to retrieve DB from context")
	}
}

func TestGetCurrent(t *testing.T) {
	tests := []struct {
		name      string
		setupCtx  func() context.Context
		expectNil bool
	}{
		{
			name:      "nil context",
			setupCtx:  func() context.Context { return nil },
			expectNil: true,
		},
		{
			name:      "context without STX",
			setupCtx:  func() context.Context { return context.Background() },
			expectNil: true,
		},
		{
			name: "context with STX",
			setupCtx: func() context.Context {
				db := setupTestDB(t)
				return New(context.Background(), db)
			},
			expectNil: false,
		},
		{
			name: "context with invalid value type",
			setupCtx: func() context.Context {
				return context.WithValue(context.Background(), txContextKey, "invalid")
			},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			db := GetCurrent(ctx)

			if tt.expectNil && db != nil {
				t.Error("expected nil DB")
			}
			if !tt.expectNil && db == nil {
				t.Error("expected non-nil DB")
			}
		})
	}
}

func TestWithTransaction(t *testing.T) {
	db := setupTestDB(t)
	ctx := New(context.Background(), db)

	t.Run("successful transaction", func(t *testing.T) {
		var count int64
		err := WithTransaction(ctx, func(txCtx context.Context) error {
			txDB := GetCurrent(txCtx)
			if txDB == nil {
				t.Fatal("expected DB in transaction context")
			}

			model := TestModel{Name: "test"}
			if err := txDB.Create(&model).Error; err != nil {
				return err
			}

			txDB.Model(&TestModel{}).Count(&count)
			if count != 1 {
				t.Errorf("expected 1 record in transaction, got %d", count)
			}

			return nil
		})

		if err != nil {
			t.Fatalf("transaction failed: %v", err)
		}

		db.Model(&TestModel{}).Count(&count)
		if count != 1 {
			t.Errorf("expected 1 record after commit, got %d", count)
		}
	})

	t.Run("failed transaction rollback", func(t *testing.T) {
		var countBefore int64
		db.Model(&TestModel{}).Count(&countBefore)

		testErr := errors.New("test error")
		err := WithTransaction(ctx, func(txCtx context.Context) error {
			txDB := GetCurrent(txCtx)
			model := TestModel{Name: "rollback-test"}
			if err := txDB.Create(&model).Error; err != nil {
				return err
			}
			return testErr
		})

		if err != testErr {
			t.Fatalf("expected test error, got: %v", err)
		}

		var countAfter int64
		db.Model(&TestModel{}).Count(&countAfter)
		if countAfter != countBefore {
			t.Errorf("expected count to remain %d after rollback, got %d", countBefore, countAfter)
		}
	})

	t.Run("with nil context", func(t *testing.T) {
		err := WithTransaction(nil, func(ctx context.Context) error {
			return nil
		})
		if err != gorm.ErrInvalidTransaction {
			t.Errorf("expected ErrInvalidTransaction, got: %v", err)
		}
	})

	t.Run("with context without DB", func(t *testing.T) {
		err := WithTransaction(context.Background(), func(ctx context.Context) error {
			return nil
		})
		if err != gorm.ErrInvalidTransaction {
			t.Errorf("expected ErrInvalidTransaction, got: %v", err)
		}
	})
}

func TestBeginCommitRollback(t *testing.T) {
	db := setupTestDB(t)
	ctx := New(context.Background(), db)

	t.Run("begin and commit", func(t *testing.T) {
		txCtx := Begin(ctx)
		txDB := GetCurrent(txCtx)
		if txDB == nil {
			t.Fatal("expected DB after Begin")
		}

		model := TestModel{Name: "begin-commit-test"}
		if err := txDB.Create(&model).Error; err != nil {
			t.Fatalf("failed to create model: %v", err)
		}

		if err := Commit(txCtx); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		var count int64
		db.Model(&TestModel{}).Where("name = ?", "begin-commit-test").Count(&count)
		if count != 1 {
			t.Errorf("expected 1 record after commit, got %d", count)
		}
	})

	t.Run("begin and rollback", func(t *testing.T) {
		txCtx := Begin(ctx)
		txDB := GetCurrent(txCtx)

		model := TestModel{Name: "begin-rollback-test"}
		if err := txDB.Create(&model).Error; err != nil {
			t.Fatalf("failed to create model: %v", err)
		}

		if err := Rollback(txCtx); err != nil {
			t.Fatalf("failed to rollback: %v", err)
		}

		var count int64
		db.Model(&TestModel{}).Where("name = ?", "begin-rollback-test").Count(&count)
		if count != 0 {
			t.Errorf("expected 0 records after rollback, got %d", count)
		}
	})

	t.Run("begin with nil context", func(t *testing.T) {
		newCtx := Begin(nil)
		if newCtx != nil {
			t.Error("expected nil context when Begin called with nil")
		}
	})

	t.Run("commit with context without transaction", func(t *testing.T) {
		err := Commit(context.Background())
		if err != nil {
			t.Errorf("expected nil, got: %v", err)
		}
	})

	t.Run("rollback with context without transaction", func(t *testing.T) {
		err := Rollback(context.Background())
		if err != nil {
			t.Errorf("expected nil, got: %v", err)
		}
	})

	t.Run("commit with nil context", func(t *testing.T) {
		err := Commit(nil)
		if err != nil {
			t.Errorf("expected nil, got: %v", err)
		}
	})

	t.Run("rollback with nil context", func(t *testing.T) {
		err := Rollback(nil)
		if err != nil {
			t.Errorf("expected nil, got: %v", err)
		}
	})
}

func TestIsTransaction(t *testing.T) {
	db := setupTestDB(t)
	ctx := New(context.Background(), db)

	t.Run("not in transaction", func(t *testing.T) {
		if IsTransaction(ctx) {
			t.Error("expected IsTransaction to return false")
		}
	})

	t.Run("in transaction", func(t *testing.T) {
		txCtx := Begin(ctx)
		if !IsTransaction(txCtx) {
			t.Error("expected IsTransaction to return true")
		}
		Rollback(txCtx)
	})

	t.Run("nil context", func(t *testing.T) {
		if IsTransaction(nil) {
			t.Error("expected IsTransaction to return false for nil context")
		}
	})

	t.Run("context without DB", func(t *testing.T) {
		if IsTransaction(context.Background()) {
			t.Error("expected IsTransaction to return false for context without DB")
		}
	})
}

func TestConcurrency(t *testing.T) {
	db := setupTestDB(t)
	ctx := New(context.Background(), db)

	var wg sync.WaitGroup
	const numRoutines = 10

	for i := 0; i < numRoutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Test concurrent access to GetCurrent
			retrievedDB := GetCurrent(ctx)
			if retrievedDB == nil {
				t.Errorf("GetCurrent returned nil in goroutine %d", id)
				return
			}

			// Test concurrent context creation
			newCtx := New(context.Background(), retrievedDB)
			if newCtx == nil {
				t.Errorf("New returned nil context in goroutine %d", id)
				return
			}

			// Test concurrent transaction context creation
			txCtx := Begin(newCtx)
			txDB := GetCurrent(txCtx)
			if txDB == nil {
				t.Errorf("Failed to get transaction DB in goroutine %d", id)
				return
			}

			// Test IsTransaction
			if !IsTransaction(txCtx) {
				t.Errorf("IsTransaction returned false for transaction context in goroutine %d", id)
			}

			// Clean up
			Rollback(txCtx)
		}(i)
	}

	wg.Wait()
}

func TestNestedTransactions(t *testing.T) {
	db := setupTestDB(t)
	ctx := New(context.Background(), db)

	var initialCount int64
	db.Model(&TestModel{}).Count(&initialCount)

	err := WithTransaction(ctx, func(outerCtx context.Context) error {
		outerDB := GetCurrent(outerCtx)
		model1 := TestModel{Name: "outer"}
		if err := outerDB.Create(&model1).Error; err != nil {
			return err
		}

		return WithTransaction(outerCtx, func(innerCtx context.Context) error {
			innerDB := GetCurrent(innerCtx)
			model2 := TestModel{Name: "inner"}
			return innerDB.Create(&model2).Error
		})
	})

	if err != nil {
		t.Fatalf("nested transaction failed: %v", err)
	}

	var finalCount int64
	db.Model(&TestModel{}).Count(&finalCount)
	expectedCount := initialCount + 2
	if finalCount != expectedCount {
		t.Errorf("expected %d records after nested transaction, got %d", expectedCount, finalCount)
	}
}

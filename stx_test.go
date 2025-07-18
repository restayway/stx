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

	retrievedDB := Current(newCtx)
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
			db := Current(ctx)

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
			txDB := Current(txCtx)
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
			txDB := Current(txCtx)
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
		txDB := Current(txCtx)
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
		txDB := Current(txCtx)

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
		if IsTx(ctx) {
			t.Error("expected IsTx to return false")
		}
	})

	t.Run("in transaction", func(t *testing.T) {
		txCtx := Begin(ctx)
		if !IsTx(txCtx) {
			t.Error("expected IsTx to return true")
		}
		Rollback(txCtx)
	})

	t.Run("nil context", func(t *testing.T) {
		if IsTx(nil) {
			t.Error("expected IsTx to return false for nil context")
		}
	})

	t.Run("context without DB", func(t *testing.T) {
		if IsTx(context.Background()) {
			t.Error("expected IsTx to return false for context without DB")
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

			// Test concurrent access to Current
			retrievedDB := Current(ctx)
			if retrievedDB == nil {
				t.Errorf("Current returned nil in goroutine %d", id)
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
			txDB := Current(txCtx)
			if txDB == nil {
				t.Errorf("Failed to get transaction DB in goroutine %d", id)
				return
			}

			// Test IsTx
			if !IsTx(txCtx) {
				t.Errorf("IsTx returned false for transaction context in goroutine %d", id)
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
		outerDB := Current(outerCtx)
		model1 := TestModel{Name: "outer"}
		if err := outerDB.Create(&model1).Error; err != nil {
			return err
		}

		return WithTransaction(outerCtx, func(innerCtx context.Context) error {
			innerDB := Current(innerCtx)
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

func TestWithDefer(t *testing.T) {
	db := setupTestDB(t)
	ctx := New(context.Background(), db)

	t.Run("successful operation with defer", func(t *testing.T) {
		var initialCount int64
		db.Model(&TestModel{}).Count(&initialCount)

		err := func() (err error) {
			txCtx, cleanup := WithDefer(ctx)
			defer cleanup(&err)

			txDB := Current(txCtx)
			if txDB == nil {
				return errors.New("expected DB in transaction context")
			}

			model := TestModel{Name: "defer-success"}
			if err := txDB.Create(&model).Error; err != nil {
				return err
			}

			return nil
		}()

		if err != nil {
			t.Fatalf("defer operation failed: %v", err)
		}

		var finalCount int64
		db.Model(&TestModel{}).Count(&finalCount)
		if finalCount != initialCount+1 {
			t.Errorf("expected %d records after successful defer, got %d", initialCount+1, finalCount)
		}
	})

	t.Run("failed operation with defer", func(t *testing.T) {
		var initialCount int64
		db.Model(&TestModel{}).Count(&initialCount)

		testErr := errors.New("test error")
		err := func() (err error) {
			txCtx, cleanup := WithDefer(ctx)
			defer cleanup(&err)

			txDB := Current(txCtx)
			model := TestModel{Name: "defer-fail"}
			if err := txDB.Create(&model).Error; err != nil {
				return err
			}

			return testErr // This should trigger rollback
		}()

		if err != testErr {
			t.Fatalf("expected test error, got: %v", err)
		}

		var finalCount int64
		db.Model(&TestModel{}).Count(&finalCount)
		if finalCount != initialCount {
			t.Errorf("expected %d records after failed defer (rollback), got %d", initialCount, finalCount)
		}
	})

	t.Run("panic recovery with defer", func(t *testing.T) {
		var initialCount int64
		db.Model(&TestModel{}).Count(&initialCount)

		err := func() (err error) {
			txCtx, cleanup := WithDefer(ctx)
			defer cleanup(&err)

			txDB := Current(txCtx)
			model := TestModel{Name: "defer-panic"}
			if err := txDB.Create(&model).Error; err != nil {
				return err
			}

			panic("test panic")
		}()

		if err == nil {
			t.Fatal("expected error from panic recovery")
		}

		expectedMsg := "recovered from panic: test panic"
		if err.Error() != expectedMsg {
			t.Errorf("expected panic recovery error message %q, got %q", expectedMsg, err.Error())
		}

		var finalCount int64
		db.Model(&TestModel{}).Count(&finalCount)
		if finalCount != initialCount {
			t.Errorf("expected %d records after panic (rollback), got %d", initialCount, finalCount)
		}
	})

	t.Run("defer with nil context", func(t *testing.T) {
		txCtx, cleanup := WithDefer(nil)
		if txCtx != nil {
			t.Error("expected nil context when WithDefer called with nil")
		}

		var err error
		cleanup(&err)
		if err != nil {
			t.Errorf("expected nil error for cleanup with nil context, got: %v", err)
		}
	})

	t.Run("defer with context without DB", func(t *testing.T) {
		txCtx, cleanup := WithDefer(context.Background())
		if txCtx != context.Background() {
			t.Error("expected same context when WithDefer called with context without DB")
		}

		var err error
		cleanup(&err)
		if err != nil {
			t.Errorf("expected nil error for cleanup with context without DB, got: %v", err)
		}
	})

	t.Run("defer transaction status", func(t *testing.T) {
		txCtx, cleanup := WithDefer(ctx)
		defer func() {
			var err error
			cleanup(&err)
		}()

		if !IsTx(txCtx) {
			t.Error("expected IsTx to return true for defer transaction context")
		}

		txDB := Current(txCtx)
		if txDB == nil {
			t.Error("expected DB from defer transaction context")
		}
	})
}

package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/restayway/stx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type User struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"not null"`
	Age  int
}

func main() {
	// Open database connection
	db, err := gorm.Open(sqlite.Open("example.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate the schema
	db.AutoMigrate(&User{})

	// Create context with database
	ctx := stx.New(context.Background(), db)

	// Example 1: Basic usage
	fmt.Println("=== Basic Usage ===")
	basicUsage(ctx)

	// Example 2: Transaction management
	fmt.Println("\n=== Transaction Management ===")
	transactionExample(ctx)

	// Example 3: Manual transaction control
	fmt.Println("\n=== Manual Transaction Control ===")
	manualTransactionExample(ctx)

	// Example 4: Nested transactions
	fmt.Println("\n=== Nested Transactions ===")
	nestedTransactionExample(ctx)

	// Example 5: Defer pattern
	fmt.Println("\n=== Defer Pattern ===")
	deferPatternExample(ctx)

	fmt.Println("\n=== Done ===")
}

func basicUsage(ctx context.Context) {
	// Get database from context
	db := stx.Current(ctx)

	// Create a user
	user := User{Name: "John Doe", Age: 30}
	result := db.Create(&user)
	if result.Error != nil {
		log.Printf("Error creating user: %v", result.Error)
		return
	}

	fmt.Printf("Created user: %+v\n", user)
}

func transactionExample(ctx context.Context) {
	// Use automatic transaction management
	err := stx.WithTransaction(ctx, func(txCtx context.Context) error {
		txDB := stx.Current(txCtx)

		// Create multiple users in a transaction
		users := []User{
			{Name: "Alice", Age: 25},
			{Name: "Bob", Age: 35},
		}

		for _, user := range users {
			if err := txDB.Create(&user).Error; err != nil {
				return err // This will rollback the transaction
			}
			fmt.Printf("Created user in transaction: %+v\n", user)
		}

		return nil // Transaction will be committed
	})

	if err != nil {
		log.Printf("Transaction failed: %v", err)
	} else {
		fmt.Println("Transaction completed successfully")
	}
}

func manualTransactionExample(ctx context.Context) {
	// Begin transaction manually
	txCtx := stx.Begin(ctx)
	txDB := stx.Current(txCtx)

	// Check if we're in a transaction
	if stx.IsTx(txCtx) {
		fmt.Println("Successfully started transaction")
	}

	// Create a user
	user := User{Name: "Charlie", Age: 28}
	if err := txDB.Create(&user).Error; err != nil {
		fmt.Printf("Error creating user: %v\n", err)
		stx.Rollback(txCtx)
		return
	}

	fmt.Printf("Created user: %+v\n", user)

	// Commit the transaction
	if err := stx.Commit(txCtx); err != nil {
		fmt.Printf("Error committing transaction: %v\n", err)
	} else {
		fmt.Println("Transaction committed successfully")
	}

	// Demonstrate graceful handling - safe to call even without transaction
	fmt.Println("Calling commit on non-transaction context...")
	if err := stx.Commit(ctx); err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Gracefully handled: no error returned")
	}
}

func nestedTransactionExample(ctx context.Context) {
	err := stx.WithTransaction(ctx, func(outerCtx context.Context) error {
		outerDB := stx.Current(outerCtx)

		// Create user in outer transaction
		user1 := User{Name: "David", Age: 40}
		if err := outerDB.Create(&user1).Error; err != nil {
			return err
		}
		fmt.Printf("Created user in outer transaction: %+v\n", user1)

		// Nested transaction
		return stx.WithTransaction(outerCtx, func(innerCtx context.Context) error {
			innerDB := stx.Current(innerCtx)

			// Create user in inner transaction
			user2 := User{Name: "Eve", Age: 22}
			if err := innerDB.Create(&user2).Error; err != nil {
				return err
			}
			fmt.Printf("Created user in inner transaction: %+v\n", user2)

			return nil
		})
	})

	if err != nil {
		log.Printf("Nested transaction failed: %v", err)
	} else {
		fmt.Println("Nested transaction completed successfully")
	}
}

func deferPatternExample(ctx context.Context) {
	// Example 1: Successful defer pattern
	fmt.Println("=== Successful Defer Pattern ===")
	err := deferSuccess(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Defer pattern completed successfully")
	}

	// Example 2: Error handling with defer
	fmt.Println("\n=== Error Handling with Defer ===")
	err = deferError(ctx)
	if err != nil {
		fmt.Printf("Expected error: %v\n", err)
	}

	// Example 3: Panic recovery with defer
	fmt.Println("\n=== Panic Recovery with Defer ===")
	err = deferPanic(ctx)
	if err != nil {
		fmt.Printf("Recovered from panic: %v\n", err)
	}
}

func deferSuccess(ctx context.Context) (err error) {
	txCtx, cleanup := stx.WithDefer(ctx)
	defer cleanup(&err)

	db := stx.Current(txCtx)
	
	// Create a user
	user := User{Name: "Defer Success", Age: 30}
	if err := db.Create(&user).Error; err != nil {
		return err
	}
	
	fmt.Printf("Created user with defer: %+v\n", user)
	return nil // Transaction will be committed automatically
}

func deferError(ctx context.Context) (err error) {
	txCtx, cleanup := stx.WithDefer(ctx)
	defer cleanup(&err)

	db := stx.Current(txCtx)
	
	// Create a user
	user := User{Name: "Defer Error", Age: 25}
	if err := db.Create(&user).Error; err != nil {
		return err
	}
	
	fmt.Printf("Created user but will rollback: %+v\n", user)
	return errors.New("simulated error") // This will trigger rollback
}

func deferPanic(ctx context.Context) (err error) {
	txCtx, cleanup := stx.WithDefer(ctx)
	defer cleanup(&err)

	db := stx.Current(txCtx)
	
	// Create a user
	user := User{Name: "Defer Panic", Age: 35}
	if err := db.Create(&user).Error; err != nil {
		return err
	}
	
	fmt.Printf("Created user but will panic: %+v\n", user)
	panic("simulated panic") // This will be recovered and trigger rollback
}

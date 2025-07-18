package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/restayway/stx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Account struct {
	ID      uint    `gorm:"primaryKey"`
	UserID  uint    `gorm:"not null"`
	Balance float64 `gorm:"not null;default:0"`
	User    User    `gorm:"foreignKey:UserID"`
}

type Transaction struct {
	ID        uint      `gorm:"primaryKey"`
	FromID    uint      `gorm:"not null"`
	ToID      uint      `gorm:"not null"`
	Amount    float64   `gorm:"not null"`
	Status    string    `gorm:"not null;default:'pending'"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	From      Account   `gorm:"foreignKey:FromID"`
	To        Account   `gorm:"foreignKey:ToID"`
}

func runAdvancedExamples() {
	// Open database connection
	db, err := gorm.Open(sqlite.Open("advanced_defer.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate the schema
	db.AutoMigrate(&User{}, &Account{}, &Transaction{})

	// Create context with database
	ctx := stx.New(context.Background(), db)

	// Example 1: Money transfer with defer
	fmt.Println("=== Advanced Example 1: Money Transfer ===")
	if err := moneyTransferWithDefer(ctx); err != nil {
		fmt.Printf("Transfer failed: %v\n", err)
	} else {
		fmt.Println("Success: Money transfer completed")
	}

	// Example 2: Concurrent operations with defer
	fmt.Println("\n=== Advanced Example 2: Concurrent Operations ===")
	concurrentOperationsWithDefer(ctx)

	// Example 3: Nested defer operations
	fmt.Println("\n=== Advanced Example 3: Nested Defer Operations ===")
	if err := nestedDeferOperations(ctx); err != nil {
		fmt.Printf("Nested operations failed: %v\n", err)
	} else {
		fmt.Println("Success: Nested operations completed")
	}

	// Example 4: Defer with timeout
	fmt.Println("\n=== Advanced Example 4: Defer with Timeout ===")
	if err := deferWithTimeout(ctx); err != nil {
		fmt.Printf("Operation with timeout failed: %v\n", err)
	} else {
		fmt.Println("Success: Operation with timeout completed")
	}

	// Example 5: Defer with retry logic
	fmt.Println("\n=== Advanced Example 5: Defer with Retry Logic ===")
	if err := deferWithRetry(ctx); err != nil {
		fmt.Printf("Retry operation failed: %v\n", err)
	} else {
		fmt.Println("Success: Retry operation completed")
	}

	fmt.Println("\n=== Advanced Examples Done ===")
}

// Advanced Example 1: Money transfer with defer
func moneyTransferWithDefer(ctx context.Context) (err error) {
	txCtx, cleanup := stx.WithDefer(ctx)
	defer cleanup(&err)

	db := stx.Current(txCtx)

	// Create users and accounts
	fromUser := User{Name: "Alice", Email: "alice@bank.com", Age: 30}
	toUser := User{Name: "Bob", Email: "bob@bank.com", Age: 25}

	if err := db.Create(&fromUser).Error; err != nil {
		return err
	}
	if err := db.Create(&toUser).Error; err != nil {
		return err
	}

	fromAccount := Account{UserID: fromUser.ID, Balance: 1000.0}
	toAccount := Account{UserID: toUser.ID, Balance: 500.0}

	if err := db.Create(&fromAccount).Error; err != nil {
		return err
	}
	if err := db.Create(&toAccount).Error; err != nil {
		return err
	}

	transferAmount := 200.0

	// Validate transfer
	if fromAccount.Balance < transferAmount {
		return errors.New("insufficient funds")
	}

	// Create transaction record
	transaction := Transaction{
		FromID: fromAccount.ID,
		ToID:   toAccount.ID,
		Amount: transferAmount,
		Status: "processing",
	}

	if err := db.Create(&transaction).Error; err != nil {
		return err
	}

	// Update balances
	fromAccount.Balance -= transferAmount
	toAccount.Balance += transferAmount

	if err := db.Save(&fromAccount).Error; err != nil {
		return err
	}
	if err := db.Save(&toAccount).Error; err != nil {
		return err
	}

	// Mark transaction as completed
	transaction.Status = "completed"
	if err := db.Save(&transaction).Error; err != nil {
		return err
	}

	fmt.Printf("Transferred $%.2f from account %d to account %d\n", 
		transferAmount, fromAccount.ID, toAccount.ID)
	
	return nil
}

// Advanced Example 2: Concurrent operations with defer
func concurrentOperationsWithDefer(ctx context.Context) {
	var wg sync.WaitGroup
	results := make(chan error, 3)

	// Run multiple concurrent operations
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			err := func() (err error) {
				txCtx, cleanup := stx.WithDefer(ctx)
				defer cleanup(&err)

				db := stx.Current(txCtx)
				
				user := User{
					Name:  fmt.Sprintf("Concurrent User %d", id),
					Email: fmt.Sprintf("user%d@concurrent.com", id),
					Age:   20 + id,
				}

				if err := db.Create(&user).Error; err != nil {
					return err
				}

				// Simulate some work
				time.Sleep(time.Duration(id*10) * time.Millisecond)

				fmt.Printf("Concurrent operation %d: Created user %s\n", id, user.Name)
				return nil
			}()

			results <- err
		}(i)
	}

	// Wait for all operations to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	successCount := 0
	for err := range results {
		if err != nil {
			fmt.Printf("Concurrent operation failed: %v\n", err)
		} else {
			successCount++
		}
	}

	fmt.Printf("Concurrent operations: %d successful\n", successCount)
}

// Advanced Example 3: Nested defer operations
func nestedDeferOperations(ctx context.Context) (err error) {
	txCtx, cleanup := stx.WithDefer(ctx)
	defer cleanup(&err)

	db := stx.Current(txCtx)

	// Create main user
	user := User{
		Name:  "Nested User",
		Email: "nested@example.com",
		Age:   35,
	}

	if err := db.Create(&user).Error; err != nil {
		return err
	}

	fmt.Printf("Created main user: %s\n", user.Name)

	// Nested operation: Create account
	if err := createAccountForUser(txCtx, user.ID); err != nil {
		return err
	}

	// Nested operation: Create initial transaction
	if err := createInitialTransaction(txCtx, user.ID); err != nil {
		return err
	}

	return nil
}

// Helper function for nested operations
func createAccountForUser(ctx context.Context, userID uint) (err error) {
	// Note: This uses the same transaction context, no new defer needed
	db := stx.Current(ctx)

	account := Account{
		UserID:  userID,
		Balance: 1000.0,
	}

	if err := db.Create(&account).Error; err != nil {
		return err
	}

	fmt.Printf("Created account for user %d with balance $%.2f\n", userID, account.Balance)
	return nil
}

// Helper function for nested operations
func createInitialTransaction(ctx context.Context, userID uint) (err error) {
	// Note: This uses the same transaction context, no new defer needed
	db := stx.Current(ctx)

	// Find the account
	var account Account
	if err := db.Where("user_id = ?", userID).First(&account).Error; err != nil {
		return err
	}

	transaction := Transaction{
		FromID: account.ID,
		ToID:   account.ID,
		Amount: 0.0,
		Status: "initial",
	}

	if err := db.Create(&transaction).Error; err != nil {
		return err
	}

	fmt.Printf("Created initial transaction for account %d\n", account.ID)
	return nil
}

// Advanced Example 4: Defer with timeout
func deferWithTimeout(ctx context.Context) (err error) {
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	txCtx, cleanup := stx.WithDefer(timeoutCtx)
	defer cleanup(&err)

	db := stx.Current(txCtx)

	user := User{
		Name:  "Timeout User",
		Email: "timeout@example.com",
		Age:   40,
	}

	if err := db.Create(&user).Error; err != nil {
		return err
	}

	// Simulate long-running operation
	select {
	case <-time.After(150 * time.Millisecond):
		fmt.Println("Long operation completed")
	case <-timeoutCtx.Done():
		return timeoutCtx.Err()
	}

	fmt.Printf("Created user with timeout: %s\n", user.Name)
	return nil
}

// Advanced Example 5: Defer with retry logic
func deferWithRetry(ctx context.Context) error {
	const maxRetries = 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := func() (err error) {
			txCtx, cleanup := stx.WithDefer(ctx)
			defer cleanup(&err)

			db := stx.Current(txCtx)

			user := User{
				Name:  fmt.Sprintf("Retry User %d", attempt),
				Email: fmt.Sprintf("retry%d@example.com", attempt),
				Age:   20 + attempt,
			}

			if err := db.Create(&user).Error; err != nil {
				return err
			}

			// Simulate operation that might fail
			if attempt < 3 {
				return fmt.Errorf("simulated failure on attempt %d", attempt)
			}

			fmt.Printf("Retry successful on attempt %d: Created user %s\n", attempt, user.Name)
			return nil
		}()

		if err == nil {
			return nil
		}

		lastErr = err
		fmt.Printf("Attempt %d failed: %v\n", attempt, err)
		
		if attempt < maxRetries {
			time.Sleep(time.Duration(attempt*100) * time.Millisecond)
		}
	}

	return fmt.Errorf("all retry attempts failed, last error: %v", lastErr)
}
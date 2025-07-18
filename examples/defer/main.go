package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/restayway/stx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type User struct {
	ID        uint      `gorm:"primaryKey"`
	Name      string    `gorm:"not null"`
	Email     string    `gorm:"unique;not null"`
	Age       int
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type Order struct {
	ID       uint   `gorm:"primaryKey"`
	UserID   uint   `gorm:"not null"`
	Product  string `gorm:"not null"`
	Quantity int    `gorm:"not null"`
	Total    float64
	User     User `gorm:"foreignKey:UserID"`
}

func main() {
	// Open database connection
	db, err := gorm.Open(sqlite.Open("defer_examples.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate the schema
	db.AutoMigrate(&User{}, &Order{})

	// Create context with database
	ctx := stx.New(context.Background(), db)

	// Example 1: Basic defer with success
	fmt.Println("=== Example 1: Basic Defer Success ===")
	if err := basicDeferSuccess(ctx); err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Success: User created and committed")
	}

	// Example 2: Defer with business logic error
	fmt.Println("\n=== Example 2: Defer with Business Logic Error ===")
	if err := deferWithBusinessError(ctx); err != nil {
		fmt.Printf("Expected error: %v\n", err)
	}

	// Example 3: Defer with panic recovery
	fmt.Println("\n=== Example 3: Defer with Panic Recovery ===")
	if err := deferWithPanicRecovery(ctx); err != nil {
		fmt.Printf("Panic recovered: %v\n", err)
	}

	// Example 4: Complex business transaction
	fmt.Println("\n=== Example 4: Complex Business Transaction ===")
	if err := complexBusinessTransaction(ctx); err != nil {
		fmt.Printf("Transaction failed: %v\n", err)
	} else {
		fmt.Println("Success: Order with user created")
	}

	// Example 5: Defer with validation
	fmt.Println("\n=== Example 5: Defer with Validation ===")
	if err := deferWithValidation(ctx); err != nil {
		fmt.Printf("Validation failed: %v\n", err)
	}

	// Example 6: Defer with multiple operations
	fmt.Println("\n=== Example 6: Defer with Multiple Operations ===")
	if err := deferWithMultipleOps(ctx); err != nil {
		fmt.Printf("Multiple operations failed: %v\n", err)
	} else {
		fmt.Println("Success: Multiple operations completed")
	}

	// Example 7: Defer with external API call simulation
	fmt.Println("\n=== Example 7: Defer with External API Call ===")
	if err := deferWithExternalAPI(ctx); err != nil {
		fmt.Printf("External API failed: %v\n", err)
	} else {
		fmt.Println("Success: User created with external API call")
	}

	// Example 8: Defer with conditional logic
	fmt.Println("\n=== Example 8: Defer with Conditional Logic ===")
	if err := deferWithConditionalLogic(ctx, true); err != nil {
		fmt.Printf("Conditional logic failed: %v\n", err)
	} else {
		fmt.Println("Success: Conditional logic completed")
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("=== ADVANCED EXAMPLES ===")
	fmt.Println(strings.Repeat("=", 50))

	// Run advanced examples
	runAdvancedExamples()

	fmt.Println("\n=== All Examples Done ===")
}

// Example 1: Basic defer with success
func basicDeferSuccess(ctx context.Context) (err error) {
	txCtx, cleanup := stx.WithDefer(ctx)
	defer cleanup(&err)

	db := stx.Current(txCtx)
	
	user := User{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
	}
	
	if err := db.Create(&user).Error; err != nil {
		return err
	}
	
	fmt.Printf("Created user: %s (ID: %d)\n", user.Name, user.ID)
	return nil
}

// Example 2: Defer with business logic error
func deferWithBusinessError(ctx context.Context) (err error) {
	txCtx, cleanup := stx.WithDefer(ctx)
	defer cleanup(&err)

	db := stx.Current(txCtx)
	
	user := User{
		Name:  "Jane Smith",
		Email: "jane@example.com",
		Age:   25,
	}
	
	if err := db.Create(&user).Error; err != nil {
		return err
	}
	
	fmt.Printf("Created user: %s (ID: %d)\n", user.Name, user.ID)
	
	// Simulate business logic error
	if user.Age < 30 {
		return errors.New("user must be at least 30 years old")
	}
	
	return nil
}

// Example 3: Defer with panic recovery
func deferWithPanicRecovery(ctx context.Context) (err error) {
	txCtx, cleanup := stx.WithDefer(ctx)
	defer cleanup(&err)

	db := stx.Current(txCtx)
	
	user := User{
		Name:  "Bob Wilson",
		Email: "bob@example.com",
		Age:   35,
	}
	
	if err := db.Create(&user).Error; err != nil {
		return err
	}
	
	fmt.Printf("Created user: %s (ID: %d)\n", user.Name, user.ID)
	
	// Simulate panic
	panic("something went wrong in business logic")
}

// Example 4: Complex business transaction
func complexBusinessTransaction(ctx context.Context) (err error) {
	txCtx, cleanup := stx.WithDefer(ctx)
	defer cleanup(&err)

	db := stx.Current(txCtx)
	
	// Create user
	user := User{
		Name:  "Alice Johnson",
		Email: "alice@example.com",
		Age:   28,
	}
	
	if err := db.Create(&user).Error; err != nil {
		return err
	}
	
	fmt.Printf("Created user: %s (ID: %d)\n", user.Name, user.ID)
	
	// Create order
	order := Order{
		UserID:   user.ID,
		Product:  "Laptop",
		Quantity: 1,
		Total:    999.99,
	}
	
	if err := db.Create(&order).Error; err != nil {
		return err
	}
	
	fmt.Printf("Created order: %s for user %d (Total: $%.2f)\n", 
		order.Product, order.UserID, order.Total)
	
	// Simulate inventory check
	if order.Quantity > 10 {
		return errors.New("insufficient inventory")
	}
	
	return nil
}

// Example 5: Defer with validation
func deferWithValidation(ctx context.Context) (err error) {
	txCtx, cleanup := stx.WithDefer(ctx)
	defer cleanup(&err)

	db := stx.Current(txCtx)
	
	user := User{
		Name:  "Charlie Brown",
		Email: "charlie@example.com",
		Age:   -5, // Invalid age
	}
	
	// Validate before creating
	if err := validateUser(&user); err != nil {
		return err
	}
	
	if err := db.Create(&user).Error; err != nil {
		return err
	}
	
	fmt.Printf("Created user: %s (ID: %d)\n", user.Name, user.ID)
	return nil
}

// Example 6: Defer with multiple operations
func deferWithMultipleOps(ctx context.Context) (err error) {
	txCtx, cleanup := stx.WithDefer(ctx)
	defer cleanup(&err)

	db := stx.Current(txCtx)
	
	// Create multiple users
	users := []User{
		{Name: "User1", Email: "user1@example.com", Age: 25},
		{Name: "User2", Email: "user2@example.com", Age: 30},
		{Name: "User3", Email: "user3@example.com", Age: 35},
	}
	
	for i, user := range users {
		if err := db.Create(&user).Error; err != nil {
			return err
		}
		users[i] = user
		fmt.Printf("Created user: %s (ID: %d)\n", user.Name, user.ID)
	}
	
	// Create orders for each user
	for _, user := range users {
		order := Order{
			UserID:   user.ID,
			Product:  "Product " + strconv.Itoa(int(user.ID)),
			Quantity: 2,
			Total:    float64(user.Age) * 10.0,
		}
		
		if err := db.Create(&order).Error; err != nil {
			return err
		}
		
		fmt.Printf("Created order for user %d: %s\n", user.ID, order.Product)
	}
	
	return nil
}

// Example 7: Defer with external API call simulation
func deferWithExternalAPI(ctx context.Context) (err error) {
	txCtx, cleanup := stx.WithDefer(ctx)
	defer cleanup(&err)

	db := stx.Current(txCtx)
	
	user := User{
		Name:  "David Miller",
		Email: "david@example.com",
		Age:   40,
	}
	
	if err := db.Create(&user).Error; err != nil {
		return err
	}
	
	fmt.Printf("Created user: %s (ID: %d)\n", user.Name, user.ID)
	
	// Simulate external API call
	if err := simulateExternalAPI(user.ID); err != nil {
		return err
	}
	
	fmt.Printf("External API call successful for user %d\n", user.ID)
	return nil
}

// Example 8: Defer with conditional logic
func deferWithConditionalLogic(ctx context.Context, shouldCreateOrder bool) (err error) {
	txCtx, cleanup := stx.WithDefer(ctx)
	defer cleanup(&err)

	db := stx.Current(txCtx)
	
	user := User{
		Name:  "Eve Davis",
		Email: "eve@example.com",
		Age:   32,
	}
	
	if err := db.Create(&user).Error; err != nil {
		return err
	}
	
	fmt.Printf("Created user: %s (ID: %d)\n", user.Name, user.ID)
	
	// Conditional order creation
	if shouldCreateOrder {
		order := Order{
			UserID:   user.ID,
			Product:  "Conditional Product",
			Quantity: 1,
			Total:    199.99,
		}
		
		if err := db.Create(&order).Error; err != nil {
			return err
		}
		
		fmt.Printf("Created conditional order for user %d\n", user.ID)
	}
	
	return nil
}

// Helper functions

func validateUser(user *User) error {
	if user.Name == "" {
		return errors.New("user name is required")
	}
	if user.Email == "" {
		return errors.New("user email is required")
	}
	if user.Age < 0 {
		return errors.New("user age must be non-negative")
	}
	if user.Age > 150 {
		return errors.New("user age must be realistic")
	}
	return nil
}

func simulateExternalAPI(userID uint) error {
	// Simulate network delay
	time.Sleep(10 * time.Millisecond)
	
	// Simulate API failure for certain IDs
	if userID%7 == 0 {
		return errors.New("external API returned error 500")
	}
	
	return nil
}
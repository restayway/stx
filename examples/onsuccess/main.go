package main

import (
	"context"
	"fmt"
	"log"
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

type EventStream struct {
	events []string
}

func (es *EventStream) Emit(event string, data any) {
	es.events = append(es.events, fmt.Sprintf("%s: %v", event, data))
	fmt.Printf("[EVENT] %s: %v\n", event, data)
}

func (es *EventStream) GetEvents() []string {
	return es.events
}

func main() {
	// Open database connection
	db, err := gorm.Open(sqlite.Open("onsuccess_examples.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate the schema
	db.AutoMigrate(&User{})

	// Create context with database
	ctx := stx.New(context.Background(), db)
	
	// Initialize event stream
	eventStream := &EventStream{}

	fmt.Println("=== OnSuccess Examples ===")

	// Example 1: Basic OnSuccess with successful transaction
	fmt.Println("\n1. Basic OnSuccess with successful transaction:")
	if err := exampleBasicSuccess(ctx, eventStream); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Example 2: OnSuccess with transaction rollback (callback should NOT execute)
	fmt.Println("\n2. OnSuccess with transaction rollback:")
	if err := exampleWithRollback(ctx, eventStream); err != nil {
		fmt.Printf("Expected error: %v\n", err)
	}

	// Example 3: OnSuccess without transaction context (immediate execution)
	fmt.Println("\n3. OnSuccess without transaction context:")
	exampleWithoutTransaction(ctx, eventStream)

	// Example 4: Multiple OnSuccess callbacks
	fmt.Println("\n4. Multiple OnSuccess callbacks:")
	if err := exampleMultipleCallbacks(ctx, eventStream); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Example 5: OnSuccess with event streaming pattern
	fmt.Println("\n5. OnSuccess with event streaming pattern:")
	if err := exampleEventStreamingPattern(ctx, eventStream); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Example 6: OnSuccess with complex business logic
	fmt.Println("\n6. OnSuccess with complex business logic:")
	if err := exampleComplexBusinessLogic(ctx, eventStream); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Show all events that were emitted
	fmt.Println("\n=== All Events Emitted ===")
	for i, event := range eventStream.GetEvents() {
		fmt.Printf("%d. %s\n", i+1, event)
	}

	fmt.Println("\n=== OnSuccess Examples Done ===")
}

// Example 1: Basic OnSuccess with successful transaction
func exampleBasicSuccess(ctx context.Context, eventStream *EventStream) (err error) {
	txCtx, cleanup := stx.WithDefer(ctx)
	defer cleanup(&err)

	// Register success callback
	stx.OnSuccess(txCtx, func() {
		fmt.Println("✓ Success callback executed!")
		eventStream.Emit("basic_success", "transaction completed")
	})

	// Create user
	user := User{
		Name:  "Alice Johnson",
		Email: "alice@example.com",
		Age:   30,
	}

	if err := stx.Current(txCtx).Create(&user).Error; err != nil {
		return err
	}

	fmt.Printf("Created user: %s (ID: %d)\n", user.Name, user.ID)
	return nil
}

// Example 2: OnSuccess with transaction rollback
func exampleWithRollback(ctx context.Context, eventStream *EventStream) (err error) {
	txCtx, cleanup := stx.WithDefer(ctx)
	defer cleanup(&err)

	// Register success callback (should NOT execute due to rollback)
	stx.OnSuccess(txCtx, func() {
		fmt.Println("✗ This callback should NOT execute due to rollback!")
		eventStream.Emit("rollback_callback", "this should not happen")
	})

	// Create user
	user := User{
		Name:  "Bob Wilson",
		Email: "bob@example.com",
		Age:   25,
	}

	if err := stx.Current(txCtx).Create(&user).Error; err != nil {
		return err
	}

	fmt.Printf("Created user: %s (ID: %d)\n", user.Name, user.ID)
	
	// Force rollback by returning error
	return fmt.Errorf("forced rollback - callback should not execute")
}

// Example 3: OnSuccess without transaction context
func exampleWithoutTransaction(ctx context.Context, eventStream *EventStream) {
	fmt.Println("Registering OnSuccess without transaction context...")
	
	// This should execute immediately since there's no transaction
	stx.OnSuccess(ctx, func() {
		fmt.Println("✓ Callback executed immediately (no transaction context)")
		eventStream.Emit("immediate_execution", "no transaction context")
	})
	
	fmt.Println("OnSuccess call completed")
}

// Example 4: Multiple OnSuccess callbacks
func exampleMultipleCallbacks(ctx context.Context, eventStream *EventStream) (err error) {
	txCtx, cleanup := stx.WithDefer(ctx)
	defer cleanup(&err)

	// Register multiple success callbacks
	stx.OnSuccess(txCtx, func() {
		fmt.Println("✓ First callback executed")
		eventStream.Emit("callback_1", "first callback")
	})

	stx.OnSuccess(txCtx, func() {
		fmt.Println("✓ Second callback executed")
		eventStream.Emit("callback_2", "second callback")
	})

	stx.OnSuccess(txCtx, func() {
		fmt.Println("✓ Third callback executed")
		eventStream.Emit("callback_3", "third callback")
	})

	// Create user
	user := User{
		Name:  "Charlie Brown",
		Email: "charlie@example.com",
		Age:   28,
	}

	if err := stx.Current(txCtx).Create(&user).Error; err != nil {
		return err
	}

	fmt.Printf("Created user: %s (ID: %d)\n", user.Name, user.ID)
	return nil
}

// Example 5: OnSuccess with event streaming pattern
func exampleEventStreamingPattern(ctx context.Context, eventStream *EventStream) (err error) {
	txCtx, cleanup := stx.WithDefer(ctx)
	defer cleanup(&err)

	// Create user
	user := User{
		Name:  "Diana Prince",
		Email: "diana@example.com",
		Age:   32,
	}

	// Register success callback for user creation event
	stx.OnSuccess(txCtx, func() {
		eventStream.Emit("user_created", map[string]any{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
		})
	})

	// Register success callback for welcome email
	stx.OnSuccess(txCtx, func() {
		eventStream.Emit("send_welcome_email", map[string]any{
			"email": user.Email,
			"name":  user.Name,
		})
	})

	// Register success callback for analytics
	stx.OnSuccess(txCtx, func() {
		eventStream.Emit("track_user_signup", map[string]any{
			"user_id":   user.ID,
			"timestamp": time.Now().Unix(),
		})
	})

	if err := stx.Current(txCtx).Create(&user).Error; err != nil {
		return err
	}

	fmt.Printf("Created user: %s (ID: %d)\n", user.Name, user.ID)
	return nil
}

// Example 6: OnSuccess with complex business logic
func exampleComplexBusinessLogic(ctx context.Context, eventStream *EventStream) (err error) {
	txCtx, cleanup := stx.WithDefer(ctx)
	defer cleanup(&err)

	// Create user
	user := User{
		Name:  "Eve Davis",
		Email: "eve@example.com",
		Age:   35,
	}

	// Register success callback with complex logic
	stx.OnSuccess(txCtx, func() {
		// Simulate complex post-transaction logic
		fmt.Println("✓ Executing complex post-transaction logic...")
		
		// Simulate notification service
		eventStream.Emit("notification_sent", map[string]any{
			"type":    "user_registered",
			"user_id": user.ID,
			"channel": "email",
		})
		
		// Simulate audit log
		eventStream.Emit("audit_log", map[string]any{
			"action":    "user_created",
			"user_id":   user.ID,
			"timestamp": time.Now().Format(time.RFC3339),
		})
		
		// Simulate cache invalidation
		eventStream.Emit("cache_invalidated", map[string]any{
			"keys": []string{"user_list", "user_count"},
		})
		
		fmt.Println("✓ Complex post-transaction logic completed")
	})

	if err := stx.Current(txCtx).Create(&user).Error; err != nil {
		return err
	}

	fmt.Printf("Created user: %s (ID: %d)\n", user.Name, user.ID)
	
	// Simulate some additional business logic
	time.Sleep(10 * time.Millisecond)
	
	return nil
}
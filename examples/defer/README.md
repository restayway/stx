# STX Defer Pattern Examples

This directory contains comprehensive examples demonstrating the powerful defer pattern capabilities of the STX library.

## Overview

The defer pattern in STX provides automatic transaction management with panic recovery and error handling. It's perfect for complex business logic where you need guaranteed cleanup and transaction integrity.

## Basic Pattern

```go
func businessOperation(ctx context.Context) (err error) {
    txCtx, cleanup := stx.WithDefer(ctx)
    defer cleanup(&err)

    db := stx.Current(txCtx)
    
    // Your database operations here
    // - Transaction commits automatically on success
    // - Transaction rolls back on error or panic
    
    return nil
}
```

## Examples

### Basic Examples (`main.go`)

1. **Basic Defer Success** - Simple successful operation with automatic commit
2. **Business Logic Error** - Demonstrates automatic rollback on business errors
3. **Panic Recovery** - Shows how panics are caught and transactions rolled back
4. **Complex Business Transaction** - Multi-step operation with user and order creation
5. **Validation Example** - Input validation with transaction rollback
6. **Multiple Operations** - Batch operations within a single transaction
7. **External API Integration** - Combining database operations with external API calls
8. **Conditional Logic** - Transaction behavior based on runtime conditions

### Advanced Examples (`advanced.go`)

1. **Money Transfer** - Complete financial transaction with balance updates
2. **Concurrent Operations** - Multiple goroutines using defer pattern safely
3. **Nested Operations** - Functions calling other functions within same transaction
4. **Timeout Handling** - Operations with context timeout
5. **Retry Logic** - Automatic retry with exponential backoff

## Key Benefits

### 1. Automatic Transaction Management
- No need to manually call `Commit()` or `Rollback()`
- Transactions are automatically committed on success
- Automatic rollback on errors or panics

### 2. Panic Recovery
- Panics are automatically caught and converted to errors
- Transactions are safely rolled back even during panics
- No risk of leaving transactions in inconsistent state

### 3. Error Handling
- Simple error propagation with the `err` parameter
- Structured error messages with context
- Wrapping of underlying errors for better debugging

### 4. Clean Code
- Eliminates boilerplate transaction management code
- Makes business logic more readable
- Reduces chance of forgetting to handle edge cases

## Running the Examples

```bash
# Run basic examples
go run main.go

# Run advanced examples
go run main.go advanced.go
```

## Common Patterns

### 1. Simple Operation
```go
func createUser(ctx context.Context, name, email string) (err error) {
    txCtx, cleanup := stx.WithDefer(ctx)
    defer cleanup(&err)

    db := stx.Current(txCtx)
    
    user := User{Name: name, Email: email}
    return db.Create(&user).Error
}
```

### 2. Multi-Step Operation
```go
func createUserWithAccount(ctx context.Context, name, email string) (err error) {
    txCtx, cleanup := stx.WithDefer(ctx)
    defer cleanup(&err)

    db := stx.Current(txCtx)
    
    // Step 1: Create user
    user := User{Name: name, Email: email}
    if err := db.Create(&user).Error; err != nil {
        return err
    }
    
    // Step 2: Create account
    account := Account{UserID: user.ID, Balance: 0}
    if err := db.Create(&account).Error; err != nil {
        return err
    }
    
    // Both operations commit together or roll back together
    return nil
}
```

### 3. With Business Logic Validation
```go
func transferMoney(ctx context.Context, fromID, toID uint, amount float64) (err error) {
    txCtx, cleanup := stx.WithDefer(ctx)
    defer cleanup(&err)

    db := stx.Current(txCtx)
    
    // Get accounts
    var fromAccount, toAccount Account
    if err := db.First(&fromAccount, fromID).Error; err != nil {
        return err
    }
    if err := db.First(&toAccount, toID).Error; err != nil {
        return err
    }
    
    // Validate business rules
    if fromAccount.Balance < amount {
        return errors.New("insufficient funds")
    }
    
    // Update balances
    fromAccount.Balance -= amount
    toAccount.Balance += amount
    
    if err := db.Save(&fromAccount).Error; err != nil {
        return err
    }
    if err := db.Save(&toAccount).Error; err != nil {
        return err
    }
    
    return nil
}
```

### 4. With External API Calls
```go
func createUserWithNotification(ctx context.Context, name, email string) (err error) {
    txCtx, cleanup := stx.WithDefer(ctx)
    defer cleanup(&err)

    db := stx.Current(txCtx)
    
    // Create user in database
    user := User{Name: name, Email: email}
    if err := db.Create(&user).Error; err != nil {
        return err
    }
    
    // Call external API
    if err := sendWelcomeEmail(user.Email); err != nil {
        return err // This will rollback the user creation
    }
    
    return nil
}
```

## Error Handling

The defer pattern provides rich error handling capabilities:

```go
func handleErrors(ctx context.Context) (err error) {
    txCtx, cleanup := stx.WithDefer(ctx)
    defer cleanup(&err)

    db := stx.Current(txCtx)
    
    // Database errors are automatically handled
    if err := db.Create(&User{}).Error; err != nil {
        return err // Automatic rollback
    }
    
    // Business logic errors
    if someCondition {
        return errors.New("business rule violated") // Automatic rollback
    }
    
    // Panics are automatically recovered
    if panicCondition {
        panic("something went wrong") // Automatic rollback + error conversion
    }
    
    return nil // Automatic commit
}
```

## Best Practices

1. **Always use the error parameter**: `defer cleanup(&err)`
2. **Keep transactions short**: Minimize time between begin and commit
3. **Handle external dependencies carefully**: External API failures should rollback DB changes
4. **Use meaningful error messages**: Help with debugging and monitoring
5. **Test error scenarios**: Make sure rollback behavior works as expected
6. **Consider timeouts**: Use context timeouts for long-running operations

## Performance Considerations

- Defer pattern has minimal overhead compared to manual transaction management
- Automatic panic recovery adds slight performance cost but improves reliability
- Use connection pooling for better performance in concurrent scenarios
- Consider batch operations for multiple similar database operations

## Migration from Manual Transaction Management

Before (manual):
```go
func oldWay(ctx context.Context) error {
    txCtx := stx.Begin(ctx)
    defer func() {
        if r := recover(); r != nil {
            stx.Rollback(txCtx)
            panic(r)
        }
    }()
    
    db := stx.Current(txCtx)
    
    if err := db.Create(&User{}).Error; err != nil {
        stx.Rollback(txCtx)
        return err
    }
    
    return stx.Commit(txCtx)
}
```

After (defer pattern):
```go
func newWay(ctx context.Context) (err error) {
    txCtx, cleanup := stx.WithDefer(ctx)
    defer cleanup(&err)
    
    db := stx.Current(txCtx)
    
    return db.Create(&User{}).Error
}
```

The defer pattern significantly reduces boilerplate code while improving error handling and transaction safety.
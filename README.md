# STX - Context-based GORM Transaction Manager

[![CI](https://github.com/restayway/stx/actions/workflows/ci.yml/badge.svg)](https://github.com/restayway/stx/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/restayway/stx)](https://goreportcard.com/report/github.com/restayway/stx)
[![codecov](https://codecov.io/gh/restayway/stx/branch/main/graph/badge.svg)](https://codecov.io/gh/restayway/stx)
[![GoDoc](https://godoc.org/github.com/restayway/stx?status.svg)](https://godoc.org/github.com/restayway/stx)

STX is a lightweight Go package that provides context-based transaction management for GORM. It allows you to seamlessly integrate database transactions with Go's context package, making it easy to pass database connections and transactions through your application layers.

> state (ctx) + transaction = stx

## Features

- **Context-based transaction management**: Embed GORM database instances in Go contexts
- **Thread-safe operations**: Concurrent access to database connections is handled safely
- **Flexible transaction control**: Support for both automatic and manual transaction management
- **Graceful error handling**: Commit/Rollback operations return nil when no transaction is active
- **GORM integration**: Works seamlessly with existing GORM code
- **Zero dependencies**: Only depends on GORM and standard Go libraries
- **100% test coverage**: Comprehensive test suite with full coverage

## Installation

```bash
go get github.com/restayway/stx
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/restayway/stx"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

type User struct {
    ID   uint   `gorm:"primaryKey"`
    Name string `gorm:"not null"`
}

func main() {
    db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
    if err != nil {
        log.Fatal(err)
    }
    
    // Create context with database
    ctx := stx.New(context.Background(), db)
    
    // Use the context to get database connection
    db = stx.GetCurrent(ctx)
    
    // Auto-migrate
    db.AutoMigrate(&User{})
    
    // Create user
    user := User{Name: "John Doe"}
    db.Create(&user)
}
```

### Transaction Management

```go
// Automatic transaction management
err := stx.WithTransaction(ctx, func(txCtx context.Context) error {
    txDB := stx.GetCurrent(txCtx)
    
    // All operations within this function are in a transaction
    user := User{Name: "Jane Doe"}
    if err := txDB.Create(&user).Error; err != nil {
        return err // Transaction will be rolled back
    }
    
    // Transaction will be committed if no error is returned
    return nil
})
```

### Manual Transaction Control

```go
// Begin transaction
txCtx := stx.Begin(ctx)
txDB := stx.GetCurrent(txCtx)

// Perform operations
user := User{Name: "Bob Smith"}
if err := txDB.Create(&user).Error; err != nil {
    stx.Rollback(txCtx)
    return err
}

// Commit transaction
if err := stx.Commit(txCtx); err != nil {
    return err
}
```

### Check Transaction Status

```go
if stx.IsTransaction(ctx) {
    log.Println("Currently in a transaction")
} else {
    log.Println("Not in a transaction")
}
```

## API Reference

### Functions

#### `New(ctx context.Context, db *gorm.DB) context.Context`

Creates a new context with the given GORM database instance.

#### `GetCurrent(ctx context.Context) *gorm.DB`

Retrieves the current GORM database instance from the context. Returns nil if no database is found.

#### `WithTransaction(ctx context.Context, fn func(context.Context) error, opts ...*sql.TxOptions) error`

Executes the given function within a database transaction. The transaction is automatically committed if the function returns nil, or rolled back if it returns an error.

#### `Begin(ctx context.Context, opts ...*sql.TxOptions) context.Context`

Begins a new database transaction and returns a new context with the transaction.

#### `Commit(ctx context.Context) error`

Commits the current transaction. Returns `nil` if no transaction is active (operations were performed directly without transactions).

#### `Rollback(ctx context.Context) error`

Rolls back the current transaction. Returns `nil` if no transaction is active (operations were performed directly without transactions).

#### `IsTransaction(ctx context.Context) bool`

Returns true if the current context contains an active transaction.

## Graceful Error Handling

STX provides graceful error handling for transaction operations:

- **Commit/Rollback without transaction**: When `Commit()` or `Rollback()` is called on a context without an active transaction, they return `nil` instead of an error. This design acknowledges that operations were performed directly without transactions.
- **Safe operation**: This behavior allows for more flexible code patterns where you can safely call commit/rollback operations without checking if a transaction is active.

```go
// Safe to call even if no transaction is active
ctx := stx.New(context.Background(), db)
err := stx.Commit(ctx) // Returns nil, no error
```

## Testing

Run the test suite:

```bash
# Run tests
make test

# Run tests with coverage
make cover

# Run all checks (format, lint, test)
make check
```

## Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [GORM](https://gorm.io/) for the excellent ORM library
- The Go community for inspiration and best practices

package stx

import (
	"context"
	"database/sql"
	"errors"
	"sync"

	"gorm.io/gorm"
)

type contextKey string

const txContextKey contextKey = "stx:tx"

type STX struct {
	mu        sync.RWMutex
	db        *gorm.DB
	callbacks []func()
}

// STXError represents an error with additional context
type STXError struct {
	Message string
	Err     error
}

func (e *STXError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *STXError) Unwrap() error {
	return e.Err
}

// newSTXError creates a new STX error
func newSTXError(message string, err error) *STXError {
	return &STXError{Message: message, Err: err}
}

// panicError creates an error for panic recovery
func panicError(v any) error {
	if err, ok := v.(error); ok {
		return newSTXError("recovered from panic", err)
	}
	if str, ok := v.(string); ok {
		return newSTXError("recovered from panic", errors.New(str))
	}
	return errors.New("recovered from panic")
}

func New(ctx context.Context, db *gorm.DB) context.Context {
	return context.WithValue(ctx, txContextKey, &STX{db: db})
}

func Current(ctx context.Context) *gorm.DB {
	if ctx == nil {
		return nil
	}

	val := ctx.Value(txContextKey)
	if val == nil {
		return nil
	}

	stx, ok := val.(*STX)
	if !ok || stx == nil {
		return nil
	}

	stx.mu.RLock()
	defer stx.mu.RUnlock()
	return stx.db
}

// GetCurrent is deprecated, use Current instead
func GetCurrent(ctx context.Context) *gorm.DB {
	return Current(ctx)
}

func WithTransaction(ctx context.Context, fn func(context.Context) error, opts ...*sql.TxOptions) error {
	db := Current(ctx)
	if db == nil {
		return gorm.ErrInvalidTransaction
	}

	return db.Transaction(func(tx *gorm.DB) error {
		newCtx := context.WithValue(ctx, txContextKey, &STX{db: tx})
		err := fn(newCtx)
		
		// Execute success callbacks if no error occurred
		if err == nil {
			if val := newCtx.Value(txContextKey); val != nil {
				if stx, ok := val.(*STX); ok && stx != nil {
					stx.mu.RLock()
					callbacks := make([]func(), len(stx.callbacks))
					copy(callbacks, stx.callbacks)
					stx.mu.RUnlock()
					
					for _, callback := range callbacks {
						if callback != nil {
							callback()
						}
					}
				}
			}
		}
		
		return err
	}, opts...)
}

// OnSuccess registers a callback to execute when the transaction successfully commits.
// If the context does not contain a transaction, the callback executes immediately.
// This is useful for triggering events, notifications, or other side effects after
// successful database operations.
//
// Example usage:
//   stx.OnSuccess(ctx, func() {
//       fmt.Println("Transaction completed successfully!")
//   })
//
// For event streaming:
//   stx.OnSuccess(ctx, func() {
//       eventStream.Emit("user_created", userID)
//   })
func OnSuccess(ctx context.Context, callback func()) {
	if ctx == nil || callback == nil {
		return
	}

	val := ctx.Value(txContextKey)
	if val == nil {
		// No transaction context, execute immediately
		callback()
		return
	}

	stx, ok := val.(*STX)
	if !ok || stx == nil {
		// Invalid transaction context, execute immediately
		callback()
		return
	}

	// Add callback to be executed on successful commit
	stx.mu.Lock()
	stx.callbacks = append(stx.callbacks, callback)
	stx.mu.Unlock()
}

func Begin(ctx context.Context, opts ...*sql.TxOptions) context.Context {
	db := Current(ctx)
	if db == nil {
		return ctx
	}

	tx := db.Begin(opts...)
	return context.WithValue(ctx, txContextKey, &STX{db: tx})
}

func Commit(ctx context.Context) error {
	db := Current(ctx)
	if db == nil {
		return nil
	}

	// Only commit if we're actually in a transaction
	if !IsTx(ctx) {
		return nil
	}

	return db.Commit().Error
}

func Rollback(ctx context.Context) error {
	db := Current(ctx)
	if db == nil {
		return nil
	}

	// Only rollback if we're actually in a transaction
	if !IsTx(ctx) {
		return nil
	}

	return db.Rollback().Error
}

func IsTx(ctx context.Context) bool {
	db := Current(ctx)
	if db == nil {
		return false
	}

	return db.Statement.ConnPool != nil &&
		db.Statement.ConnPool != db.Statement.DB.ConnPool
}

// IsTransaction is deprecated, use IsTx instead
func IsTransaction(ctx context.Context) bool {
	return IsTx(ctx)
}

// WithDefer begins a transaction and returns a context and cleanup function.
// The cleanup function should be called with defer and handles panic recovery
// and automatic commit/rollback based on the error state.
//
// Success callbacks registered with OnSuccess will be executed after a successful
// commit, making this ideal for triggering events, notifications, or other side
// effects that should only occur when the transaction is successfully persisted.
//
// Example usage:
//   func createUser(ctx context.Context, user *User) (err error) {
//       txCtx, cleanup := stx.WithDefer(ctx)
//       defer cleanup(&err)
//
//       // Register success callback for event streaming
//       stx.OnSuccess(txCtx, func() {
//           eventStream.Emit("user_created", user.ID)
//       })
//
//       // Perform database operations
//       return stx.Current(txCtx).Create(user).Error
//   }
func WithDefer(ctx context.Context, opts ...*sql.TxOptions) (context.Context, func(*error)) {
	txCtx := Begin(ctx, opts...)
	
	cleanup := func(err *error) {
		if r := recover(); r != nil {
			Rollback(txCtx)
			if err != nil {
				*err = panicError(r)
			}
			return
		}
		
		if err != nil && *err != nil {
			Rollback(txCtx)
			return
		}
		
		if commitErr := Commit(txCtx); commitErr != nil {
			if err != nil {
				*err = newSTXError("failed to commit transaction", commitErr)
			}
			return
		}
		
		// Execute success callbacks after successful commit
		if txCtx != nil {
			if val := txCtx.Value(txContextKey); val != nil {
				if stx, ok := val.(*STX); ok && stx != nil {
					stx.mu.RLock()
					callbacks := make([]func(), len(stx.callbacks))
					copy(callbacks, stx.callbacks)
					stx.mu.RUnlock()
					
					for _, callback := range callbacks {
						if callback != nil {
							callback()
						}
					}
				}
			}
		}
	}
	
	return txCtx, cleanup
}

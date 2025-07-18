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
	mu sync.RWMutex
	db *gorm.DB
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
		return fn(newCtx)
	}, opts...)
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
		}
	}
	
	return txCtx, cleanup
}

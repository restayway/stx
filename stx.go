package stx

import (
	"context"
	"database/sql"
	"sync"

	"gorm.io/gorm"
)

type contextKey string

const txContextKey contextKey = "stx:tx"

type STX struct {
	mu sync.RWMutex
	db *gorm.DB
}

func New(ctx context.Context, db *gorm.DB) context.Context {
	return context.WithValue(ctx, txContextKey, &STX{db: db})
}

func GetCurrent(ctx context.Context) *gorm.DB {
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

func WithTransaction(ctx context.Context, fn func(context.Context) error, opts ...*sql.TxOptions) error {
	db := GetCurrent(ctx)
	if db == nil {
		return gorm.ErrInvalidTransaction
	}

	return db.Transaction(func(tx *gorm.DB) error {
		newCtx := context.WithValue(ctx, txContextKey, &STX{db: tx})
		return fn(newCtx)
	}, opts...)
}

func Begin(ctx context.Context, opts ...*sql.TxOptions) context.Context {
	db := GetCurrent(ctx)
	if db == nil {
		return ctx
	}

	tx := db.Begin(opts...)
	return context.WithValue(ctx, txContextKey, &STX{db: tx})
}

func Commit(ctx context.Context) error {
	db := GetCurrent(ctx)
	if db == nil {
		return nil
	}

	return db.Commit().Error
}

func Rollback(ctx context.Context) error {
	db := GetCurrent(ctx)
	if db == nil {
		return nil
	}

	return db.Rollback().Error
}

func IsTransaction(ctx context.Context) bool {
	db := GetCurrent(ctx)
	if db == nil {
		return false
	}

	return db.Statement.ConnPool != nil &&
		db.Statement.ConnPool != db.Statement.DB.ConnPool
}

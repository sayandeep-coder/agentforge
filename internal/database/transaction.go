package database

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// WithinTransaction executes fn in a transaction and rolls back when fn returns an error.
func WithinTransaction(ctx context.Context, db *gorm.DB, fn func(*gorm.DB) error) error {
	if err := db.WithContext(ctx).Transaction(fn); err != nil {
		return fmt.Errorf("transaction: %w", err)
	}
	return nil
}

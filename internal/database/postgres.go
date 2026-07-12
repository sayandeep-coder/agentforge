// Package database owns PostgreSQL connection setup and lifecycle.
package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/agentforge/agentforge/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Connection contains the GORM handle and its underlying SQL pool.
type Connection struct {
	DB   *gorm.DB
	Pool *sql.DB
}

// Open creates a PostgreSQL-backed GORM connection and verifies reachability.
func Open(ctx context.Context, cfg config.DatabaseConfig) (*Connection, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("database URL is required")
	}

	db, err := gorm.Open(postgres.Open(cfg.URL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	pool, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get database pool: %w", err)
	}
	pool.SetMaxOpenConns(cfg.MaxOpen)
	pool.SetMaxIdleConns(cfg.MaxIdle)
	pool.SetConnMaxLifetime(cfg.MaxLifetime)

	pingContext, cancel := context.WithTimeout(ctx, cfg.PingTimeout)
	defer cancel()
	if err := pool.PingContext(pingContext); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Connection{DB: db, Pool: pool}, nil
}

// Close releases the underlying PostgreSQL connection pool.
func (c *Connection) Close() error {
	if c == nil || c.Pool == nil {
		return nil
	}
	return c.Pool.Close()
}

// Package config loads and validates AgentForge runtime configuration.
package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

const (
	defaultEnvironment       = "development"
	defaultServerPort        = 8080
	defaultShutdownTimeout   = 10 * time.Second
	defaultDatabaseMaxOpen   = 25
	defaultDatabaseMaxIdle   = 10
	defaultDatabaseMaxLife   = time.Hour
	defaultDatabasePingLimit = 5 * time.Second
	defaultWorkerCount       = 4
	defaultQueueCapacity     = 1000
)

// Config contains configuration required by the AgentForge process.
type Config struct {
	App      AppConfig
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Log      LogConfig
	Worker   WorkerConfig
}

// AppConfig contains application-level settings.
type AppConfig struct {
	Environment string
}

// ServerConfig contains HTTP server settings.
type ServerConfig struct {
	Port            int
	ShutdownTimeout time.Duration
}

// DatabaseConfig contains PostgreSQL connection settings.
type DatabaseConfig struct {
	URL         string
	MaxOpen     int
	MaxIdle     int
	MaxLifetime time.Duration
	PingTimeout time.Duration
}

// RedisConfig contains Redis connection settings reserved for queue and cache adapters.
type RedisConfig struct {
	URL string
}

// LogConfig contains structured logging settings.
type LogConfig struct {
	Level string
}

// WorkerConfig contains execution pool settings.
type WorkerConfig struct {
	Count         int
	QueueCapacity int
}

// Load reads .env when present, overlays process environment variables, and validates values.
func Load() (Config, error) {
	_ = godotenv.Load()

	settings := viper.New()
	settings.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	settings.AutomaticEnv()
	settings.SetDefault("app.environment", defaultEnvironment)
	settings.SetDefault("server.port", defaultServerPort)
	settings.SetDefault("server.shutdown_timeout", defaultShutdownTimeout)
	settings.SetDefault("database.max_open", defaultDatabaseMaxOpen)
	settings.SetDefault("database.max_idle", defaultDatabaseMaxIdle)
	settings.SetDefault("database.max_lifetime", defaultDatabaseMaxLife)
	settings.SetDefault("database.ping_timeout", defaultDatabasePingLimit)
	settings.SetDefault("log.level", "info")
	settings.SetDefault("worker.count", defaultWorkerCount)
	settings.SetDefault("worker.queue_capacity", defaultQueueCapacity)

	config := Config{
		App: AppConfig{
			Environment: settings.GetString("app.environment"),
		},
		Server: ServerConfig{
			Port:            settings.GetInt("server.port"),
			ShutdownTimeout: settings.GetDuration("server.shutdown_timeout"),
		},
		Database: DatabaseConfig{
			URL:         settings.GetString("database.url"),
			MaxOpen:     settings.GetInt("database.max_open"),
			MaxIdle:     settings.GetInt("database.max_idle"),
			MaxLifetime: settings.GetDuration("database.max_lifetime"),
			PingTimeout: settings.GetDuration("database.ping_timeout"),
		},
		Redis: RedisConfig{
			URL: settings.GetString("redis.url"),
		},
		Log: LogConfig{
			Level: settings.GetString("log.level"),
		},
		Worker: WorkerConfig{
			Count:         settings.GetInt("worker.count"),
			QueueCapacity: settings.GetInt("worker.queue_capacity"),
		},
	}

	if err := config.Validate(); err != nil {
		return Config{}, err
	}

	return config, nil
}

// Validate checks values that are required for safe process startup.
func (c Config) Validate() error {
	if c.App.Environment == "" {
		return errors.New("app environment must not be empty")
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535: %d", c.Server.Port)
	}
	if c.Server.ShutdownTimeout <= 0 {
		return errors.New("server shutdown timeout must be positive")
	}
	if c.Database.MaxOpen < 1 || c.Database.MaxIdle < 0 || c.Database.MaxIdle > c.Database.MaxOpen {
		return fmt.Errorf("invalid database pool limits: max_open=%d max_idle=%d", c.Database.MaxOpen, c.Database.MaxIdle)
	}
	if c.Database.MaxLifetime <= 0 || c.Database.PingTimeout <= 0 {
		return errors.New("database durations must be positive")
	}
	if c.Log.Level == "" {
		return errors.New("log level must not be empty")
	}
	if c.Worker.Count < 1 || c.Worker.QueueCapacity < 1 {
		return errors.New("worker count and queue capacity must be positive")
	}
	return nil
}

// Address returns the TCP address used by the HTTP server.
func (c Config) Address() string {
	return fmt.Sprintf(":%d", c.Server.Port)
}

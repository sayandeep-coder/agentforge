package config

import (
	"strings"
	"testing"
	"time"
)

func validConfig() Config {
	return Config{
		App: AppConfig{Environment: "test"},
		Server: ServerConfig{
			Port:            8080,
			ShutdownTimeout: time.Second,
		},
		Database: DatabaseConfig{
			MaxOpen:     2,
			MaxIdle:     1,
			MaxLifetime: time.Minute,
			PingTimeout: time.Second,
		},
		Log:    LogConfig{Level: "info"},
		Worker: WorkerConfig{Count: 1, QueueCapacity: 1},
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*Config)
		wantError string
	}{
		{name: "valid", mutate: func(*Config) {}},
		{name: "invalid port", mutate: func(c *Config) { c.Server.Port = 0 }, wantError: "server port"},
		{name: "invalid pool", mutate: func(c *Config) { c.Database.MaxIdle = 3 }, wantError: "database pool"},
		{name: "missing environment", mutate: func(c *Config) { c.App.Environment = "" }, wantError: "environment"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := validConfig()
			test.mutate(&cfg)
			err := cfg.Validate()
			if test.wantError == "" && err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if test.wantError != "" && (err == nil || !strings.Contains(err.Error(), test.wantError)) {
				t.Fatalf("Validate() error = %v, want substring %q", err, test.wantError)
			}
		})
	}
}

func TestConfigAddress(t *testing.T) {
	cfg := validConfig()
	if got, want := cfg.Address(), ":8080"; got != want {
		t.Fatalf("Address() = %q, want %q", got, want)
	}
}

// Package logger provides the process-wide logging implementation through dependency injection.
package logger

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New constructs a structured logger for the requested environment and level.
func New(environment, level string) (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	if strings.EqualFold(environment, "development") {
		config = zap.NewDevelopmentConfig()
	}

	parsedLevel, err := zapcore.ParseLevel(strings.ToLower(level))
	if err != nil {
		return nil, err
	}
	config.Level = zap.NewAtomicLevelAt(parsedLevel)
	return config.Build()
}

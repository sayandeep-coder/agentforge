// Package service contains application use cases and business validation.
package service

import "errors"

var (
	// ErrInvalidInput indicates that a command violates application validation rules.
	ErrInvalidInput = errors.New("service: invalid input")
	// ErrNotFound indicates that a requested domain resource does not exist.
	ErrNotFound = errors.New("service: resource not found")
	// ErrConflict indicates that a state transition cannot be applied.
	ErrConflict = errors.New("service: conflict")
)

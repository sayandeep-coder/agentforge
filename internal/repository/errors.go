// Package repository defines persistence ports and their GORM adapters.
package repository

import "errors"

var (
	// ErrNotFound indicates that the requested record does not exist.
	ErrNotFound = errors.New("repository: record not found")
	// ErrConflict indicates that a persistence uniqueness or state constraint was violated.
	ErrConflict = errors.New("repository: conflict")
)

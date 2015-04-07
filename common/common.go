package common

import "errors"

var (
	// Used for NotFound errors
	ErrNotFound = errors.New("not found")
	// Used for AlreadyExists errors
	ErrAlreadyExists = errors.New("already exists")
)

// Check if an error is a NotFound
func IsNotFound(e error) bool {
	return e == ErrNotFound
}

// Check if an error is a AlreadyExists
func IsAlreadyExist(e error) bool {
	return e == ErrAlreadyExists
}

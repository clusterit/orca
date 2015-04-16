package common

import (
	"fmt"

	"github.com/satori/go.uuid"
)

import "errors"

const OrcaPrefix = "orca"

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

// Generate a UUID string
func GenerateUUID() string {
	return uuid.NewV4().String()
}

func NetworkUser(n, u string) string {
	return fmt.Sprintf("%s@%s", u, n)
}

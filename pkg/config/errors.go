package config

import (
	"errors"
	"fmt"
)

type ValidationError struct {
	Field   string
	Message string
}

type ValidationErrors struct {
	Errors []ValidationError
}

func (e *ValidationErrors) Error() string {
	if len(e.Errors) == 0 {
		return "no error"
	}

	msg := "validation failed:\n"
	for i, _ := range e.Errors {
		err := e.Errors[i]
		msg += fmt.Sprintf("  %d. %s: %s\n", i+1, err.Field, err.Message)
	}

	return msg

}

func (e *ValidationErrors) Add(field, message string) {
	e.Errors = append(e.Errors, ValidationError{Field: field, Message: message})
}

func (e *ValidationErrors) HasErrors() bool {
	return len(e.Errors) > 0
}

var ErrConfigNotFound = fmt.Errorf("config not found")
var ErrInvalidDNSName = errors.New("name must be DNS-1123 compliant (lowercase letters, numbers, and hyphens only)")
var ErrInvalidPort = errors.New("port must be between 1 and 65535")
var ErrInvalidReplicas = errors.New("replicas must be at least 1")

type ErrConfigLoadFailed struct {
	Path  string
	Cause error
}

func (e *ErrConfigLoadFailed) Error() string {
	return fmt.Sprintf("failed to load config from %s: %v", e.Path, e.Cause)
}

func (e *ErrConfigLoadFailed) Unwrap() error {
	return e.Cause
}

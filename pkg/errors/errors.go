package errors

import (
	"fmt"
)

// KudevError is the interface for all kudev errors.
type KudevError interface {
	error

	// ExitCode returns the shell exit code.
	ExitCode() int

	// UserMessage returns a user-friendly message.
	UserMessage() string

	// SuggestedAction returns a helpful suggestion.
	SuggestedAction() string
}

// Exit codes
const (
	ExitGeneral  = 1 // General error
	ExitConfig   = 2 // Configuration error
	ExitKubeAuth = 3 // Kubernetes authentication error
	ExitBuild    = 4 // Build error
	ExitDeploy   = 5 // Deployment error
	ExitWatch    = 6 // Watch error
)

// ConfigError represents configuration-related errors.
type ConfigError struct {
	Message    string
	Suggestion string
	Cause      error
}

func (e *ConfigError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *ConfigError) ExitCode() int           { return ExitConfig }
func (e *ConfigError) UserMessage() string     { return e.Message }
func (e *ConfigError) SuggestedAction() string { return e.Suggestion }
func (e *ConfigError) Unwrap() error           { return e.Cause }

// KubeAuthError represents Kubernetes authentication errors.
type KubeAuthError struct {
	Message    string
	Suggestion string
	Cause      error
}

func (e *KubeAuthError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *KubeAuthError) ExitCode() int           { return ExitKubeAuth }
func (e *KubeAuthError) UserMessage() string     { return e.Message }
func (e *KubeAuthError) SuggestedAction() string { return e.Suggestion }
func (e *KubeAuthError) Unwrap() error           { return e.Cause }

// BuildError represents image build errors.
type BuildError struct {
	Message    string
	Suggestion string
	Cause      error
}

func (e *BuildError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *BuildError) ExitCode() int           { return ExitBuild }
func (e *BuildError) UserMessage() string     { return e.Message }
func (e *BuildError) SuggestedAction() string { return e.Suggestion }
func (e *BuildError) Unwrap() error           { return e.Cause }

// DeployError represents Kubernetes deployment errors.
type DeployError struct {
	Message    string
	Suggestion string
	Cause      error
}

func (e *DeployError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *DeployError) ExitCode() int           { return ExitDeploy }
func (e *DeployError) UserMessage() string     { return e.Message }
func (e *DeployError) SuggestedAction() string { return e.Suggestion }
func (e *DeployError) Unwrap() error           { return e.Cause }

// WatchError represents file watching errors.
type WatchError struct {
	Message    string
	Suggestion string
	Cause      error
}

func (e *WatchError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *WatchError) ExitCode() int           { return ExitWatch }
func (e *WatchError) UserMessage() string     { return e.Message }
func (e *WatchError) SuggestedAction() string { return e.Suggestion }
func (e *WatchError) Unwrap() error           { return e.Cause }

// Ensure all types implement KudevError
var (
	_ KudevError = (*ConfigError)(nil)
	_ KudevError = (*KubeAuthError)(nil)
	_ KudevError = (*BuildError)(nil)
	_ KudevError = (*DeployError)(nil)
	_ KudevError = (*WatchError)(nil)
)

// Package errors provides custom error types for dex operations.
//
// This package defines domain-specific error types that provide rich context
// for debugging and user-friendly error messages. All error types that wrap
// underlying errors implement the Unwrap method for use with errors.Is and
// errors.As from the standard library.
//
// Error types include:
//   - ConfigError: Configuration file parsing errors with location info
//   - RegistryError: Registry operation failures
//   - InstallError: Plugin installation failures with phase information
//   - ValidationError: Resource validation failures
//   - NotFoundError: Resource not found errors
//   - VersionError: Version constraint resolution failures
package errors

import (
	"errors"
	"fmt"
	"strings"
)

// ConfigError represents an error in configuration parsing.
// It includes file location information to help users identify
// the exact location of the problem.
type ConfigError struct {
	File    string // Path to the config file
	Line    int    // Line number (0 if unknown)
	Column  int    // Column number (0 if unknown)
	Message string // Error description
	Err     error  // Underlying error
}

// Error returns a human-readable error message with file location.
func (e *ConfigError) Error() string {
	var location string
	if e.Line > 0 {
		if e.Column > 0 {
			location = fmt.Sprintf("%s:%d:%d", e.File, e.Line, e.Column)
		} else {
			location = fmt.Sprintf("%s:%d", e.File, e.Line)
		}
	} else {
		location = e.File
	}

	if e.Err != nil {
		return fmt.Sprintf("config error at %s: %s: %v", location, e.Message, e.Err)
	}
	return fmt.Sprintf("config error at %s: %s", location, e.Message)
}

// Unwrap returns the underlying error for use with errors.Is and errors.As.
func (e *ConfigError) Unwrap() error {
	return e.Err
}

// RegistryError represents an error with a registry operation.
// It includes the registry URL and the operation that failed.
type RegistryError struct {
	URL string // Registry URL
	Op  string // Operation: "fetch", "resolve", "list", "connect"
	Err error  // Underlying error
}

// Error returns a human-readable error message describing the registry failure.
func (e *RegistryError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("registry error: %s failed for %s: %v", e.Op, e.URL, e.Err)
	}
	return fmt.Sprintf("registry error: %s failed for %s", e.Op, e.URL)
}

// Unwrap returns the underlying error for use with errors.Is and errors.As.
func (e *RegistryError) Unwrap() error {
	return e.Err
}

// InstallError represents an error during plugin installation.
// It includes the plugin name and the installation phase where the error occurred.
type InstallError struct {
	Plugin string // Plugin name
	Phase  string // Phase: "fetch", "parse", "validate", "install", "merge"
	Err    error  // Underlying error
}

// Error returns a human-readable error message describing the installation failure.
func (e *InstallError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("install error for %s during %s: %v", e.Plugin, e.Phase, e.Err)
	}
	return fmt.Sprintf("install error for %s during %s", e.Plugin, e.Phase)
}

// Unwrap returns the underlying error for use with errors.Is and errors.As.
func (e *InstallError) Unwrap() error {
	return e.Err
}

// ValidationError represents a validation error for a resource.
// It includes the resource identifier, the field that failed validation,
// and a message describing the validation failure.
type ValidationError struct {
	Resource string // Resource type and name (e.g., "plugin:my-plugin")
	Field    string // Field that failed validation
	Message  string // Validation error message
}

// Error returns a human-readable error message describing the validation failure.
func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error for %s: field %q: %s", e.Resource, e.Field, e.Message)
	}
	return fmt.Sprintf("validation error for %s: %s", e.Resource, e.Message)
}

// NotFoundError represents a not found error.
// It is used when a requested resource (plugin, file, registry, etc.) cannot be found.
type NotFoundError struct {
	What string // What wasn't found (e.g., "plugin", "file", "registry")
	Name string // Name of the thing
}

// Error returns a human-readable error message describing what was not found.
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.What, e.Name)
}

// VersionError represents a version resolution error.
// It is used when a version constraint cannot be satisfied by any available version.
type VersionError struct {
	Plugin     string   // Plugin name
	Constraint string   // Version constraint that couldn't be satisfied
	Available  []string // Available versions
	Message    string   // Additional context message
}

// Error returns a human-readable error message describing the version resolution failure.
func (e *VersionError) Error() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("version error for %s: ", e.Plugin))

	if e.Message != "" {
		sb.WriteString(e.Message)
	} else {
		sb.WriteString(fmt.Sprintf("constraint %q cannot be satisfied", e.Constraint))
	}

	if len(e.Available) > 0 {
		sb.WriteString(fmt.Sprintf(" (available: %s)", strings.Join(e.Available, ", ")))
	}

	return sb.String()
}

// NewConfigError creates a new ConfigError with the given parameters.
// Use line=0 and col=0 if the location is unknown.
func NewConfigError(file string, line, col int, msg string, err error) *ConfigError {
	return &ConfigError{
		File:    file,
		Line:    line,
		Column:  col,
		Message: msg,
		Err:     err,
	}
}

// NewRegistryError creates a new RegistryError with the given parameters.
// Common operations are: "fetch", "resolve", "list", "connect".
func NewRegistryError(url, op string, err error) *RegistryError {
	return &RegistryError{
		URL: url,
		Op:  op,
		Err: err,
	}
}

// NewInstallError creates a new InstallError with the given parameters.
// Common phases are: "fetch", "parse", "validate", "install", "merge".
func NewInstallError(plugin, phase string, err error) *InstallError {
	return &InstallError{
		Plugin: plugin,
		Phase:  phase,
		Err:    err,
	}
}

// NewValidationError creates a new ValidationError with the given parameters.
// Use an empty field string if the error applies to the resource as a whole.
func NewValidationError(resource, field, message string) *ValidationError {
	return &ValidationError{
		Resource: resource,
		Field:    field,
		Message:  message,
	}
}

// NewNotFoundError creates a new NotFoundError with the given parameters.
// Common values for what: "plugin", "file", "registry", "version", "command".
func NewNotFoundError(what, name string) *NotFoundError {
	return &NotFoundError{
		What: what,
		Name: name,
	}
}

// NewVersionError creates a new VersionError with the given parameters.
// The available slice may be nil or empty if available versions are unknown.
func NewVersionError(plugin, constraint string, available []string, msg string) *VersionError {
	return &VersionError{
		Plugin:     plugin,
		Constraint: constraint,
		Available:  available,
		Message:    msg,
	}
}

// Re-export standard library error functions for convenience.
// This allows callers to use errors.Is, errors.As, etc. without
// importing both this package and the standard errors package.
var (
	// Is reports whether any error in err's tree matches target.
	Is = errors.Is
	// As finds the first error in err's tree that matches target.
	As = errors.As
	// New returns an error that formats as the given text.
	New = errors.New
	// Join returns an error that wraps the given errors.
	Join = errors.Join
	// Unwrap returns the result of calling the Unwrap method on err.
	Unwrap = errors.Unwrap
)

// Wrap wraps an error with additional context message.
// If err is nil, Wrap returns nil.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

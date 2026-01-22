package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ConfigError
		expected string
	}{
		{
			name: "with line and column",
			err: &ConfigError{
				File:    "config.hcl",
				Line:    10,
				Column:  5,
				Message: "invalid syntax",
				Err:     nil,
			},
			expected: "config error at config.hcl:10:5: invalid syntax",
		},
		{
			name: "with line only",
			err: &ConfigError{
				File:    "config.hcl",
				Line:    10,
				Column:  0,
				Message: "invalid syntax",
				Err:     nil,
			},
			expected: "config error at config.hcl:10: invalid syntax",
		},
		{
			name: "file only",
			err: &ConfigError{
				File:    "config.hcl",
				Line:    0,
				Column:  0,
				Message: "file not found",
				Err:     nil,
			},
			expected: "config error at config.hcl: file not found",
		},
		{
			name: "with wrapped error",
			err: &ConfigError{
				File:    "config.hcl",
				Line:    10,
				Column:  5,
				Message: "parsing failed",
				Err:     errors.New("unexpected token"),
			},
			expected: "config error at config.hcl:10:5: parsing failed: unexpected token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestConfigError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := &ConfigError{
		File:    "config.hcl",
		Line:    1,
		Column:  1,
		Message: "test",
		Err:     underlying,
	}

	assert.Equal(t, underlying, err.Unwrap())

	// Test with nil wrapped error
	errNoWrap := &ConfigError{
		File:    "config.hcl",
		Message: "test",
	}
	assert.Nil(t, errNoWrap.Unwrap())
}

func TestRegistryError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *RegistryError
		expected string
	}{
		{
			name: "with underlying error",
			err: &RegistryError{
				URL: "https://registry.example.com",
				Op:  "fetch",
				Err: errors.New("connection refused"),
			},
			expected: "registry error: fetch failed for https://registry.example.com: connection refused",
		},
		{
			name: "without underlying error",
			err: &RegistryError{
				URL: "https://registry.example.com",
				Op:  "list",
				Err: nil,
			},
			expected: "registry error: list failed for https://registry.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestRegistryError_Unwrap(t *testing.T) {
	underlying := errors.New("network error")
	err := &RegistryError{
		URL: "https://registry.example.com",
		Op:  "connect",
		Err: underlying,
	}

	assert.Equal(t, underlying, err.Unwrap())
}

func TestInstallError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *InstallError
		expected string
	}{
		{
			name: "with underlying error",
			err: &InstallError{
				Plugin: "my-plugin",
				Phase:  "validate",
				Err:    errors.New("missing required field"),
			},
			expected: "install error for my-plugin during validate: missing required field",
		},
		{
			name: "without underlying error",
			err: &InstallError{
				Plugin: "my-plugin",
				Phase:  "fetch",
				Err:    nil,
			},
			expected: "install error for my-plugin during fetch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestInstallError_Unwrap(t *testing.T) {
	underlying := errors.New("permission denied")
	err := &InstallError{
		Plugin: "my-plugin",
		Phase:  "install",
		Err:    underlying,
	}

	assert.Equal(t, underlying, err.Unwrap())
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ValidationError
		expected string
	}{
		{
			name: "with field",
			err: &ValidationError{
				Resource: "plugin:my-plugin",
				Field:    "name",
				Message:  "cannot be empty",
			},
			expected: "validation error for plugin:my-plugin: field \"name\": cannot be empty",
		},
		{
			name: "without field",
			err: &ValidationError{
				Resource: "plugin:my-plugin",
				Field:    "",
				Message:  "invalid configuration",
			},
			expected: "validation error for plugin:my-plugin: invalid configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestNotFoundError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *NotFoundError
		expected string
	}{
		{
			name: "plugin not found",
			err: &NotFoundError{
				What: "plugin",
				Name: "my-plugin",
			},
			expected: "plugin not found: my-plugin",
		},
		{
			name: "file not found",
			err: &NotFoundError{
				What: "file",
				Name: "/path/to/file.txt",
			},
			expected: "file not found: /path/to/file.txt",
		},
		{
			name: "registry not found",
			err: &NotFoundError{
				What: "registry",
				Name: "local",
			},
			expected: "registry not found: local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestVersionError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *VersionError
		expected string
	}{
		{
			name: "with constraint and available",
			err: &VersionError{
				Plugin:     "my-plugin",
				Constraint: "^2.0.0",
				Available:  []string{"1.0.0", "1.5.0"},
				Message:    "",
			},
			expected: "version error for my-plugin: constraint \"^2.0.0\" cannot be satisfied (available: 1.0.0, 1.5.0)",
		},
		{
			name: "with custom message",
			err: &VersionError{
				Plugin:     "my-plugin",
				Constraint: "^2.0.0",
				Available:  []string{"1.0.0"},
				Message:    "no compatible version found",
			},
			expected: "version error for my-plugin: no compatible version found (available: 1.0.0)",
		},
		{
			name: "without available versions",
			err: &VersionError{
				Plugin:     "my-plugin",
				Constraint: "^2.0.0",
				Available:  nil,
				Message:    "",
			},
			expected: "version error for my-plugin: constraint \"^2.0.0\" cannot be satisfied",
		},
		{
			name: "with empty available list",
			err: &VersionError{
				Plugin:     "my-plugin",
				Constraint: "^2.0.0",
				Available:  []string{},
				Message:    "no versions available",
			},
			expected: "version error for my-plugin: no versions available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		message  string
		expected string
	}{
		{
			name:     "wrap error",
			err:      errors.New("original error"),
			message:  "additional context",
			expected: "additional context: original error",
		},
		{
			name:     "wrap nil",
			err:      nil,
			message:  "should be nil",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Wrap(tt.err, tt.message)
			if tt.err == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result.Error())
			}
		})
	}
}

func TestWrap_Unwrap(t *testing.T) {
	original := errors.New("original error")
	wrapped := Wrap(original, "context")

	// Should be able to unwrap to get original
	unwrapped := errors.Unwrap(wrapped)
	assert.Equal(t, original, unwrapped)
}

func TestErrorsIs(t *testing.T) {
	originalErr := errors.New("original")
	wrappedErr := Wrap(originalErr, "wrapped")

	assert.True(t, Is(wrappedErr, originalErr))
	assert.False(t, Is(wrappedErr, errors.New("different")))
}

func TestErrorsAs(t *testing.T) {
	configErr := &ConfigError{
		File:    "test.hcl",
		Message: "test error",
	}
	wrappedErr := Wrap(configErr, "wrapped")

	var target *ConfigError
	assert.True(t, As(wrappedErr, &target))
	assert.Equal(t, "test.hcl", target.File)
	assert.Equal(t, "test error", target.Message)

	var notFoundTarget *NotFoundError
	assert.False(t, As(wrappedErr, &notFoundTarget))
}

func TestNewConfigError(t *testing.T) {
	underlying := errors.New("parse error")
	err := NewConfigError("config.hcl", 10, 5, "invalid syntax", underlying)

	assert.Equal(t, "config.hcl", err.File)
	assert.Equal(t, 10, err.Line)
	assert.Equal(t, 5, err.Column)
	assert.Equal(t, "invalid syntax", err.Message)
	assert.Equal(t, underlying, err.Err)
}

func TestNewRegistryError(t *testing.T) {
	underlying := errors.New("network error")
	err := NewRegistryError("https://registry.example.com", "fetch", underlying)

	assert.Equal(t, "https://registry.example.com", err.URL)
	assert.Equal(t, "fetch", err.Op)
	assert.Equal(t, underlying, err.Err)
}

func TestNewInstallError(t *testing.T) {
	underlying := errors.New("validation failed")
	err := NewInstallError("my-plugin", "validate", underlying)

	assert.Equal(t, "my-plugin", err.Plugin)
	assert.Equal(t, "validate", err.Phase)
	assert.Equal(t, underlying, err.Err)
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("plugin:my-plugin", "name", "cannot be empty")

	assert.Equal(t, "plugin:my-plugin", err.Resource)
	assert.Equal(t, "name", err.Field)
	assert.Equal(t, "cannot be empty", err.Message)
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("plugin", "my-plugin")

	assert.Equal(t, "plugin", err.What)
	assert.Equal(t, "my-plugin", err.Name)
}

func TestNewVersionError(t *testing.T) {
	err := NewVersionError("my-plugin", "^2.0.0", []string{"1.0.0", "1.5.0"}, "custom message")

	assert.Equal(t, "my-plugin", err.Plugin)
	assert.Equal(t, "^2.0.0", err.Constraint)
	assert.Equal(t, []string{"1.0.0", "1.5.0"}, err.Available)
	assert.Equal(t, "custom message", err.Message)
}

func TestNewVersionError_NilAvailable(t *testing.T) {
	err := NewVersionError("my-plugin", "^2.0.0", nil, "")

	assert.Equal(t, "my-plugin", err.Plugin)
	assert.Nil(t, err.Available)
}

func TestExportedFunctions(t *testing.T) {
	// Test that re-exported functions work correctly
	err1 := New("test error")
	assert.Equal(t, "test error", err1.Error())

	err2 := errors.New("other error")
	joined := Join(err1, err2)
	assert.True(t, Is(joined, err1))
	assert.True(t, Is(joined, err2))

	wrapped := Wrap(err1, "context")
	unwrapped := Unwrap(wrapped)
	assert.Equal(t, err1, unwrapped)
}

func TestConfigError_ErrorChaining(t *testing.T) {
	// Test that error chaining works correctly with ConfigError
	innerErr := errors.New("inner error")
	configErr := NewConfigError("config.hcl", 1, 1, "outer error", innerErr)
	wrappedErr := Wrap(configErr, "top level")

	// Should be able to find the ConfigError
	var target *ConfigError
	assert.True(t, As(wrappedErr, &target))
	assert.Equal(t, "config.hcl", target.File)

	// Should be able to find the inner error
	assert.True(t, Is(wrappedErr, innerErr))
}

func TestRegistryError_ErrorChaining(t *testing.T) {
	innerErr := errors.New("connection failed")
	registryErr := NewRegistryError("https://example.com", "connect", innerErr)

	// Should be able to unwrap to inner error
	assert.True(t, Is(registryErr, innerErr))

	// Should be able to cast to RegistryError
	var target *RegistryError
	assert.True(t, As(registryErr, &target))
	assert.Equal(t, "connect", target.Op)
}

func TestInstallError_ErrorChaining(t *testing.T) {
	innerErr := errors.New("permission denied")
	installErr := NewInstallError("my-plugin", "install", innerErr)

	// Should be able to unwrap to inner error
	assert.True(t, Is(installErr, innerErr))

	// Should be able to cast to InstallError
	var target *InstallError
	assert.True(t, As(installErr, &target))
	assert.Equal(t, "install", target.Phase)
}

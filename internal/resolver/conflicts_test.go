package resolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConflict_Error(t *testing.T) {
	c := &Conflict{
		Package:    "core-lib",
		Required:   []string{"app-a requires core-lib@^2.0.0", "app-b requires core-lib@^1.0.0"},
		Available:  []string{"1.0.0", "1.5.0", "2.0.0"},
		Resolution: "Update app-b to support core-lib@^2.0.0",
	}

	errMsg := c.Error()
	assert.Contains(t, errMsg, "version conflict for \"core-lib\"")
	assert.Contains(t, errMsg, "Required by:")
	assert.Contains(t, errMsg, "app-a requires core-lib@^2.0.0")
	assert.Contains(t, errMsg, "app-b requires core-lib@^1.0.0")
	assert.Contains(t, errMsg, "Available versions: 1.0.0, 1.5.0, 2.0.0")
	assert.Contains(t, errMsg, "Suggestion: Update app-b")
}

func TestConflict_Error_NoAvailable(t *testing.T) {
	c := &Conflict{
		Package:  "missing-lib",
		Required: []string{"app requires missing-lib@^1.0.0"},
	}

	errMsg := c.Error()
	assert.Contains(t, errMsg, "version conflict for \"missing-lib\"")
	assert.Contains(t, errMsg, "app requires missing-lib@^1.0.0")
	assert.NotContains(t, errMsg, "Available versions")
}

func TestConflictError_Error(t *testing.T) {
	err := &ConflictError{
		Conflicts: []*Conflict{
			{
				Package:  "lib-a",
				Required: []string{"app requires lib-a@^1.0.0"},
			},
			{
				Package:  "lib-b",
				Required: []string{"app requires lib-b@^2.0.0"},
			},
		},
	}

	errMsg := err.Error()
	assert.Contains(t, errMsg, "Cannot resolve dependencies")
	assert.Contains(t, errMsg, "lib-a")
	assert.Contains(t, errMsg, "lib-b")
}

func TestConflictError_Error_Empty(t *testing.T) {
	err := &ConflictError{}
	assert.Equal(t, "unknown conflict", err.Error())
}

func TestVersionNotFoundError_Error(t *testing.T) {
	err := &VersionNotFoundError{
		Package:    "my-lib",
		Constraint: "^3.0.0",
		Available:  []string{"1.0.0", "2.0.0"},
	}

	errMsg := err.Error()
	assert.Contains(t, errMsg, "my-lib")
	assert.Contains(t, errMsg, "^3.0.0")
	assert.Contains(t, errMsg, "1.0.0, 2.0.0")
}

func TestVersionNotFoundError_Error_NoAvailable(t *testing.T) {
	err := &VersionNotFoundError{
		Package:    "my-lib",
		Constraint: "^1.0.0",
	}

	errMsg := err.Error()
	assert.Contains(t, errMsg, "no versions found")
	assert.Contains(t, errMsg, "my-lib")
	assert.Contains(t, errMsg, "^1.0.0")
}

func TestPackageNotFoundError_Error(t *testing.T) {
	err := &PackageNotFoundError{
		Package:  "missing-pkg",
		Registry: "my-registry",
	}

	errMsg := err.Error()
	assert.Contains(t, errMsg, "missing-pkg")
	assert.Contains(t, errMsg, "my-registry")
}

func TestPackageNotFoundError_Error_NoRegistry(t *testing.T) {
	err := &PackageNotFoundError{
		Package: "missing-pkg",
	}

	errMsg := err.Error()
	assert.Contains(t, errMsg, "missing-pkg")
	assert.Contains(t, errMsg, "not found")
	assert.NotContains(t, errMsg, "registry")
}

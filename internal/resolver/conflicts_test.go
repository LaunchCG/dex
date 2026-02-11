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
	expected := "version conflict for \"core-lib\":\n" +
		"  Required by:\n" +
		"    - app-a requires core-lib@^2.0.0\n" +
		"    - app-b requires core-lib@^1.0.0\n" +
		"  Available versions: 1.0.0, 1.5.0, 2.0.0\n" +
		"  Suggestion: Update app-b to support core-lib@^2.0.0\n"
	assert.Equal(t, expected, errMsg)
}

func TestConflict_Error_NoAvailable(t *testing.T) {
	c := &Conflict{
		Package:  "missing-lib",
		Required: []string{"app requires missing-lib@^1.0.0"},
	}

	errMsg := c.Error()
	expected := "version conflict for \"missing-lib\":\n" +
		"  Required by:\n" +
		"    - app requires missing-lib@^1.0.0\n"
	assert.Equal(t, expected, errMsg)
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
	expected := "Cannot resolve dependencies\n\n" +
		"version conflict for \"lib-a\":\n" +
		"  Required by:\n" +
		"    - app requires lib-a@^1.0.0\n" +
		"\n" +
		"version conflict for \"lib-b\":\n" +
		"  Required by:\n" +
		"    - app requires lib-b@^2.0.0\n"
	assert.Equal(t, expected, errMsg)
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
	assert.Equal(t, `no version of "my-lib" matches constraint "^3.0.0" (available: 1.0.0, 2.0.0)`, errMsg)
}

func TestVersionNotFoundError_Error_NoAvailable(t *testing.T) {
	err := &VersionNotFoundError{
		Package:    "my-lib",
		Constraint: "^1.0.0",
	}

	errMsg := err.Error()
	assert.Equal(t, `no versions found for package "my-lib" matching constraint "^1.0.0"`, errMsg)
}

func TestPackageNotFoundError_Error(t *testing.T) {
	err := &PackageNotFoundError{
		Package:  "missing-pkg",
		Registry: "my-registry",
	}

	errMsg := err.Error()
	assert.Equal(t, `package "missing-pkg" not found in registry "my-registry"`, errMsg)
}

func TestPackageNotFoundError_Error_NoRegistry(t *testing.T) {
	err := &PackageNotFoundError{
		Package: "missing-pkg",
	}

	errMsg := err.Error()
	assert.Equal(t, `package "missing-pkg" not found`, errMsg)
}

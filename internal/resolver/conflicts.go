package resolver

import (
	"fmt"
	"strings"
)

// Conflict represents a version conflict between dependencies.
type Conflict struct {
	// Package is the package with conflicting requirements
	Package string

	// Required lists the packages and their constraints that require this package
	// Format: "pkg@constraint"
	Required []string

	// Available lists the available versions of the package
	Available []string

	// Resolution is a hint about how to resolve the conflict
	Resolution string
}

// Error returns a human-readable error message for the conflict.
func (c *Conflict) Error() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("version conflict for %q:\n", c.Package))
	sb.WriteString("  Required by:\n")
	for _, req := range c.Required {
		sb.WriteString(fmt.Sprintf("    - %s\n", req))
	}
	if len(c.Available) > 0 {
		sb.WriteString(fmt.Sprintf("  Available versions: %s\n", strings.Join(c.Available, ", ")))
	}
	if c.Resolution != "" {
		sb.WriteString(fmt.Sprintf("  Suggestion: %s\n", c.Resolution))
	}
	return sb.String()
}

// ConflictError wraps multiple conflicts for reporting.
type ConflictError struct {
	Conflicts []*Conflict
}

// Error returns a formatted error message for all conflicts.
func (e *ConflictError) Error() string {
	if len(e.Conflicts) == 0 {
		return "unknown conflict"
	}

	var sb strings.Builder
	sb.WriteString("Cannot resolve dependencies\n\n")
	for i, c := range e.Conflicts {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(c.Error())
	}
	return sb.String()
}

// VersionNotFoundError indicates no version satisfies the constraint.
type VersionNotFoundError struct {
	Package    string
	Constraint string
	Available  []string
}

func (e *VersionNotFoundError) Error() string {
	if len(e.Available) == 0 {
		return fmt.Sprintf("no versions found for package %q matching constraint %q",
			e.Package, e.Constraint)
	}
	return fmt.Sprintf("no version of %q matches constraint %q (available: %s)",
		e.Package, e.Constraint, strings.Join(e.Available, ", "))
}

// PackageNotFoundError indicates a package doesn't exist in any registry.
type PackageNotFoundError struct {
	Package  string
	Registry string
}

func (e *PackageNotFoundError) Error() string {
	if e.Registry != "" {
		return fmt.Sprintf("package %q not found in registry %q", e.Package, e.Registry)
	}
	return fmt.Sprintf("package %q not found", e.Package)
}

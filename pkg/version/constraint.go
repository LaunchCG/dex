package version

import (
	"fmt"
	"regexp"
	"strings"
)

// constraintRegex matches constraint operators and version numbers.
// Supported operators: =, >, <, >=, <=, ~, ^
// Note: >= and <= must come before > and < in the alternation to match correctly.
var constraintRegex = regexp.MustCompile(`^(>=|<=|[=><~^])?(.+)$`)

// Constraint represents a version constraint that can be matched against
// versions to determine compatibility.
//
// Supported constraint formats:
//   - "1.2.3"     - Exact version match
//   - "=1.2.3"    - Explicit exact version match
//   - "^1.2.3"    - Caret: compatible with 1.x.x (>=1.2.3, <2.0.0)
//   - "~1.2.3"    - Tilde: compatible with 1.2.x (>=1.2.3, <1.3.0)
//   - ">=1.2.3"   - Greater than or equal
//   - ">1.2.3"    - Greater than
//   - "<=1.2.3"   - Less than or equal
//   - "<1.2.3"    - Less than
//   - "latest"    - Special: matches any version
type Constraint struct {
	Original string  // Original constraint string for display
	checks   []check // Internal checks to evaluate
}

// check represents a single version check within a constraint.
type check struct {
	op      string   // Operator: "=", ">", "<", ">=", "<=", "~", "^", "latest"
	version *Version // Version to compare against (nil for "latest")
}

// ParseConstraint parses a version constraint string into a Constraint.
//
// Supported formats:
//   - "1.2.3"     - Exact version match
//   - "=1.2.3"    - Explicit exact version match
//   - "^1.2.3"    - Caret: compatible with 1.x.x (>=1.2.3, <2.0.0)
//   - "~1.2.3"    - Tilde: compatible with 1.2.x (>=1.2.3, <1.3.0)
//   - ">=1.2.3"   - Greater than or equal to 1.2.3
//   - ">1.2.3"    - Greater than 1.2.3
//   - "<=1.2.3"   - Less than or equal to 1.2.3
//   - "<1.2.3"    - Less than 1.2.3
//   - "latest"    - Matches any version
//
// The caret (^) constraint follows npm/Cargo conventions:
//   - ^1.2.3  allows >=1.2.3 and <2.0.0 (major version fixed)
//   - ^0.2.3  allows >=0.2.3 and <0.3.0 (minor version fixed when major is 0)
//   - ^0.0.3  allows >=0.0.3 and <0.0.4 (patch version fixed when major.minor is 0.0)
//
// The tilde (~) constraint:
//   - ~1.2.3  allows >=1.2.3 and <1.3.0 (patch-level changes only)
//
// Returns an error if the constraint string is invalid.
func ParseConstraint(s string) (*Constraint, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("constraint string cannot be empty")
	}

	c := &Constraint{
		Original: s,
	}

	// Handle special "latest" constraint
	if strings.ToLower(s) == "latest" {
		c.checks = []check{{op: "latest", version: nil}}
		return c, nil
	}

	// Parse the constraint with optional operator
	matches := constraintRegex.FindStringSubmatch(s)
	if matches == nil {
		return nil, fmt.Errorf("invalid constraint format: %q", s)
	}

	op := matches[1]
	versionStr := matches[2]

	// Default operator is exact match
	if op == "" {
		op = "="
	}

	// Parse the version
	v, err := Parse(versionStr)
	if err != nil {
		return nil, fmt.Errorf("invalid version in constraint %q: %w", s, err)
	}

	// Build the checks based on the operator
	switch op {
	case "=":
		c.checks = []check{{op: "=", version: v}}
	case ">":
		c.checks = []check{{op: ">", version: v}}
	case "<":
		c.checks = []check{{op: "<", version: v}}
	case ">=":
		c.checks = []check{{op: ">=", version: v}}
	case "<=":
		c.checks = []check{{op: "<=", version: v}}
	case "~":
		// Tilde: >=version, <next minor
		c.checks = []check{
			{op: ">=", version: v},
			{op: "<", version: &Version{Major: v.Major, Minor: v.Minor + 1, Patch: 0}},
		}
	case "^":
		// Caret: compatible versions (npm/Cargo style)
		c.checks = buildCaretChecks(v)
	default:
		return nil, fmt.Errorf("unknown operator %q in constraint %q", op, s)
	}

	return c, nil
}

// buildCaretChecks builds the checks for a caret (^) constraint.
//
// Caret constraints follow npm/Cargo conventions:
//   - ^1.2.3 -> >=1.2.3, <2.0.0 (major version fixed)
//   - ^0.2.3 -> >=0.2.3, <0.3.0 (minor version fixed when major is 0)
//   - ^0.0.3 -> >=0.0.3, <0.0.4 (patch version fixed when major.minor is 0.0)
func buildCaretChecks(v *Version) []check {
	checks := []check{{op: ">=", version: v}}

	if v.Major != 0 {
		// ^1.2.3 -> <2.0.0
		checks = append(checks, check{
			op:      "<",
			version: &Version{Major: v.Major + 1, Minor: 0, Patch: 0},
		})
	} else if v.Minor != 0 {
		// ^0.2.3 -> <0.3.0
		checks = append(checks, check{
			op:      "<",
			version: &Version{Major: 0, Minor: v.Minor + 1, Patch: 0},
		})
	} else {
		// ^0.0.3 -> <0.0.4
		checks = append(checks, check{
			op:      "<",
			version: &Version{Major: 0, Minor: 0, Patch: v.Patch + 1},
		})
	}

	return checks
}

// Match returns true if the given version satisfies this constraint.
//
// A version must pass all internal checks to match the constraint.
// The "latest" constraint matches any version.
//
// Examples:
//
//	constraint, _ := ParseConstraint("^1.2.0")
//	constraint.Match(MustParse("1.5.0"))  // true
//	constraint.Match(MustParse("2.0.0"))  // false
func (c *Constraint) Match(v *Version) bool {
	if v == nil {
		return false
	}

	for _, chk := range c.checks {
		if !chk.match(v) {
			return false
		}
	}

	return true
}

// match evaluates a single check against a version.
func (chk *check) match(v *Version) bool {
	switch chk.op {
	case "latest":
		return true
	case "=":
		return v.Equal(chk.version)
	case ">":
		return v.GreaterThan(chk.version)
	case "<":
		return v.LessThan(chk.version)
	case ">=":
		return v.GreaterThan(chk.version) || v.Equal(chk.version)
	case "<=":
		return v.LessThan(chk.version) || v.Equal(chk.version)
	default:
		return false
	}
}

// String returns the original constraint string.
func (c *Constraint) String() string {
	return c.Original
}

// FindBest finds the best (highest) matching version from a list of versions.
//
// Returns the highest version that satisfies the constraint, or nil if no
// version matches.
//
// Example:
//
//	constraint, _ := ParseConstraint("^1.0.0")
//	versions := []*Version{
//	    MustParse("0.9.0"),
//	    MustParse("1.0.0"),
//	    MustParse("1.5.0"),
//	    MustParse("2.0.0"),
//	}
//	best := constraint.FindBest(versions) // Returns 1.5.0
func (c *Constraint) FindBest(versions []*Version) *Version {
	if len(versions) == 0 {
		return nil
	}

	// Filter to matching versions
	var matching []*Version
	for _, v := range versions {
		if c.Match(v) {
			matching = append(matching, v)
		}
	}

	if len(matching) == 0 {
		return nil
	}

	// Find the highest matching version
	best := matching[0]
	for _, v := range matching[1:] {
		if v.GreaterThan(best) {
			best = v
		}
	}

	return best
}

// MustParse is like Parse but panics if the version string is invalid.
//
// This is useful for initializing package-level version variables or in tests.
// Do not use this with untrusted input.
func MustParse(s string) *Version {
	v, err := Parse(s)
	if err != nil {
		panic(fmt.Sprintf("version.MustParse(%q): %v", s, err))
	}
	return v
}

// MustParseConstraint is like ParseConstraint but panics if the constraint
// string is invalid.
//
// This is useful for initializing package-level constraint variables or in tests.
// Do not use this with untrusted input.
func MustParseConstraint(s string) *Constraint {
	c, err := ParseConstraint(s)
	if err != nil {
		panic(fmt.Sprintf("version.MustParseConstraint(%q): %v", s, err))
	}
	return c
}

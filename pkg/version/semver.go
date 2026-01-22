// Package version provides semantic versioning utilities for parsing,
// comparing, and constraining software versions.
package version

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// semverRegex matches semantic version strings with optional v prefix,
// prerelease, and build metadata.
// Examples: "1.2.3", "v1.2.3", "1.2.3-alpha", "1.2.3-beta.1+build.123"
var semverRegex = regexp.MustCompile(`^v?(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)

// Version represents a semantic version (major.minor.patch) with optional
// prerelease and build metadata.
//
// Semantic versioning follows the specification at https://semver.org/
// A version has the format: MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]
//
// Examples:
//   - 1.0.0
//   - 2.1.3-alpha
//   - 3.0.0-beta.1+build.123
type Version struct {
	Major      int    // Major version (breaking changes)
	Minor      int    // Minor version (new features, backwards compatible)
	Patch      int    // Patch version (bug fixes, backwards compatible)
	Prerelease string // Prerelease identifier (e.g., "alpha", "beta.1")
	Build      string // Build metadata (e.g., "build.123")
}

// Parse parses a version string into a Version struct.
//
// Supported formats:
//   - "1.2.3"              - Standard semver
//   - "v1.2.3"             - With 'v' prefix (common in git tags)
//   - "1.2"                - Minor version only (patch defaults to 0)
//   - "1"                  - Major version only (minor and patch default to 0)
//   - "1.2.3-beta.1"       - With prerelease
//   - "1.2.3+build.123"    - With build metadata
//   - "1.2.3-beta.1+build" - With both prerelease and build
//
// Returns an error if the string is not a valid semantic version.
func Parse(s string) (*Version, error) {
	if s == "" {
		return nil, fmt.Errorf("version string cannot be empty")
	}

	matches := semverRegex.FindStringSubmatch(s)
	if matches == nil {
		return nil, fmt.Errorf("invalid version format: %q", s)
	}

	v := &Version{}

	// Parse major version (always present if regex matched)
	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, fmt.Errorf("invalid major version: %s", matches[1])
	}
	v.Major = major

	// Parse minor version (optional, defaults to 0)
	if matches[2] != "" {
		minor, err := strconv.Atoi(matches[2])
		if err != nil {
			return nil, fmt.Errorf("invalid minor version: %s", matches[2])
		}
		v.Minor = minor
	}

	// Parse patch version (optional, defaults to 0)
	if matches[3] != "" {
		patch, err := strconv.Atoi(matches[3])
		if err != nil {
			return nil, fmt.Errorf("invalid patch version: %s", matches[3])
		}
		v.Patch = patch
	}

	// Prerelease (optional)
	v.Prerelease = matches[4]

	// Build metadata (optional)
	v.Build = matches[5]

	return v, nil
}

// String returns the version as a string without the 'v' prefix.
//
// The format is: MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]
//
// Examples:
//   - "1.2.3"
//   - "1.2.3-alpha"
//   - "1.2.3-beta.1+build.123"
func (v *Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)

	if v.Prerelease != "" {
		s += "-" + v.Prerelease
	}

	if v.Build != "" {
		s += "+" + v.Build
	}

	return s
}

// Compare compares two versions according to semantic versioning rules.
//
// Returns:
//   - -1 if v < other
//   - 0 if v == other
//   - 1 if v > other
//
// Comparison rules:
//  1. Major, minor, and patch versions are compared numerically
//  2. A version with a prerelease has lower precedence than the same version without
//     (e.g., 1.0.0-alpha < 1.0.0)
//  3. Prerelease versions are compared by splitting on '.' and comparing each identifier
//  4. Build metadata is ignored in comparisons per semver spec
func (v *Version) Compare(other *Version) int {
	// Compare major version
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}

	// Compare minor version
	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}

	// Compare patch version
	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}

	// Compare prerelease
	// Per semver: a version without prerelease > version with prerelease
	// e.g., 1.0.0 > 1.0.0-alpha
	if v.Prerelease == "" && other.Prerelease != "" {
		return 1
	}
	if v.Prerelease != "" && other.Prerelease == "" {
		return -1
	}
	if v.Prerelease != other.Prerelease {
		return comparePrereleases(v.Prerelease, other.Prerelease)
	}

	// Build metadata is ignored in comparison per semver spec
	return 0
}

// comparePrereleases compares two prerelease strings according to semver rules.
//
// Rules from semver spec:
//  1. Split by '.' to get identifiers
//  2. Numeric identifiers are compared as integers
//  3. Alphanumeric identifiers are compared lexically
//  4. Numeric identifiers always have lower precedence than alphanumeric
//  5. A larger set of identifiers has higher precedence if all preceding are equal
func comparePrereleases(a, b string) int {
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")

	minLen := len(partsA)
	if len(partsB) < minLen {
		minLen = len(partsB)
	}

	for i := 0; i < minLen; i++ {
		cmp := comparePrereleaseIdentifier(partsA[i], partsB[i])
		if cmp != 0 {
			return cmp
		}
	}

	// If all compared parts are equal, the longer one has higher precedence
	if len(partsA) < len(partsB) {
		return -1
	}
	if len(partsA) > len(partsB) {
		return 1
	}

	return 0
}

// comparePrereleaseIdentifier compares two individual prerelease identifiers.
func comparePrereleaseIdentifier(a, b string) int {
	// Try to parse both as integers
	aInt, aErr := strconv.Atoi(a)
	bInt, bErr := strconv.Atoi(b)

	// Both are numeric
	if aErr == nil && bErr == nil {
		if aInt < bInt {
			return -1
		}
		if aInt > bInt {
			return 1
		}
		return 0
	}

	// Numeric identifiers have lower precedence than alphanumeric
	if aErr == nil && bErr != nil {
		return -1
	}
	if aErr != nil && bErr == nil {
		return 1
	}

	// Both are alphanumeric - compare lexically
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// LessThan returns true if v < other.
//
// This is a convenience method equivalent to v.Compare(other) == -1.
func (v *Version) LessThan(other *Version) bool {
	return v.Compare(other) < 0
}

// GreaterThan returns true if v > other.
//
// This is a convenience method equivalent to v.Compare(other) == 1.
func (v *Version) GreaterThan(other *Version) bool {
	return v.Compare(other) > 0
}

// Equal returns true if v == other.
//
// This is a convenience method equivalent to v.Compare(other) == 0.
// Note that build metadata is ignored in equality checks per semver spec.
func (v *Version) Equal(other *Version) bool {
	return v.Compare(other) == 0
}

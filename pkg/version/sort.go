package version

import "sort"

// Sort sorts a slice of versions in ascending order (oldest first).
//
// The sort is performed in-place and follows semantic versioning comparison
// rules. Versions with prereleases are sorted before their release counterparts
// (e.g., 1.0.0-alpha < 1.0.0).
//
// Example:
//
//	versions := []*Version{
//	    MustParse("2.0.0"),
//	    MustParse("1.0.0"),
//	    MustParse("1.5.0"),
//	}
//	Sort(versions)
//	// Result: [1.0.0, 1.5.0, 2.0.0]
func Sort(versions []*Version) {
	sort.Sort(versionSlice(versions))
}

// SortDesc sorts a slice of versions in descending order (newest first).
//
// The sort is performed in-place and follows semantic versioning comparison
// rules. This is useful for displaying available versions with the most
// recent at the top.
//
// Example:
//
//	versions := []*Version{
//	    MustParse("1.0.0"),
//	    MustParse("2.0.0"),
//	    MustParse("1.5.0"),
//	}
//	SortDesc(versions)
//	// Result: [2.0.0, 1.5.0, 1.0.0]
func SortDesc(versions []*Version) {
	sort.Sort(sort.Reverse(versionSlice(versions)))
}

// versionSlice implements sort.Interface for sorting versions in ascending order.
type versionSlice []*Version

// Len returns the number of versions in the slice.
func (vs versionSlice) Len() int {
	return len(vs)
}

// Less reports whether the version at index i is less than the version at index j.
func (vs versionSlice) Less(i, j int) bool {
	return vs[i].LessThan(vs[j])
}

// Swap swaps the versions at indices i and j.
func (vs versionSlice) Swap(i, j int) {
	vs[i], vs[j] = vs[j], vs[i]
}

// SortStrings parses and sorts version strings in ascending order.
//
// Invalid version strings are silently excluded from the result.
// If you need error handling, parse the versions individually with Parse().
//
// Example:
//
//	sorted := SortStrings([]string{"2.0.0", "1.0.0", "invalid", "1.5.0"})
//	// Result: []*Version{1.0.0, 1.5.0, 2.0.0}
func SortStrings(versionStrings []string) []*Version {
	var versions []*Version
	for _, s := range versionStrings {
		if v, err := Parse(s); err == nil {
			versions = append(versions, v)
		}
	}
	Sort(versions)
	return versions
}

// SortStringsDesc parses and sorts version strings in descending order.
//
// Invalid version strings are silently excluded from the result.
// If you need error handling, parse the versions individually with Parse().
//
// Example:
//
//	sorted := SortStringsDesc([]string{"1.0.0", "2.0.0", "invalid", "1.5.0"})
//	// Result: []*Version{2.0.0, 1.5.0, 1.0.0}
func SortStringsDesc(versionStrings []string) []*Version {
	var versions []*Version
	for _, s := range versionStrings {
		if v, err := Parse(s); err == nil {
			versions = append(versions, v)
		}
	}
	SortDesc(versions)
	return versions
}

// Latest returns the highest version from a slice of versions.
//
// Returns nil if the slice is empty.
//
// Example:
//
//	latest := Latest([]*Version{
//	    MustParse("1.0.0"),
//	    MustParse("2.0.0"),
//	    MustParse("1.5.0"),
//	})
//	// Result: 2.0.0
func Latest(versions []*Version) *Version {
	if len(versions) == 0 {
		return nil
	}

	latest := versions[0]
	for _, v := range versions[1:] {
		if v.GreaterThan(latest) {
			latest = v
		}
	}
	return latest
}

// Oldest returns the lowest version from a slice of versions.
//
// Returns nil if the slice is empty.
//
// Example:
//
//	oldest := Oldest([]*Version{
//	    MustParse("2.0.0"),
//	    MustParse("1.0.0"),
//	    MustParse("1.5.0"),
//	})
//	// Result: 1.0.0
func Oldest(versions []*Version) *Version {
	if len(versions) == 0 {
		return nil
	}

	oldest := versions[0]
	for _, v := range versions[1:] {
		if v.LessThan(oldest) {
			oldest = v
		}
	}
	return oldest
}

"""Tests for dex.utils.version module."""

import pytest

from dex.utils.version import (
    SemVer,
    VersionRange,
    find_best_version,
    is_compatible,
)


class TestSemVerParse:
    """Tests for SemVer.parse()."""

    def test_parse_basic_version(self):
        """Parse a basic semver string."""
        v = SemVer.parse("1.2.3")
        assert v.major == 1
        assert v.minor == 2
        assert v.patch == 3
        assert v.prerelease is None
        assert v.build is None

    def test_parse_version_with_prerelease(self):
        """Parse a version with prerelease."""
        v = SemVer.parse("1.0.0-alpha.1")
        assert v.major == 1
        assert v.minor == 0
        assert v.patch == 0
        assert v.prerelease == "alpha.1"
        assert v.build is None

    def test_parse_version_with_build_metadata(self):
        """Parse a version with build metadata."""
        v = SemVer.parse("1.0.0+build.123")
        assert v.major == 1
        assert v.minor == 0
        assert v.patch == 0
        assert v.prerelease is None
        assert v.build == "build.123"

    def test_parse_version_with_prerelease_and_build(self):
        """Parse a version with both prerelease and build."""
        v = SemVer.parse("2.0.0-beta.1+build.456")
        assert v.major == 2
        assert v.minor == 0
        assert v.patch == 0
        assert v.prerelease == "beta.1"
        assert v.build == "build.456"

    def test_parse_invalid_version_raises(self):
        """Invalid version strings raise ValueError."""
        with pytest.raises(ValueError, match="Invalid semver"):
            SemVer.parse("invalid")

    def test_parse_incomplete_version_raises(self):
        """Incomplete version strings raise ValueError."""
        with pytest.raises(ValueError, match="Invalid semver"):
            SemVer.parse("1.2")

    def test_parse_leading_zeros_invalid(self):
        """Leading zeros in major/minor/patch are invalid."""
        with pytest.raises(ValueError, match="Invalid semver"):
            SemVer.parse("01.2.3")

    def test_parse_zero_version(self):
        """Parse 0.0.0 version."""
        v = SemVer.parse("0.0.0")
        assert v.major == 0
        assert v.minor == 0
        assert v.patch == 0


class TestSemVerComparison:
    """Tests for SemVer comparison operators."""

    def test_equal_versions(self):
        """Equal versions compare as equal."""
        assert SemVer.parse("1.2.3") == SemVer.parse("1.2.3")

    def test_different_versions_not_equal(self):
        """Different versions compare as not equal."""
        assert SemVer.parse("1.2.3") != SemVer.parse("1.2.4")

    def test_less_than_major(self):
        """Lower major version is less than."""
        assert SemVer.parse("1.0.0") < SemVer.parse("2.0.0")

    def test_less_than_minor(self):
        """Lower minor version is less than."""
        assert SemVer.parse("1.1.0") < SemVer.parse("1.2.0")

    def test_less_than_patch(self):
        """Lower patch version is less than."""
        assert SemVer.parse("1.2.3") < SemVer.parse("1.2.4")

    def test_prerelease_less_than_release(self):
        """Prerelease version is less than release."""
        assert SemVer.parse("1.0.0-alpha") < SemVer.parse("1.0.0")

    def test_prerelease_ordering(self):
        """Prerelease versions are ordered correctly."""
        assert SemVer.parse("1.0.0-alpha") < SemVer.parse("1.0.0-beta")
        assert SemVer.parse("1.0.0-alpha.1") < SemVer.parse("1.0.0-alpha.2")

    def test_greater_than(self):
        """Greater than comparison works."""
        assert SemVer.parse("2.0.0") > SemVer.parse("1.0.0")

    def test_less_than_or_equal(self):
        """Less than or equal comparison works."""
        assert SemVer.parse("1.0.0") <= SemVer.parse("1.0.0")
        assert SemVer.parse("1.0.0") <= SemVer.parse("2.0.0")

    def test_greater_than_or_equal(self):
        """Greater than or equal comparison works."""
        assert SemVer.parse("2.0.0") >= SemVer.parse("2.0.0")
        assert SemVer.parse("2.0.0") >= SemVer.parse("1.0.0")

    def test_build_metadata_ignored_in_comparison(self):
        """Build metadata is ignored in equality comparison."""
        v1 = SemVer.parse("1.0.0+build1")
        v2 = SemVer.parse("1.0.0+build2")
        assert v1 == v2


class TestSemVerStr:
    """Tests for SemVer string representation."""

    def test_str_basic(self):
        """Basic version string representation."""
        assert str(SemVer.parse("1.2.3")) == "1.2.3"

    def test_str_with_prerelease(self):
        """Version string with prerelease."""
        assert str(SemVer.parse("1.0.0-beta.1")) == "1.0.0-beta.1"

    def test_str_with_build(self):
        """Version string with build metadata."""
        assert str(SemVer.parse("1.0.0+build.123")) == "1.0.0+build.123"


class TestVersionRange:
    """Tests for VersionRange class."""

    def test_caret_range_major(self):
        """Caret range for major version."""
        r = VersionRange("^1.2.3")
        assert r.matches("1.2.3")
        assert r.matches("1.9.9")
        assert not r.matches("2.0.0")
        assert not r.matches("1.2.2")

    def test_caret_range_zero_minor(self):
        """Caret range for 0.x version."""
        r = VersionRange("^0.2.3")
        assert r.matches("0.2.3")
        assert r.matches("0.2.9")
        assert not r.matches("0.3.0")
        assert not r.matches("1.0.0")

    def test_caret_range_zero_zero(self):
        """Caret range for 0.0.x version."""
        r = VersionRange("^0.0.3")
        assert r.matches("0.0.3")
        assert not r.matches("0.0.4")
        assert not r.matches("0.1.0")

    def test_tilde_range(self):
        """Tilde range (patch-level changes)."""
        r = VersionRange("~1.2.3")
        assert r.matches("1.2.3")
        assert r.matches("1.2.9")
        assert not r.matches("1.3.0")
        assert not r.matches("1.2.2")

    def test_exact_version(self):
        """Exact version match."""
        r = VersionRange("1.2.3")
        assert r.matches("1.2.3")
        assert not r.matches("1.2.4")
        assert not r.matches("1.2.2")

    def test_greater_or_equal(self):
        """Greater or equal range."""
        r = VersionRange(">=1.0.0")
        assert r.matches("1.0.0")
        assert r.matches("2.0.0")
        assert not r.matches("0.9.9")

    def test_less_than(self):
        """Less than range."""
        r = VersionRange("<2.0.0")
        assert r.matches("1.0.0")
        assert r.matches("1.9.9")
        assert not r.matches("2.0.0")

    def test_latest_matches_all(self):
        """Latest matches all versions."""
        r = VersionRange("latest")
        assert r.matches("0.0.1")
        assert r.matches("1.0.0")
        assert r.matches("99.99.99")

    def test_matches_with_string(self):
        """Matches accepts string version."""
        r = VersionRange("^1.0.0")
        assert r.matches("1.5.0")


class TestIsCompatible:
    """Tests for is_compatible function."""

    def test_compatible_caret(self):
        """Version compatible with caret range."""
        assert is_compatible("^1.0.0", "1.5.0")
        assert not is_compatible("^1.0.0", "2.0.0")

    def test_compatible_tilde(self):
        """Version compatible with tilde range."""
        assert is_compatible("~1.2.0", "1.2.5")
        assert not is_compatible("~1.2.0", "1.3.0")

    def test_compatible_exact(self):
        """Version compatible with exact match."""
        assert is_compatible("1.0.0", "1.0.0")
        assert not is_compatible("1.0.0", "1.0.1")

    def test_invalid_spec_returns_false(self):
        """Invalid spec returns False."""
        assert not is_compatible("invalid", "1.0.0")


class TestFindBestVersion:
    """Tests for find_best_version function."""

    def test_finds_highest_matching(self):
        """Finds the highest matching version."""
        available = ["1.0.0", "1.1.0", "1.2.0", "2.0.0"]
        assert find_best_version("^1.0.0", available) == "1.2.0"

    def test_returns_none_when_no_match(self):
        """Returns None when no version matches."""
        available = ["0.1.0", "0.2.0"]
        assert find_best_version("^1.0.0", available) is None

    def test_handles_prerelease(self):
        """Handles prerelease versions correctly."""
        available = ["1.0.0-alpha", "1.0.0-beta", "1.0.0"]
        # Exact prerelease match
        assert find_best_version("1.0.0-alpha", available) == "1.0.0-alpha"

    def test_skips_invalid_versions(self):
        """Skips invalid version strings."""
        available = ["1.0.0", "invalid", "1.1.0"]
        assert find_best_version("^1.0.0", available) == "1.1.0"

    def test_empty_available_returns_none(self):
        """Empty available list returns None."""
        assert find_best_version("^1.0.0", []) is None

    def test_latest_returns_highest(self):
        """Latest spec returns highest version."""
        available = ["1.0.0", "2.0.0", "1.5.0"]
        assert find_best_version("latest", available) == "2.0.0"

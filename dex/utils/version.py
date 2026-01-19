"""Semantic versioning utilities."""

import re
from dataclasses import dataclass
from functools import total_ordering


@total_ordering
@dataclass
class SemVer:
    """Semantic version representation."""

    major: int
    minor: int
    patch: int
    prerelease: str | None = None
    build: str | None = None

    _SEMVER_PATTERN = re.compile(
        r"^(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)"
        r"(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)"
        r"(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?"
        r"(?:\+(?P<build>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$"
    )

    @classmethod
    def parse(cls, version_str: str) -> "SemVer":
        """Parse a semver string.

        Args:
            version_str: Version string (e.g., "1.2.3", "2.0.0-beta.1+build.123")

        Returns:
            SemVer instance

        Raises:
            ValueError: If the string is not valid semver
        """
        match = cls._SEMVER_PATTERN.match(version_str)
        if not match:
            raise ValueError(f"Invalid semver: {version_str}")

        return cls(
            major=int(match.group("major")),
            minor=int(match.group("minor")),
            patch=int(match.group("patch")),
            prerelease=match.group("prerelease"),
            build=match.group("build"),
        )

    def __str__(self) -> str:
        version = f"{self.major}.{self.minor}.{self.patch}"
        if self.prerelease:
            version += f"-{self.prerelease}"
        if self.build:
            version += f"+{self.build}"
        return version

    def __eq__(self, other: object) -> bool:
        if not isinstance(other, SemVer):
            return NotImplemented
        return (
            self.major == other.major
            and self.minor == other.minor
            and self.patch == other.patch
            and self.prerelease == other.prerelease
        )

    def __lt__(self, other: object) -> bool:
        if not isinstance(other, SemVer):
            return NotImplemented

        # Compare major.minor.patch
        if (self.major, self.minor, self.patch) != (other.major, other.minor, other.patch):
            return (self.major, self.minor, self.patch) < (other.major, other.minor, other.patch)

        # Prerelease versions have lower precedence
        if self.prerelease and not other.prerelease:
            return True
        if not self.prerelease and other.prerelease:
            return False
        if self.prerelease and other.prerelease:
            return self._compare_prerelease(self.prerelease, other.prerelease) < 0

        return False

    @staticmethod
    def _compare_prerelease(a: str, b: str) -> int:
        """Compare two prerelease strings."""
        parts_a = a.split(".")
        parts_b = b.split(".")

        for pa, pb in zip(parts_a, parts_b, strict=False):
            # Numeric identifiers are compared as integers
            try:
                na, nb = int(pa), int(pb)
                if na != nb:
                    return na - nb
            except ValueError:
                # Alphanumeric identifiers are compared lexically
                if pa != pb:
                    return -1 if pa < pb else 1

        # Longer prerelease has higher precedence
        return len(parts_a) - len(parts_b)

    def __hash__(self) -> int:
        return hash((self.major, self.minor, self.patch, self.prerelease))


class VersionRange:
    """A version range specification."""

    def __init__(self, spec: str):
        """Initialize a version range.

        Args:
            spec: Version specifier (e.g., "^1.2.3", "~2.0.0", ">=1.0.0 <2.0.0")
        """
        self.spec = spec
        self._constraints = self._parse_spec(spec)

    def _parse_spec(self, spec: str) -> list[tuple[str, SemVer]]:
        """Parse a version specifier into constraints."""
        spec = spec.strip()

        if spec == "latest":
            return [(">=", SemVer(0, 0, 0))]

        # Caret range: ^1.2.3 means >=1.2.3 <2.0.0
        if spec.startswith("^"):
            base = SemVer.parse(spec[1:])
            if base.major == 0:
                if base.minor == 0:
                    # ^0.0.x means exactly that patch
                    return [(">=", base), ("<", SemVer(0, 0, base.patch + 1))]
                # ^0.y.z means >=0.y.z <0.(y+1).0
                return [(">=", base), ("<", SemVer(0, base.minor + 1, 0))]
            return [(">=", base), ("<", SemVer(base.major + 1, 0, 0))]

        # Tilde range: ~1.2.3 means >=1.2.3 <1.3.0
        if spec.startswith("~"):
            base = SemVer.parse(spec[1:])
            return [(">=", base), ("<", SemVer(base.major, base.minor + 1, 0))]

        # Range operators
        for op in (">=", "<=", ">", "<", "="):
            if spec.startswith(op):
                version = SemVer.parse(spec[len(op) :].strip())
                return [(op, version)]

        # Exact version
        return [("=", SemVer.parse(spec))]

    def matches(self, version: SemVer | str) -> bool:
        """Check if a version matches this range.

        Args:
            version: Version to check

        Returns:
            True if the version satisfies the range
        """
        if isinstance(version, str):
            version = SemVer.parse(version)

        for op, constraint in self._constraints:
            if op == ">=" and version < constraint:
                return False
            if op == "<=" and version > constraint:
                return False
            if op == ">" and version <= constraint:
                return False
            if op == "<" and version >= constraint:
                return False
            if op == "=" and version != constraint:
                return False

        return True

    def __str__(self) -> str:
        return self.spec

    def __repr__(self) -> str:
        return f"VersionRange({self.spec!r})"


def is_compatible(spec: str, version: str) -> bool:
    """Check if a version is compatible with a specifier.

    Args:
        spec: Version specifier (e.g., "^1.2.3", "~2.0.0")
        version: Version string to check

    Returns:
        True if compatible
    """
    try:
        return VersionRange(spec).matches(version)
    except ValueError:
        return False


def find_best_version(spec: str, available: list[str]) -> str | None:
    """Find the best matching version from a list.

    Args:
        spec: Version specifier
        available: List of available versions

    Returns:
        Best matching version, or None if no match
    """
    range_ = VersionRange(spec)
    matching = []

    for v in available:
        try:
            semver = SemVer.parse(v)
            if range_.matches(semver):
                matching.append(semver)
        except ValueError:
            continue

    if not matching:
        return None

    # Return the highest matching version
    return str(max(matching))

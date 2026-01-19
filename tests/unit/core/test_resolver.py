"""Tests for dex.core.resolver module."""

import pytest

from dex.core.resolver import (
    CircularDependencyError,
    DependencyResolver,
    ResolutionResult,
    ResolvedDependency,
    VersionConflictError,
    resolve_dependencies,
)


class TestDependencyResolver:
    """Tests for DependencyResolver class."""

    def test_resolves_simple_dependencies(self):
        """Resolves simple dependencies without transitive deps."""
        available = {
            "pkg-a": ["1.0.0", "1.1.0"],
            "pkg-b": ["2.0.0"],
        }
        resolver = DependencyResolver(available)

        result = resolver.resolve({"pkg-a": "^1.0.0", "pkg-b": "^2.0.0"})

        assert len(result.resolved) == 2
        names = [r.name for r in result.resolved]
        assert "pkg-a" in names
        assert "pkg-b" in names

    def test_resolves_highest_compatible_version(self):
        """Selects highest compatible version."""
        available = {
            "pkg-a": ["1.0.0", "1.1.0", "1.2.0", "2.0.0"],
        }
        resolver = DependencyResolver(available)

        result = resolver.resolve({"pkg-a": "^1.0.0"})

        pkg_a = next(r for r in result.resolved if r.name == "pkg-a")
        assert pkg_a.version == "1.2.0"  # Highest in ^1.x.x range


class TestDependencyResolverWithManifests:
    """Tests for DependencyResolver with transitive dependencies."""

    def test_resolves_transitive_dependencies(self):
        """Resolves transitive dependencies."""
        available = {
            "pkg-a": ["1.0.0"],
            "pkg-b": ["2.0.0"],
        }

        def get_manifest(name, version):
            """Mock manifest getter."""
            from dex.config.schemas import PluginManifest

            if name == "pkg-a":
                return PluginManifest(
                    name="pkg-a",
                    version="1.0.0",
                    description="Test",
                    dependencies={"pkg-b": "^2.0.0"},
                )
            return PluginManifest(
                name=name,
                version=version,
                description="Test",
            )

        resolver = DependencyResolver(available, get_manifest)

        result = resolver.resolve({"pkg-a": "^1.0.0"})

        names = [r.name for r in result.resolved]
        assert "pkg-a" in names
        assert "pkg-b" in names  # Transitive dep


class TestDependencyResolverErrors:
    """Tests for DependencyResolver error handling."""

    def test_detects_version_conflict(self):
        """Detects version conflicts."""

        def get_manifest(name, version):
            from dex.config.schemas import PluginManifest

            if name == "root-a":
                return PluginManifest(
                    name="root-a",
                    version="1.0.0",
                    description="Test",
                    dependencies={"pkg-a": "^1.0.0"},
                )
            elif name == "root-b":
                return PluginManifest(
                    name="root-b",
                    version="1.0.0",
                    description="Test",
                    dependencies={"pkg-a": "^2.0.0"},  # Conflict
                )
            return PluginManifest(
                name=name,
                version=version,
                description="Test",
            )

        resolver = DependencyResolver(
            {
                "root-a": ["1.0.0"],
                "root-b": ["1.0.0"],
                "pkg-a": ["1.0.0"],  # No 2.x available
            },
            get_manifest,
        )

        with pytest.raises(VersionConflictError) as exc_info:
            resolver.resolve({"root-a": "^1.0.0", "root-b": "^1.0.0"})

        assert exc_info.value.package == "pkg-a"

    def test_detects_circular_dependency(self):
        """Detects circular dependencies."""

        def get_manifest(name, version):
            from dex.config.schemas import PluginManifest

            deps = {
                "pkg-a": {"pkg-b": "^1.0.0"},
                "pkg-b": {"pkg-c": "^1.0.0"},
                "pkg-c": {"pkg-a": "^1.0.0"},  # Circular
            }
            return PluginManifest(
                name=name,
                version=version,
                description="Test",
                dependencies=deps.get(name, {}),
            )

        resolver = DependencyResolver(
            {
                "pkg-a": ["1.0.0"],
                "pkg-b": ["1.0.0"],
                "pkg-c": ["1.0.0"],
            },
            get_manifest,
        )

        with pytest.raises(CircularDependencyError) as exc_info:
            resolver.resolve({"pkg-a": "^1.0.0"})

        # Chain should include the circular path
        assert len(exc_info.value.chain) > 2


class TestResolutionResult:
    """Tests for ResolutionResult class."""

    def test_installation_order(self):
        """Returns package names in installation order."""
        result = ResolutionResult(
            resolved=[
                ResolvedDependency(name="dep1", version="1.0.0"),
                ResolvedDependency(name="dep2", version="2.0.0"),
            ]
        )

        order = result.installation_order()

        assert order == ["dep1", "dep2"]


class TestResolvedDependency:
    """Tests for ResolvedDependency class."""

    def test_attributes(self):
        """Has expected attributes."""
        dep = ResolvedDependency(
            name="pkg",
            version="1.0.0",
            required_by=["root"],
        )

        assert dep.name == "pkg"
        assert dep.version == "1.0.0"
        assert dep.required_by == ["root"]


class TestVersionConflictError:
    """Tests for VersionConflictError exception."""

    def test_error_attributes(self):
        """Has package and requirements attributes."""
        error = VersionConflictError(
            package="pkg-a",
            requirements=[("pkg-b", "^1.0.0"), ("pkg-c", "^2.0.0")],
        )

        assert error.package == "pkg-a"
        assert len(error.requirements) == 2

    def test_error_message(self):
        """Has descriptive message."""
        error = VersionConflictError(
            package="pkg-a",
            requirements=[("pkg-b", "^1.0.0")],
        )

        assert "pkg-a" in str(error)
        assert "conflict" in str(error).lower()


class TestCircularDependencyError:
    """Tests for CircularDependencyError exception."""

    def test_error_attributes(self):
        """Has chain attribute."""
        error = CircularDependencyError(chain=["pkg-a", "pkg-b", "pkg-a"])

        assert error.chain == ["pkg-a", "pkg-b", "pkg-a"]

    def test_error_message(self):
        """Has descriptive message."""
        error = CircularDependencyError(chain=["a", "b", "a"])

        assert "a" in str(error)
        assert "->" in str(error)


class TestResolveDependencies:
    """Tests for resolve_dependencies convenience function."""

    def test_resolves_dependencies(self):
        """Resolves dependencies using convenience function."""
        available = {
            "pkg-a": ["1.0.0"],
            "pkg-b": ["2.0.0"],
        }

        result = resolve_dependencies(
            {"pkg-a": "^1.0.0", "pkg-b": "^2.0.0"},
            available,
        )

        assert len(result.resolved) == 2

    def test_without_available_packages(self):
        """Works without available packages dict."""
        result = resolve_dependencies({"pkg-a": "1.0.0"})

        # Should still resolve, using the spec as version
        assert len(result.resolved) == 1

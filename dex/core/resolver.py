"""Dependency resolver for Dex.

This module handles resolving plugin dependencies, detecting conflicts,
and determining installation order.
"""

from dataclasses import dataclass, field
from typing import Any

from dex.utils.version import SemVer, is_compatible


class DependencyError(Exception):
    """Error during dependency resolution."""

    pass


class VersionConflictError(DependencyError):
    """Error when version requirements conflict."""

    def __init__(
        self,
        package: str,
        requirements: list[tuple[str, str]],
    ):
        self.package = package
        self.requirements = requirements

        req_strs = [f"{pkg} requires {ver}" for pkg, ver in requirements]
        message = f"Version conflict for {package}: {', '.join(req_strs)}"
        super().__init__(message)


class CircularDependencyError(DependencyError):
    """Error when circular dependencies are detected."""

    def __init__(self, chain: list[str]):
        self.chain = chain
        message = f"Circular dependency detected: {' -> '.join(chain)}"
        super().__init__(message)


@dataclass
class ResolvedDependency:
    """A resolved dependency."""

    name: str
    version: str
    required_by: list[str] = field(default_factory=list)


@dataclass
class ResolutionResult:
    """Result of dependency resolution."""

    resolved: list[ResolvedDependency] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)

    def installation_order(self) -> list[str]:
        """Get package names in installation order (dependencies first)."""
        return [r.name for r in self.resolved]


class DependencyResolver:
    """Resolves plugin dependencies.

    This resolver:
    1. Detects version conflicts (incompatible version requirements)
    2. Detects circular dependencies
    3. Selects the best version that satisfies all requirements
    4. Determines installation order
    """

    def __init__(
        self,
        available_packages: dict[str, list[str]] | None = None,
        get_manifest: Any = None,
    ):
        """Initialize the resolver.

        Args:
            available_packages: Dict mapping package names to available versions
            get_manifest: Callable to get a PluginManifest by name and version
        """
        self._available = available_packages or {}
        self._get_manifest = get_manifest
        self._resolved: dict[str, ResolvedDependency] = {}
        self._requirements: dict[str, list[tuple[str, str]]] = {}  # pkg -> [(requirer, spec)]
        self._visiting: set[str] = set()

    def resolve(
        self,
        root_requirements: dict[str, str],
    ) -> ResolutionResult:
        """Resolve dependencies for a set of root requirements.

        Args:
            root_requirements: Dict mapping package names to version specs

        Returns:
            ResolutionResult with resolved dependencies

        Raises:
            VersionConflictError: If version requirements conflict
            CircularDependencyError: If circular dependencies exist
        """
        self._resolved.clear()
        self._requirements.clear()
        self._visiting.clear()

        # Add root requirements
        for name, spec in root_requirements.items():
            self._add_requirement(name, spec, "root")

        # Resolve all requirements
        result = ResolutionResult()

        for name in list(self._requirements.keys()):
            if name not in self._resolved:
                self._resolve_package(name, [])

        # Build result in installation order (dependencies first)
        order = self._topological_sort()
        for name in order:
            if name in self._resolved:
                result.resolved.append(self._resolved[name])

        return result

    def _add_requirement(self, name: str, spec: str, required_by: str) -> None:
        """Add a version requirement for a package."""
        if name not in self._requirements:
            self._requirements[name] = []
        self._requirements[name].append((required_by, spec))

        # Check if this conflicts with an already-resolved version
        if name in self._resolved:
            resolved_version = self._resolved[name].version
            if not is_compatible(spec, resolved_version):
                raise VersionConflictError(name, self._requirements[name])

    def _resolve_package(self, name: str, path: list[str]) -> None:
        """Resolve a single package and its dependencies."""
        # Check for circular dependency
        if name in self._visiting:
            raise CircularDependencyError(path + [name])

        # Skip if already resolved
        if name in self._resolved:
            return

        self._visiting.add(name)
        path = path + [name]

        try:
            # Get all requirements for this package
            requirements = self._requirements.get(name, [])

            # Find the best version that satisfies all requirements
            version = self._find_compatible_version(name, requirements)

            # Create resolved entry
            self._resolved[name] = ResolvedDependency(
                name=name,
                version=version,
                required_by=[r[0] for r in requirements],
            )

            # Resolve dependencies if we have a manifest getter
            if self._get_manifest:
                manifest = self._get_manifest(name, version)
                if manifest and manifest.dependencies:
                    for dep_name, dep_spec in manifest.dependencies.items():
                        self._add_requirement(dep_name, dep_spec, name)
                        self._resolve_package(dep_name, path)

        finally:
            self._visiting.discard(name)

    def _find_compatible_version(
        self,
        name: str,
        requirements: list[tuple[str, str]],
    ) -> str:
        """Find the highest version that satisfies all requirements."""
        if not requirements:
            # No requirements - use latest
            available = self._available.get(name, ["0.0.0"])
            return available[-1] if available else "latest"

        # Get available versions
        available = self._available.get(name, [])

        if not available:
            # No versions available - return the spec from the first requirement
            # This handles the case where we don't have version info
            return requirements[0][1]

        # Find versions that satisfy all requirements
        compatible: list[SemVer] = []

        for ver_str in available:
            try:
                ver = SemVer.parse(ver_str)
                if all(is_compatible(spec, ver_str) for _, spec in requirements):
                    compatible.append(ver)
            except ValueError:
                continue

        if not compatible:
            raise VersionConflictError(name, requirements)

        # Return the highest compatible version
        return str(max(compatible))

    def _topological_sort(self) -> list[str]:
        """Sort resolved packages in installation order."""
        # Simple DFS-based topological sort
        result: list[str] = []
        visited: set[str] = set()

        def visit(name: str) -> None:
            if name in visited:
                return
            visited.add(name)

            resolved = self._resolved.get(name)
            if resolved:
                for dep in resolved.required_by:
                    if dep != "root" and dep in self._resolved:
                        visit(dep)

            result.append(name)

        for name in self._resolved:
            visit(name)

        return result


def resolve_dependencies(
    requirements: dict[str, str],
    available_packages: dict[str, list[str]] | None = None,
) -> ResolutionResult:
    """Convenience function to resolve dependencies.

    Args:
        requirements: Dict mapping package names to version specs
        available_packages: Optional dict of available versions

    Returns:
        ResolutionResult

    Raises:
        DependencyError: If resolution fails
    """
    resolver = DependencyResolver(available_packages)
    return resolver.resolve(requirements)

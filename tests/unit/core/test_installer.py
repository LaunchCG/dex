"""Tests for dex.core.installer module."""

import json
from pathlib import Path
from unittest.mock import patch

import pytest

from dex.config.schemas import (
    FileToWrite,
    InstallationPlan,
    PluginSpec,
)
from dex.core.installer import InstallError, InstallResult, InstallSummary, PluginInstaller
from dex.core.project import Project


@pytest.fixture
def installer(initialized_project: Project) -> PluginInstaller:
    """Get a PluginInstaller instance."""
    return PluginInstaller(initialized_project)


class TestPluginInstallerInit:
    """Tests for PluginInstaller initialization."""

    def test_initializes_with_project(self, initialized_project: Project):
        """Initializes with project."""
        installer = PluginInstaller(initialized_project)

        assert installer.project is initialized_project

    def test_gets_correct_adapter(self, initialized_project: Project):
        """Gets adapter for project's agent type."""
        installer = PluginInstaller(initialized_project)

        from dex.adapters.claude_code import ClaudeCodeAdapter

        assert isinstance(installer.adapter, ClaudeCodeAdapter)


class TestInstallResult:
    """Tests for InstallResult dataclass."""

    def test_successful_result(self):
        """Creates successful result."""
        result = InstallResult(
            plugin_name="test-plugin",
            version="1.0.0",
            success=True,
            message="Installed successfully",
        )

        assert result.success is True
        assert result.warnings == []

    def test_failed_result(self):
        """Creates failed result."""
        result = InstallResult(
            plugin_name="test-plugin",
            version="1.0.0",
            success=False,
            message="Installation failed",
        )

        assert result.success is False


class TestInstallSummary:
    """Tests for InstallSummary dataclass."""

    def test_success_count(self):
        """Counts successful installations."""
        summary = InstallSummary(
            results=[
                InstallResult("p1", "1.0.0", True),
                InstallResult("p2", "1.0.0", False),
                InstallResult("p3", "1.0.0", True),
            ]
        )

        assert summary.success_count == 2

    def test_failure_count(self):
        """Counts failed installations."""
        summary = InstallSummary(
            results=[
                InstallResult("p1", "1.0.0", True),
                InstallResult("p2", "1.0.0", False),
            ]
        )

        assert summary.failure_count == 1

    def test_all_successful(self):
        """Checks if all successful."""
        all_success = InstallSummary(
            results=[
                InstallResult("p1", "1.0.0", True),
                InstallResult("p2", "1.0.0", True),
            ]
        )
        some_failed = InstallSummary(
            results=[
                InstallResult("p1", "1.0.0", True),
                InstallResult("p2", "1.0.0", False),
            ]
        )

        assert all_success.all_successful is True
        assert some_failed.all_successful is False


class TestPluginInstallerInstall:
    """Tests for PluginInstaller.install() method."""

    def test_returns_empty_summary_for_no_plugins(self, installer: PluginInstaller):
        """Returns empty summary when no plugins to install."""
        summary = installer.install({})

        assert len(summary.results) == 0

    def test_uses_project_plugins_when_none_specified(
        self, initialized_project: Project, temp_plugin_dir: Path
    ):
        """Uses project plugins when plugin_specs is None."""
        # Configure project with a plugin
        initialized_project._config.registries["local"] = f"file://{temp_plugin_dir.parent}"
        initialized_project._config.default_registry = "local"
        initialized_project.add_plugin("test-plugin", PluginSpec(source=f"file:{temp_plugin_dir}"))

        installer = PluginInstaller(initialized_project)

        # Mock the install to prevent actual file operations
        with patch.object(installer, "_install_single_plugin") as mock_install:
            mock_install.return_value = InstallResult("test-plugin", "1.0.0", True)
            installer.install(use_lockfile=False)

        # Should have attempted to install
        assert mock_install.called


class TestPluginInstallerResolvePlugin:
    """Tests for PluginInstaller._resolve_plugin() method."""

    def test_uses_locked_version(self, initialized_project: Project, temp_registry: Path):
        """Uses locked version when available."""
        import tarfile

        # Create tarball for the locked version in the registry
        tarball_path = temp_registry / "test-plugin-1.0.0.tar.gz"
        with tarfile.open(tarball_path, "w:gz"):
            pass  # Empty tarball for test

        installer = PluginInstaller(initialized_project)
        installer.lockfile_manager.load()
        installer.lockfile_manager.lock_plugin(
            "test-plugin",
            "1.0.0",
            f"file://{temp_registry}/test-plugin-1.0.0.tar.gz",
        )

        # Configure registry with proper registry that has registry.json
        initialized_project._config.registries["local"] = f"file://{temp_registry}"
        initialized_project._config.default_registry = "local"

        # When using lockfile with exact locked version, that version should be used
        spec = PluginSpec(version="1.0.0")  # Exact locked version
        resolved = installer._resolve_plugin("test-plugin", spec, use_lockfile=True)

        assert resolved is not None
        assert resolved.version == "1.0.0"  # Should use locked version

    def test_resolves_from_direct_source(self, initialized_project: Project, temp_plugin_dir: Path):
        """Resolves from direct source."""
        installer = PluginInstaller(initialized_project)

        spec = PluginSpec(source=f"file:{temp_plugin_dir}")
        resolved = installer._resolve_plugin("test-plugin", spec, use_lockfile=False)

        assert resolved is not None
        assert resolved.name == "test-plugin"


class TestPluginInstallerRenderContext:
    """Tests for PluginInstaller._render_context() method."""

    def test_renders_single_file(self, installer: PluginInstaller, temp_dir: Path):
        """Renders a single context file."""
        context_file = temp_dir / "context.md"
        context_file.write_text("Hello, {{ name }}!")

        result = installer._render_context(
            "./context.md",
            temp_dir,
            {"name": "World"},
        )

        assert result == "Hello, World!"

    def test_renders_multiple_files(self, installer: PluginInstaller, temp_dir: Path):
        """Renders multiple context files."""
        (temp_dir / "file1.md").write_text("Part 1")
        (temp_dir / "file2.md").write_text("Part 2")

        result = installer._render_context(
            ["./file1.md", "./file2.md"],
            temp_dir,
            {},
        )

        assert "Part 1" in result
        assert "Part 2" in result

    def test_handles_missing_file(self, installer: PluginInstaller, temp_dir: Path):
        """Handles missing context file gracefully."""
        result = installer._render_context(
            "./nonexistent.md",
            temp_dir,
            {},
        )

        assert "not found" in result.lower()


class TestPluginInstallerEvaluateCondition:
    """Tests for PluginInstaller._evaluate_condition() method."""

    def test_evaluates_simple_equality(self, installer: PluginInstaller):
        """Evaluates simple equality condition."""
        context = {"platform": {"os": "linux"}}

        assert installer._evaluate_condition("platform.os == 'linux'", context) is True
        assert installer._evaluate_condition("platform.os == 'windows'", context) is False

    def test_handles_invalid_condition(self, installer: PluginInstaller):
        """Handles invalid condition gracefully."""
        result = installer._evaluate_condition("invalid {{ syntax", {})
        assert result is False


class TestPluginInstallerExecutePlan:
    """Tests for PluginInstaller._execute_plan() method."""

    def test_creates_directories(self, installer: PluginInstaller, temp_dir: Path):
        """Creates specified directories."""
        plan = InstallationPlan(
            directories_to_create=[temp_dir / "new_dir"],
        )

        installer._execute_plan(plan)

        assert (temp_dir / "new_dir").exists()

    def test_writes_files(self, installer: PluginInstaller, temp_dir: Path):
        """Writes specified files."""
        plan = InstallationPlan(
            directories_to_create=[temp_dir],
            files_to_write=[FileToWrite(path=temp_dir / "file.txt", content="content")],
        )

        installer._execute_plan(plan)

        assert (temp_dir / "file.txt").read_text() == "content"

    def test_copies_files(self, installer: PluginInstaller, temp_dir: Path):
        """Copies specified files."""
        src_file = temp_dir / "source.txt"
        src_file.write_text("source content")
        dest_file = temp_dir / "dest.txt"

        plan = InstallationPlan(
            files_to_copy={src_file: dest_file},
        )

        installer._execute_plan(plan)

        assert dest_file.read_text() == "source content"


class TestPluginInstallerUpdateMCPConfig:
    """Tests for PluginInstaller._update_mcp_config() method."""

    def test_creates_new_config(self, installer: PluginInstaller, temp_dir: Path):
        """Creates new MCP config file."""
        # Change project root to temp_dir for this test
        installer.project._root = temp_dir
        claude_dir = temp_dir / ".claude"
        claude_dir.mkdir(parents=True)

        mcp_configs = {"test-server": {"command": "node"}}

        installer._update_mcp_config(mcp_configs)

        config_path = temp_dir / ".mcp.json"
        assert config_path.exists()
        config_data = json.loads(config_path.read_text())
        assert "mcpServers" in config_data

    def test_merges_with_existing_config(self, installer: PluginInstaller, temp_dir: Path):
        """Merges with existing config."""
        installer.project._root = temp_dir
        claude_dir = temp_dir / ".claude"
        claude_dir.mkdir(parents=True)

        # Create existing config
        existing = {"mcpServers": {"existing": {"command": "python"}}, "other": "setting"}
        (temp_dir / ".mcp.json").write_text(json.dumps(existing))

        installer._update_mcp_config({"new-server": {"command": "node"}})

        config_data = json.loads((temp_dir / ".mcp.json").read_text())
        assert "existing" in config_data["mcpServers"]
        assert "new-server" in config_data["mcpServers"]
        assert config_data["other"] == "setting"


class TestInstallError:
    """Tests for InstallError exception."""

    def test_error_attributes(self):
        """Has plugin_name attribute."""
        error = InstallError("Error message", plugin_name="test-plugin")

        assert error.plugin_name == "test-plugin"
        assert str(error) == "Error message"

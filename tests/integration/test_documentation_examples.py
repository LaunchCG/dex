"""
Integration tests that validate the example plugins from documentation.

These tests ensure that the example plugins in docs/examples/ are valid
and can be installed correctly, keeping documentation in sync with code.
"""

import json
import shutil
from pathlib import Path

import pytest

from dex.config.parser import load_plugin_manifest
from dex.config.schemas import PluginManifest, PluginSpec
from dex.core.installer import PluginInstaller
from dex.core.plugin import Plugin
from dex.core.project import Project
from dex.template.engine import TemplateEngine

# Path to example plugins
EXAMPLES_DIR = Path(__file__).parent.parent.parent / "docs" / "examples"


class TestSimpleSkillExample:
    """Validate the simple-skill example from docs."""

    @pytest.fixture
    def plugin_dir(self) -> Path:
        """Get the simple-skill example directory."""
        return EXAMPLES_DIR / "simple-skill"

    def test_example_exists(self, plugin_dir: Path) -> None:
        """Verify the example plugin directory exists."""
        assert plugin_dir.exists(), f"Example plugin not found at {plugin_dir}"
        assert (plugin_dir / "package.json").exists()
        assert (plugin_dir / "context" / "skill.md").exists()

    def test_manifest_is_valid(self, plugin_dir: Path) -> None:
        """Verify the manifest can be parsed."""
        manifest = load_plugin_manifest(plugin_dir)
        assert isinstance(manifest, PluginManifest)
        assert manifest.name == "simple-skill"
        assert manifest.version == "1.0.0"

    def test_has_expected_skill(self, plugin_dir: Path) -> None:
        """Verify the manifest has the expected skill."""
        manifest = load_plugin_manifest(plugin_dir)
        assert len(manifest.skills) == 1
        assert manifest.skills[0].name == "greeting"

    def test_plugin_can_load(self, plugin_dir: Path) -> None:
        """Verify the plugin can be loaded."""
        plugin = Plugin.load(plugin_dir)
        assert plugin.name == "simple-skill"
        assert plugin.version == "1.0.0"

    def test_context_file_is_readable(self, plugin_dir: Path) -> None:
        """Verify the context file exists and is readable."""
        context_path = plugin_dir / "context" / "skill.md"
        content = context_path.read_text()
        assert "# Greeting Skill" in content
        assert len(content) > 100

    def test_installation_to_project(self, temp_project: Path, plugin_dir: Path) -> None:
        """Test that the plugin can be installed to a project."""
        # Initialize project
        project = Project.init(temp_project, "claude-code", project_name="test")

        # Copy plugin to temp location
        temp_plugin_dir = temp_project / "plugins" / "simple-skill"
        shutil.copytree(plugin_dir, temp_plugin_dir)

        # Install the plugin using the correct API
        installer = PluginInstaller(project)
        summary = installer.install(
            {"simple-skill": PluginSpec(source=f"file:{temp_plugin_dir}")},
            use_lockfile=False,
        )

        assert summary.all_successful

        # Verify skill was installed
        skill_dir = temp_project / ".claude" / "skills" / "simple-skill-greeting"
        assert skill_dir.exists()
        assert (skill_dir / "SKILL.md").exists()


class TestMultiComponentExample:
    """Validate the multi-component example from docs."""

    @pytest.fixture
    def plugin_dir(self) -> Path:
        """Get the multi-component example directory."""
        return EXAMPLES_DIR / "multi-component"

    def test_example_exists(self, plugin_dir: Path) -> None:
        """Verify the example plugin directory exists."""
        assert plugin_dir.exists(), f"Example plugin not found at {plugin_dir}"
        assert (plugin_dir / "package.json").exists()
        assert (plugin_dir / "context" / "lint-skill.md").exists()
        assert (plugin_dir / "context" / "format-command.md").exists()
        assert (plugin_dir / "config" / "settings.json").exists()
        assert (plugin_dir / "servers" / "mcp-server.py").exists()

    def test_manifest_is_valid(self, plugin_dir: Path) -> None:
        """Verify the manifest can be parsed."""
        manifest = load_plugin_manifest(plugin_dir)
        assert isinstance(manifest, PluginManifest)
        assert manifest.name == "multi-component"
        assert manifest.version == "1.0.0"

    def test_has_expected_components(self, plugin_dir: Path) -> None:
        """Verify the manifest has all expected components."""
        manifest = load_plugin_manifest(plugin_dir)

        # Check skill
        assert len(manifest.skills) == 1
        assert manifest.skills[0].name == "lint"
        assert manifest.skills[0].files is not None

        # Check command
        assert len(manifest.commands) == 1
        assert manifest.commands[0].name == "format"

        # Check MCP server
        assert len(manifest.mcp_servers) == 1
        assert manifest.mcp_servers[0].name == "code-analyzer"
        assert manifest.mcp_servers[0].type == "bundled"

    def test_plugin_can_load(self, plugin_dir: Path) -> None:
        """Verify the plugin can be loaded."""
        plugin = Plugin.load(plugin_dir)
        assert plugin.name == "multi-component"
        assert len(plugin.skills) == 1
        assert len(plugin.commands) == 1
        assert len(plugin.mcp_servers) == 1

    def test_config_file_is_valid_json(self, plugin_dir: Path) -> None:
        """Verify the config file is valid JSON."""
        config_path = plugin_dir / "config" / "settings.json"
        config = json.loads(config_path.read_text())
        assert "rules" in config
        assert "severity" in config
        assert "ignore" in config

    def test_mcp_server_is_valid_python(self, plugin_dir: Path) -> None:
        """Verify the MCP server script is valid Python."""
        server_path = plugin_dir / "servers" / "mcp-server.py"
        content = server_path.read_text()

        # Check it can be compiled (syntax check)
        compile(content, str(server_path), "exec")

        # Check it has expected components
        assert "def analyze_code" in content
        assert "def main" in content
        assert "argparse" in content

    def test_installation_to_project(self, temp_project: Path, plugin_dir: Path) -> None:
        """Test that the plugin can be installed to a project."""
        # Initialize project
        project = Project.init(temp_project, "claude-code", project_name="test")

        # Copy plugin to temp location
        temp_plugin_dir = temp_project / "plugins" / "multi-component"
        shutil.copytree(plugin_dir, temp_plugin_dir)

        # Install the plugin using the correct API
        installer = PluginInstaller(project)
        summary = installer.install(
            {"multi-component": PluginSpec(source=f"file:{temp_plugin_dir}")},
            use_lockfile=False,
        )

        assert summary.all_successful

        # Verify skill was installed
        skill_dir = temp_project / ".claude" / "skills" / "multi-component-lint"
        assert skill_dir.exists()
        assert (skill_dir / "SKILL.md").exists()

        # Verify MCP config was updated
        mcp_config_path = temp_project / ".mcp.json"
        assert mcp_config_path.exists()
        mcp_config = json.loads(mcp_config_path.read_text())
        assert "mcpServers" in mcp_config


class TestTemplatedExample:
    """Validate the templated example from docs."""

    @pytest.fixture
    def plugin_dir(self) -> Path:
        """Get the templated example directory."""
        return EXAMPLES_DIR / "templated"

    def test_example_exists(self, plugin_dir: Path) -> None:
        """Verify the example plugin directory exists."""
        assert plugin_dir.exists(), f"Example plugin not found at {plugin_dir}"
        assert (plugin_dir / "package.json").exists()
        assert (plugin_dir / "context" / "skill.md").exists()

    def test_manifest_is_valid(self, plugin_dir: Path) -> None:
        """Verify the manifest can be parsed."""
        manifest = load_plugin_manifest(plugin_dir)
        assert isinstance(manifest, PluginManifest)
        assert manifest.name == "templated"
        assert manifest.version == "1.0.0"

    def test_has_expected_skill(self, plugin_dir: Path) -> None:
        """Verify the manifest has the expected skill."""
        manifest = load_plugin_manifest(plugin_dir)
        assert len(manifest.skills) == 1
        assert manifest.skills[0].name == "platform-aware"

    def test_plugin_can_load(self, plugin_dir: Path) -> None:
        """Verify the plugin can be loaded."""
        plugin = Plugin.load(plugin_dir)
        assert plugin.name == "templated"

    def test_context_has_template_syntax(self, plugin_dir: Path) -> None:
        """Verify the context file contains Jinja2 template syntax."""
        context_path = plugin_dir / "context" / "skill.md"
        content = context_path.read_text()

        # Check for template variables
        assert "{{ env.project.name }}" in content
        assert "{{ plugin.name }}" in content
        assert "{{ plugin.version }}" in content
        assert "{{ platform.os }}" in content
        assert "{{ platform.arch }}" in content
        assert "{{ agent.name }}" in content

        # Check for conditionals
        assert "{% if platform.os ==" in content
        assert "{% elif platform.os ==" in content
        assert "{% else %}" in content
        assert "{% endif %}" in content

    def test_template_syntax_is_valid(self, plugin_dir: Path) -> None:
        """Verify the template syntax is valid Jinja2."""
        context_path = plugin_dir / "context" / "skill.md"
        content = context_path.read_text()

        engine = TemplateEngine()
        # check_syntax returns a list of errors (empty if valid)
        errors = engine.check_syntax(content)
        assert errors == [], f"Template syntax errors: {errors}"

    def test_template_renders_for_macos(self, plugin_dir: Path) -> None:
        """Test that template renders correctly for macOS."""
        context_path = plugin_dir / "context" / "skill.md"
        content = context_path.read_text()

        engine = TemplateEngine()
        context = {
            "env": {
                "project": {"name": "test-project", "root": "/path/to/project"},
                "home": "/Users/test",
                "USER": "testuser",
            },
            "plugin": {"name": "templated", "version": "1.0.0"},
            "platform": {"os": "macos", "arch": "arm64"},
            "agent": {"name": "claude-code"},
        }

        rendered = engine.render_string(content, context)

        # Check macOS section is included
        assert "You are running on **macOS**" in rendered
        assert "macOS-Specific Instructions" in rendered
        # Check Windows section is NOT included (conditional removed it)
        assert "Windows-Specific Instructions" not in rendered

    def test_template_renders_for_windows(self, plugin_dir: Path) -> None:
        """Test that template renders correctly for Windows."""
        context_path = plugin_dir / "context" / "skill.md"
        content = context_path.read_text()

        engine = TemplateEngine()
        context = {
            "env": {
                "project": {"name": "test-project", "root": "C:\\Projects\\test"},
                "home": "C:\\Users\\test",
                "USER": "testuser",
            },
            "plugin": {"name": "templated", "version": "1.0.0"},
            "platform": {"os": "windows", "arch": "x64"},
            "agent": {"name": "claude-code"},
        }

        rendered = engine.render_string(content, context)

        # Check Windows section is included
        assert "You are running on **Windows**" in rendered
        assert "Windows-Specific Instructions" in rendered
        # Check macOS section is NOT included
        assert "macOS-Specific Instructions" not in rendered

    def test_template_renders_for_linux(self, plugin_dir: Path) -> None:
        """Test that template renders correctly for Linux."""
        context_path = plugin_dir / "context" / "skill.md"
        content = context_path.read_text()

        engine = TemplateEngine()
        context = {
            "env": {
                "project": {"name": "test-project", "root": "/home/test/project"},
                "home": "/home/test",
                "USER": "testuser",
            },
            "plugin": {"name": "templated", "version": "1.0.0"},
            "platform": {"os": "linux", "arch": "x64"},
            "agent": {"name": "claude-code"},
        }

        rendered = engine.render_string(content, context)

        # Check Linux section is included
        assert "You are running on **Linux**" in rendered
        assert "Linux-Specific Instructions" in rendered
        # Check other sections are NOT included
        assert "Windows-Specific Instructions" not in rendered
        assert "macOS-Specific Instructions" not in rendered

    def test_installation_to_project(self, temp_project: Path, plugin_dir: Path) -> None:
        """Test that the plugin can be installed to a project."""
        # Initialize project
        project = Project.init(temp_project, "claude-code", project_name="test-project")

        # Copy plugin to temp location
        temp_plugin_dir = temp_project / "plugins" / "templated"
        shutil.copytree(plugin_dir, temp_plugin_dir)

        # Install the plugin using the correct API
        installer = PluginInstaller(project)
        summary = installer.install(
            {"templated": PluginSpec(source=f"file:{temp_plugin_dir}")},
            use_lockfile=False,
        )

        assert summary.all_successful

        # Verify skill was installed
        skill_dir = temp_project / ".claude" / "skills" / "templated-platform-aware"
        assert skill_dir.exists()
        skill_file = skill_dir / "SKILL.md"
        assert skill_file.exists()

        # Verify templates were rendered (no raw template syntax)
        content = skill_file.read_text()
        assert "{{ env.project.name }}" not in content
        assert "{% if" not in content
        # Verify rendered content
        assert "test-project" in content


class TestAllExamplesPresent:
    """Verify all documented examples exist."""

    def test_examples_directory_exists(self) -> None:
        """Verify the examples directory exists."""
        assert EXAMPLES_DIR.exists(), f"Examples directory not found at {EXAMPLES_DIR}"

    def test_all_documented_examples_exist(self) -> None:
        """Verify all examples mentioned in documentation exist."""
        expected_examples = ["simple-skill", "multi-component", "templated"]

        for example_name in expected_examples:
            example_dir = EXAMPLES_DIR / example_name
            assert example_dir.exists(), f"Example '{example_name}' not found"
            assert (
                example_dir / "package.json"
            ).exists(), f"Example '{example_name}' missing package.json"

    def test_no_unexpected_examples(self) -> None:
        """Verify no undocumented examples exist."""
        expected_examples = {"simple-skill", "multi-component", "templated"}

        if EXAMPLES_DIR.exists():
            actual_examples = {
                d.name for d in EXAMPLES_DIR.iterdir() if d.is_dir() and not d.name.startswith(".")
            }
            unexpected = actual_examples - expected_examples
            assert not unexpected, f"Unexpected examples found: {unexpected}"

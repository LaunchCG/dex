"""Full content validation integration tests for all adapters.

These tests validate the ENTIRE file contents produced by each adapter,
ensuring that frontmatter, rendered templates, and file structure are correct.

Tests cover:
- Cursor adapter
- GitHub Copilot adapter
- Antigravity adapter
- Codex adapter

Both direct source and registry-based installation are tested.
"""

import json
import tarfile
from pathlib import Path

import pytest

from dex.config.schemas import PluginSpec
from dex.core.installer import PluginInstaller
from dex.core.project import Project

# =============================================================================
# Shared Fixtures
# =============================================================================


@pytest.fixture
def test_plugin_dir(temp_dir: Path) -> Path:
    """Create a comprehensive test plugin for all adapter tests."""
    plugin_dir = temp_dir / "test-plugin"
    plugin_dir.mkdir()

    manifest = {
        "name": "test-plugin",
        "version": "1.0.0",
        "description": "A test plugin for full content validation",
        "skills": [
            {
                "name": "code-review",
                "description": "Automated code review assistance",
                "context": "./skills/code-review.md",
                "metadata": {
                    "globs": "**/*.py",
                    "alwaysApply": False,
                },
            }
        ],
        "commands": [
            {
                "name": "lint",
                "description": "Run linting on the codebase",
                "context": "./commands/lint.md",
            }
        ],
        "sub_agents": [
            {
                "name": "reviewer",
                "description": "Autonomous code review agent",
                "context": "./agents/reviewer.md",
            }
        ],
        "instructions": [
            {
                "name": "lint-guidance",
                "description": "Guidance for linting Python files",
                "context": "./instructions/lint-guidance.md",
                "apply_to": "**/*.py",
            }
        ],
        "rules": [
            {
                "name": "code-style",
                "description": "Code style rules for the project",
                "context": "./rules/code-style.md",
                "glob": "**/*.py",
            }
        ],
        "agent_file": {
            "context": "./agent_file/content.md",
        },
    }
    (plugin_dir / "package.json").write_text(json.dumps(manifest, indent=2))

    # Create skill context
    skills_dir = plugin_dir / "skills"
    skills_dir.mkdir()
    (skills_dir / "code-review.md").write_text(
        "# Code Review Skill\n\n"
        "This skill helps with code review.\n\n"
        "Plugin: {{ plugin.name }}\n"
        "Version: {{ plugin.version }}\n"
    )

    # Create command context
    commands_dir = plugin_dir / "commands"
    commands_dir.mkdir()
    (commands_dir / "lint.md").write_text(
        "# Lint Command\n\n" "Run linting tools on the codebase.\n"
    )

    # Create agent context
    agents_dir = plugin_dir / "agents"
    agents_dir.mkdir()
    (agents_dir / "reviewer.md").write_text(
        "# Reviewer Agent\n\n" "Autonomous code review agent.\n"
    )

    # Create instruction context
    instructions_dir = plugin_dir / "instructions"
    instructions_dir.mkdir()
    (instructions_dir / "lint-guidance.md").write_text(
        "# Lint Guidance\n\n" "Guidelines for linting Python files.\n"
    )

    # Create rule context
    rules_dir = plugin_dir / "rules"
    rules_dir.mkdir()
    (rules_dir / "code-style.md").write_text("# Code Style Rules\n\nFollow PEP 8 style guide.\n")

    # Create agent_file context
    agent_file_dir = plugin_dir / "agent_file"
    agent_file_dir.mkdir()
    (agent_file_dir / "content.md").write_text(
        "# Test Plugin Agent Instructions\n\n"
        "Plugin: {{ plugin.name }} v{{ plugin.version }}\n\n"
        "These are instructions for the agent.\n"
    )

    return plugin_dir


@pytest.fixture
def test_registry(temp_dir: Path, test_plugin_dir: Path) -> Path:
    """Create a test registry with the test plugin."""
    registry_dir = temp_dir / "registry"
    registry_dir.mkdir()

    # Create registry.json
    registry_data = {
        "packages": {
            "test-plugin": {
                "versions": ["1.0.0"],
                "latest": "1.0.0",
            }
        }
    }
    (registry_dir / "registry.json").write_text(json.dumps(registry_data))

    # Create tarball
    tarball_path = registry_dir / "test-plugin-1.0.0.tar.gz"
    with tarfile.open(tarball_path, "w:gz") as tar:
        tar.add(test_plugin_dir, arcname="test-plugin")

    return registry_dir


# =============================================================================
# Cursor Adapter Tests
# =============================================================================


class TestCursorAdapterFullContent:
    """Full content validation tests for Cursor adapter."""

    def test_direct_source_rule_full_content(self, temp_project: Path, test_plugin_dir: Path):
        """Cursor: validates full content of rule file from direct source."""
        project = Project.init(temp_project, "cursor", "test-project")

        installer = PluginInstaller(project)
        installer.install(
            {"test-plugin": PluginSpec(source=f"file:{test_plugin_dir}")},
            use_lockfile=False,
        )

        # Verify rule file exists and has correct full content
        rule_file = temp_project / ".cursor" / "rules" / "test-plugin-code-style.mdc"
        assert rule_file.exists()

        content = rule_file.read_text()

        # Validate frontmatter
        assert content.startswith("---\n")
        assert "description: Code style rules for the project" in content
        assert "globs: **/*.py" in content
        assert "alwaysApply:" in content
        assert "---" in content

        # Validate rendered content
        assert "# Code Style Rules" in content
        assert "Follow PEP 8 style guide." in content

    def test_registry_source_rule_full_content(self, temp_dir: Path, test_registry: Path):
        """Cursor: validates full content of rule file from registry."""
        project_dir = temp_dir / "cursor-registry-project"
        project_dir.mkdir()

        project = Project.init(project_dir, "cursor", "test-project")
        project._config.registries["local"] = f"file://{test_registry}"
        project._config.default_registry = "local"
        project.save()

        installer = PluginInstaller(project)
        installer.install(
            {"test-plugin": PluginSpec(version="^1.0.0")},
            use_lockfile=False,
        )

        rule_file = project_dir / ".cursor" / "rules" / "test-plugin-code-style.mdc"
        assert rule_file.exists()

        content = rule_file.read_text()

        # Same validations as direct source
        assert content.startswith("---\n")
        assert "description: Code style rules for the project" in content
        assert "# Code Style Rules" in content
        assert "Follow PEP 8 style guide." in content

    def test_cursor_does_not_install_skills(self, temp_project: Path, test_plugin_dir: Path):
        """Cursor: skills are not installed (not supported)."""
        project = Project.init(temp_project, "cursor")

        installer = PluginInstaller(project)
        installer.install(
            {"test-plugin": PluginSpec(source=f"file:{test_plugin_dir}")},
            use_lockfile=False,
        )

        # Skills should not be installed - only rules
        # Check that skill file was NOT created (skills not supported)
        skill_file = temp_project / ".cursor" / "rules" / "test-plugin-code-review.mdc"
        assert not skill_file.exists(), "Skills should not be installed in Cursor"

    def test_cursor_does_not_install_commands(self, temp_project: Path, test_plugin_dir: Path):
        """Cursor: commands are not installed (not supported)."""
        project = Project.init(temp_project, "cursor")

        installer = PluginInstaller(project)
        installer.install(
            {"test-plugin": PluginSpec(source=f"file:{test_plugin_dir}")},
            use_lockfile=False,
        )

        # Commands directory should not exist or should be empty
        commands_dir = temp_project / ".cursor" / "commands"
        if commands_dir.exists():
            assert not any(commands_dir.iterdir())


# =============================================================================
# GitHub Copilot Adapter Tests
# =============================================================================


class TestGitHubCopilotAdapterFullContent:
    """Full content validation tests for GitHub Copilot adapter."""

    def test_direct_source_instruction_full_content(
        self, temp_project: Path, test_plugin_dir: Path
    ):
        """GitHub Copilot: validates full content of instruction file from direct source."""
        project = Project.init(temp_project, "github-copilot", "test-project")

        installer = PluginInstaller(project)
        installer.install(
            {"test-plugin": PluginSpec(source=f"file:{test_plugin_dir}")},
            use_lockfile=False,
        )

        # Verify instruction file exists and has correct content
        instruction_file = (
            temp_project / ".github" / "instructions" / "lint-guidance.instructions.md"
        )
        assert instruction_file.exists()

        content = instruction_file.read_text()

        # applyTo was set, so frontmatter should be present
        assert content.startswith("---\n")
        assert 'applyTo: "**/*.py"' in content
        assert "---" in content

        # Validate rendered content
        assert "# Lint Guidance" in content
        assert "Guidelines for linting Python files." in content

    def test_direct_source_rules_full_content(self, temp_project: Path, test_plugin_dir: Path):
        """GitHub Copilot: validates full content of copilot-instructions.md."""
        project = Project.init(temp_project, "github-copilot")

        installer = PluginInstaller(project)
        installer.install(
            {"test-plugin": PluginSpec(source=f"file:{test_plugin_dir}")},
            use_lockfile=False,
        )

        rules_file = temp_project / ".github" / "copilot-instructions.md"
        assert rules_file.exists()

        content = rules_file.read_text()

        # Validate rule content from context file
        assert "# Code Style Rules" in content
        assert "Follow PEP 8 style guide." in content

    def test_registry_source_rules_full_content(self, temp_dir: Path, test_registry: Path):
        """GitHub Copilot: validates rules from registry installation."""
        project_dir = temp_dir / "copilot-registry-project"
        project_dir.mkdir()

        project = Project.init(project_dir, "github-copilot", "test-project")
        project._config.registries["local"] = f"file://{test_registry}"
        project._config.default_registry = "local"
        project.save()

        installer = PluginInstaller(project)
        installer.install(
            {"test-plugin": PluginSpec(version="1.0.0")},
            use_lockfile=False,
        )

        rules_file = project_dir / ".github" / "copilot-instructions.md"
        assert rules_file.exists()

        content = rules_file.read_text()
        assert "# Code Style Rules" in content
        assert "Follow PEP 8 style guide." in content


# =============================================================================
# Antigravity Adapter Tests
# =============================================================================


class TestAntigravityAdapterFullContent:
    """Full content validation tests for Antigravity adapter."""

    def test_direct_source_skill_full_content(self, temp_project: Path, test_plugin_dir: Path):
        """Antigravity: validates full content of skill file from direct source."""
        project = Project.init(temp_project, "antigravity", "test-project")

        installer = PluginInstaller(project)
        installer.install(
            {"test-plugin": PluginSpec(source=f"file:{test_plugin_dir}")},
            use_lockfile=False,
        )

        # Verify skill file exists
        skill_file = temp_project / ".agent" / "skills" / "code-review" / "SKILL.md"
        assert skill_file.exists()

        content = skill_file.read_text()

        # Validate frontmatter
        assert content.startswith("---\n")
        assert "name: code-review" in content
        assert "description: Automated code review assistance" in content
        assert "---" in content

        # Validate rendered content
        assert "# Code Review Skill" in content
        assert "Plugin: test-plugin" in content
        assert "Version: 1.0.0" in content

    def test_registry_source_skill_full_content(self, temp_dir: Path, test_registry: Path):
        """Antigravity: validates skill from registry installation."""
        project_dir = temp_dir / "antigravity-registry-project"
        project_dir.mkdir()

        project = Project.init(project_dir, "antigravity", "test-project")
        project._config.registries["local"] = f"file://{test_registry}"
        project._config.default_registry = "local"
        project.save()

        installer = PluginInstaller(project)
        installer.install(
            {"test-plugin": PluginSpec(version="~1.0.0")},
            use_lockfile=False,
        )

        skill_file = project_dir / ".agent" / "skills" / "code-review" / "SKILL.md"
        assert skill_file.exists()

        content = skill_file.read_text()
        assert "name: code-review" in content
        assert "Plugin: test-plugin" in content

    def test_antigravity_does_not_install_commands(self, temp_project: Path, test_plugin_dir: Path):
        """Antigravity: commands are not installed (not supported)."""
        project = Project.init(temp_project, "antigravity")

        installer = PluginInstaller(project)
        summary = installer.install(
            {"test-plugin": PluginSpec(source=f"file:{test_plugin_dir}")},
            use_lockfile=False,
        )

        # Should have warning about commands
        all_warnings = []
        for result in summary.results:
            all_warnings.extend(result.warnings)
        assert any("command" in w.lower() for w in all_warnings)


# =============================================================================
# Codex Adapter Tests
# =============================================================================


class TestCodexAdapterFullContent:
    """Full content validation tests for Codex adapter."""

    def test_direct_source_skill_full_content(self, temp_project: Path, test_plugin_dir: Path):
        """Codex: validates full content of skill file from direct source."""
        project = Project.init(temp_project, "codex", "test-project")

        installer = PluginInstaller(project)
        installer.install(
            {"test-plugin": PluginSpec(source=f"file:{test_plugin_dir}")},
            use_lockfile=False,
        )

        # Verify skill file exists
        skill_file = temp_project / ".codex" / "skills" / "code-review" / "SKILL.md"
        assert skill_file.exists()

        content = skill_file.read_text()

        # Validate frontmatter
        assert content.startswith("---\n")
        assert "name: code-review" in content
        assert "description: Automated code review assistance" in content
        assert "---" in content

        # Validate rendered content
        assert "# Code Review Skill" in content
        assert "Plugin: test-plugin" in content
        assert "Version: 1.0.0" in content

    def test_direct_source_rules_full_content(self, temp_project: Path, test_plugin_dir: Path):
        """Codex: validates rules go to .codex/rules/ (not AGENTS.md)."""
        project = Project.init(temp_project, "codex")

        installer = PluginInstaller(project)
        installer.install(
            {"test-plugin": PluginSpec(source=f"file:{test_plugin_dir}")},
            use_lockfile=False,
        )

        # Rules go to .codex/rules/ directory, NOT AGENTS.md
        rules_file = temp_project / ".codex" / "rules" / "code-style.md"
        assert rules_file.exists()

        content = rules_file.read_text()

        # Validate content from rule context file
        assert "# Code Style Rules" in content
        assert "Follow PEP 8 style guide." in content

    def test_registry_source_full_content(self, temp_dir: Path, test_registry: Path):
        """Codex: validates full installation from registry."""
        project_dir = temp_dir / "codex-registry-project"
        project_dir.mkdir()

        project = Project.init(project_dir, "codex", "test-project")
        project._config.registries["local"] = f"file://{test_registry}"
        project._config.default_registry = "local"
        project.save()

        installer = PluginInstaller(project)
        installer.install(
            {"test-plugin": PluginSpec(version=">=1.0.0")},
            use_lockfile=False,
        )

        # Verify skill
        skill_file = project_dir / ".codex" / "skills" / "code-review" / "SKILL.md"
        assert skill_file.exists()
        skill_content = skill_file.read_text()
        assert "name: code-review" in skill_content

        # Verify rules go to .codex/rules/, not AGENTS.md
        rules_file = project_dir / ".codex" / "rules" / "code-style.md"
        assert rules_file.exists()
        rules_content = rules_file.read_text()
        assert "# Code Style Rules" in rules_content


# =============================================================================
# Cross-Adapter Version Specifier Tests
# =============================================================================


class TestVersionSpecifiers:
    """Test various version specifier formats across adapters."""

    @pytest.mark.parametrize(
        "version_spec",
        [
            "1.0.0",  # Exact
            "^1.0.0",  # Caret (compatible)
            "~1.0.0",  # Tilde
            ">=1.0.0",  # Greater or equal
            "latest",  # Latest
        ],
    )
    @pytest.mark.parametrize(
        "agent_type",
        ["cursor", "github-copilot", "antigravity", "codex"],
    )
    def test_version_specifiers_work(
        self,
        temp_dir: Path,
        test_registry: Path,
        version_spec: str,
        agent_type: str,
    ):
        """All version specifiers work for all adapters."""
        project_dir = temp_dir / f"project-{agent_type}-{version_spec.replace('>', 'gt')}"
        project_dir.mkdir()

        project = Project.init(project_dir, agent_type, "test-project")  # type: ignore[arg-type]
        project._config.registries["local"] = f"file://{test_registry}"
        project._config.default_registry = "local"
        project.save()

        installer = PluginInstaller(project)
        summary = installer.install(
            {"test-plugin": PluginSpec(version=version_spec)},
            use_lockfile=False,
        )

        assert summary.all_successful, f"Failed for {agent_type} with {version_spec}"


# =============================================================================
# Direct Source vs Registry Comparison Tests
# =============================================================================


class TestDirectVsRegistryInstallation:
    """Ensure direct source and registry produce identical results."""

    @pytest.mark.parametrize(
        "agent_type",
        ["antigravity", "codex"],
    )
    def test_direct_and_registry_produce_same_skill_content(
        self,
        temp_dir: Path,
        test_plugin_dir: Path,
        test_registry: Path,
        agent_type: str,
    ):
        """Direct source and registry installation produce identical skill content."""
        # Install from direct source
        direct_project_dir = temp_dir / f"direct-{agent_type}"
        direct_project_dir.mkdir()
        direct_project = Project.init(
            direct_project_dir, agent_type, "test-project"  # type: ignore[arg-type]
        )
        direct_installer = PluginInstaller(direct_project)
        direct_installer.install(
            {"test-plugin": PluginSpec(source=f"file:{test_plugin_dir}")},
            use_lockfile=False,
        )

        # Install from registry
        registry_project_dir = temp_dir / f"registry-{agent_type}"
        registry_project_dir.mkdir()
        registry_project = Project.init(
            registry_project_dir, agent_type, "test-project"  # type: ignore[arg-type]
        )
        registry_project._config.registries["local"] = f"file://{test_registry}"
        registry_project._config.default_registry = "local"
        registry_project.save()
        registry_installer = PluginInstaller(registry_project)
        registry_installer.install(
            {"test-plugin": PluginSpec(version="1.0.0")},
            use_lockfile=False,
        )

        # Compare skill file contents
        if agent_type == "antigravity":
            direct_skill = direct_project_dir / ".agent" / "skills" / "code-review" / "SKILL.md"
            registry_skill = registry_project_dir / ".agent" / "skills" / "code-review" / "SKILL.md"
        else:  # codex
            direct_skill = direct_project_dir / ".codex" / "skills" / "code-review" / "SKILL.md"
            registry_skill = registry_project_dir / ".codex" / "skills" / "code-review" / "SKILL.md"

        assert direct_skill.exists()
        assert registry_skill.exists()
        assert direct_skill.read_text() == registry_skill.read_text()

    def test_cursor_direct_and_registry_produce_same_rule_content(
        self,
        temp_dir: Path,
        test_plugin_dir: Path,
        test_registry: Path,
    ):
        """Cursor: Direct source and registry installation produce identical rule content."""
        # Install from direct source
        direct_project_dir = temp_dir / "direct-cursor"
        direct_project_dir.mkdir()
        direct_project = Project.init(direct_project_dir, "cursor", "test-project")
        direct_installer = PluginInstaller(direct_project)
        direct_installer.install(
            {"test-plugin": PluginSpec(source=f"file:{test_plugin_dir}")},
            use_lockfile=False,
        )

        # Install from registry
        registry_project_dir = temp_dir / "registry-cursor"
        registry_project_dir.mkdir()
        registry_project = Project.init(registry_project_dir, "cursor", "test-project")
        registry_project._config.registries["local"] = f"file://{test_registry}"
        registry_project._config.default_registry = "local"
        registry_project.save()
        registry_installer = PluginInstaller(registry_project)
        registry_installer.install(
            {"test-plugin": PluginSpec(version="1.0.0")},
            use_lockfile=False,
        )

        # Compare rule file contents (Cursor only supports rules, not skills)
        direct_rule = direct_project_dir / ".cursor" / "rules" / "test-plugin-code-style.mdc"
        registry_rule = registry_project_dir / ".cursor" / "rules" / "test-plugin-code-style.mdc"

        assert direct_rule.exists()
        assert registry_rule.exists()
        assert direct_rule.read_text() == registry_rule.read_text()


# =============================================================================
# Agent File Full Content Tests
# =============================================================================


class TestAgentFileFullContent:
    """Full content validation tests for agent files (CLAUDE.md, AGENTS.md).

    These tests validate the ENTIRE file content, not just substrings.
    They test the marker-based content management when installing plugins
    into projects that already have content in their agent files.
    """

    def test_claude_code_agent_file_with_existing_content(
        self, temp_project: Path, test_plugin_dir: Path
    ):
        """Claude Code: agent file preserves existing content and adds plugin section."""
        # Create project with existing CLAUDE.md content
        project = Project.init(temp_project, "claude-code", "test-project")

        existing_content = """\
# My Project Instructions

This is my existing content that should be preserved.

## Important Notes

- Note 1
- Note 2
"""
        claude_md = temp_project / "CLAUDE.md"
        claude_md.write_text(existing_content)

        # Install plugin with force=True since CLAUDE.md already exists
        installer = PluginInstaller(project, force=True)
        installer.install(
            {"test-plugin": PluginSpec(source=f"file:{test_plugin_dir}")},
            use_lockfile=False,
        )

        # Validate FULL file content
        content = claude_md.read_text()

        expected = """\
# My Project Instructions

This is my existing content that should be preserved.

## Important Notes

- Note 1
- Note 2

<!-- dex:plugin:test-plugin:start -->
# Test Plugin Agent Instructions

Plugin: test-plugin v1.0.0

These are instructions for the agent.
<!-- dex:plugin:test-plugin:end -->
"""
        assert content == expected

    def test_claude_code_agent_file_empty_project(self, temp_project: Path, test_plugin_dir: Path):
        """Claude Code: creates CLAUDE.md with just plugin section when no existing file."""
        project = Project.init(temp_project, "claude-code", "test-project")

        # Install plugin (no existing CLAUDE.md)
        installer = PluginInstaller(project)
        installer.install(
            {"test-plugin": PluginSpec(source=f"file:{test_plugin_dir}")},
            use_lockfile=False,
        )

        # Validate FULL file content
        claude_md = temp_project / "CLAUDE.md"
        content = claude_md.read_text()

        expected = """\
<!-- dex:plugin:test-plugin:start -->
# Test Plugin Agent Instructions

Plugin: test-plugin v1.0.0

These are instructions for the agent.
<!-- dex:plugin:test-plugin:end -->
"""
        assert content == expected

    def test_codex_agents_md_with_existing_content(self, temp_project: Path, test_plugin_dir: Path):
        """Codex: AGENTS.md preserves existing content and adds plugin section."""
        # Create project with existing AGENTS.md content
        project = Project.init(temp_project, "codex", "test-project")

        existing_content = """\
# Project Agents

Custom agent instructions for this project.
"""
        agents_md = temp_project / "AGENTS.md"
        agents_md.write_text(existing_content)

        # Install plugin with force=True since AGENTS.md already exists
        installer = PluginInstaller(project, force=True)
        installer.install(
            {"test-plugin": PluginSpec(source=f"file:{test_plugin_dir}")},
            use_lockfile=False,
        )

        # Validate FULL file content
        content = agents_md.read_text()

        expected = """\
# Project Agents

Custom agent instructions for this project.

<!-- dex:plugin:test-plugin:start -->
# Test Plugin Agent Instructions

Plugin: test-plugin v1.0.0

These are instructions for the agent.
<!-- dex:plugin:test-plugin:end -->
"""
        assert content == expected

    def test_codex_agents_md_empty_project(self, temp_project: Path, test_plugin_dir: Path):
        """Codex: creates AGENTS.md with just plugin section when no existing file."""
        project = Project.init(temp_project, "codex", "test-project")

        # Install plugin (no existing AGENTS.md)
        installer = PluginInstaller(project)
        installer.install(
            {"test-plugin": PluginSpec(source=f"file:{test_plugin_dir}")},
            use_lockfile=False,
        )

        # Validate FULL file content
        agents_md = temp_project / "AGENTS.md"
        content = agents_md.read_text()

        expected = """\
<!-- dex:plugin:test-plugin:start -->
# Test Plugin Agent Instructions

Plugin: test-plugin v1.0.0

These are instructions for the agent.
<!-- dex:plugin:test-plugin:end -->
"""
        assert content == expected

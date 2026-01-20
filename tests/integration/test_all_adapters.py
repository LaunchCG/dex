"""Integration tests for all platform adapters with a shared plugin fixture.

This module tests installation across all supported platforms:
- Claude Code
- Cursor
- GitHub Copilot
- Antigravity
- Codex
"""

import json
from pathlib import Path

import pytest

from dex.adapters import get_adapter
from dex.config.schemas import PluginSpec
from dex.core.installer import PluginInstaller
from dex.core.project import Project


@pytest.fixture
def comprehensive_plugin(temp_dir: Path) -> Path:
    """Create a comprehensive test plugin with all component types.

    This single fixture provides a plugin suitable for testing all adapters.
    """
    plugin_dir = temp_dir / "comprehensive-plugin"
    plugin_dir.mkdir()

    manifest = {
        "name": "comprehensive-plugin",
        "version": "1.0.0",
        "description": "A comprehensive test plugin for all adapters",
        "skills": [
            {
                "name": "code-review",
                "description": "Automated code review skill",
                "context": "./skills/code-review.md",
                "files": ["./skills/config.json"],
            },
            {
                "name": "testing",
                "description": "Testing assistance skill",
                "context": "./skills/testing.md",
            },
        ],
        "commands": [
            {
                "name": "lint",
                "description": "Run linting on the project",
                "context": "./commands/lint.md",
            }
        ],
        "sub_agents": [
            {
                "name": "reviewer",
                "description": "Code review sub-agent",
                "context": "./agents/reviewer.md",
            }
        ],
        "rules": [
            {
                "name": "code-style",
                "description": "Code style rules for the project",
                "context": "./rules/code-style.md",
            }
        ],
        "mcp_servers": [
            {
                "name": "test-server",
                "type": "remote",
                "source": "npm:@test/mcp-server",
            }
        ],
        "instructions": [
            {
                "name": "lint",
                "description": "Linting instructions for Python files",
                "context": "./instructions/lint.md",
                "applyTo": "**/*.py",
            }
        ],
        "agent_file": {
            "context": "./agent_file/content.md",
        },
    }
    (plugin_dir / "package.json").write_text(json.dumps(manifest, indent=2))

    # Create skill context files
    skills_dir = plugin_dir / "skills"
    skills_dir.mkdir()
    (skills_dir / "code-review.md").write_text(
        "# Code Review Skill\n\n"
        "Plugin: {{ plugin.name }}\n"
        "Version: {{ plugin.version }}\n"
        "Project: {{ env.project.name }}\n"
    )
    (skills_dir / "testing.md").write_text(
        "# Testing Skill\n\n" "Help with writing and running tests.\n"
    )
    (skills_dir / "config.json").write_text('{"reviewers": ["claude"]}')

    # Create command context files
    commands_dir = plugin_dir / "commands"
    commands_dir.mkdir()
    (commands_dir / "lint.md").write_text(
        "# Lint Command\n\n" "Run linting tools on the codebase.\n"
    )

    # Create agent context files
    agents_dir = plugin_dir / "agents"
    agents_dir.mkdir()
    (agents_dir / "reviewer.md").write_text(
        "# Reviewer Agent\n\n" "Autonomous code review agent.\n"
    )

    # Create rule context files
    rules_dir = plugin_dir / "rules"
    rules_dir.mkdir()
    (rules_dir / "code-style.md").write_text(
        "# Code Style Rules\n\n" "Follow the project's coding style.\n"
    )

    # Create instruction context files
    instructions_dir = plugin_dir / "instructions"
    instructions_dir.mkdir()
    (instructions_dir / "lint.md").write_text(
        "# Lint Instructions\n\n" "Instructions for linting Python code.\n"
    )

    # Create agent_file context files
    agent_file_dir = plugin_dir / "agent_file"
    agent_file_dir.mkdir()
    (agent_file_dir / "content.md").write_text(
        "# Agent Instructions\n\n" "This content appears in CLAUDE.md or AGENTS.md.\n"
    )

    return plugin_dir


class TestClaudeCodeAdapterIntegration:
    """Integration tests for Claude Code adapter."""

    def test_full_installation(self, temp_project: Path, comprehensive_plugin: Path):
        """Claude Code: installs all components correctly."""
        project = Project.init(temp_project, "claude-code", "test-project")

        installer = PluginInstaller(project)
        summary = installer.install(
            {"comprehensive-plugin": PluginSpec(source=f"file:{comprehensive_plugin}")},
            use_lockfile=False,
        )

        assert summary.all_successful

        # Verify skills installed in .claude/skills/{plugin}-{skill}/SKILL.md
        skill1 = (
            temp_project / ".claude" / "skills" / "comprehensive-plugin-code-review" / "SKILL.md"
        )
        skill2 = temp_project / ".claude" / "skills" / "comprehensive-plugin-testing" / "SKILL.md"
        assert skill1.exists()
        assert skill2.exists()

        # Verify skill content has frontmatter and rendered template
        content = skill1.read_text()
        assert "---" in content  # Has frontmatter
        assert "comprehensive-plugin" in content
        assert "test-project" in content

        # Verify commands installed in .claude/commands/{plugin}-{command}.md
        command = temp_project / ".claude" / "commands" / "comprehensive-plugin-lint.md"
        assert command.exists()

        # Verify sub-agents installed in .claude/agents/{plugin}-{agent}.md
        agent = temp_project / ".claude" / "agents" / "comprehensive-plugin-reviewer.md"
        assert agent.exists()

        # Verify MCP config updated in .mcp.json
        mcp_config = temp_project / ".mcp.json"
        assert mcp_config.exists()
        config_data = json.loads(mcp_config.read_text())
        assert "mcpServers" in config_data
        assert "test-server" in config_data["mcpServers"]

    def test_skill_file_copy(self, temp_project: Path, comprehensive_plugin: Path):
        """Claude Code: copies associated files to skill directory."""
        project = Project.init(temp_project, "claude-code")
        installer = PluginInstaller(project)
        installer.install(
            {"comprehensive-plugin": PluginSpec(source=f"file:{comprehensive_plugin}")},
            use_lockfile=False,
        )

        # Verify config file copied to skill directory
        config = (
            temp_project
            / ".claude"
            / "skills"
            / "comprehensive-plugin-code-review"
            / "skills"
            / "config.json"
        )
        assert config.exists()


class TestCursorAdapterIntegration:
    """Integration tests for Cursor adapter."""

    def test_full_installation(self, temp_project: Path, comprehensive_plugin: Path):
        """Cursor: installs rules as MDC files."""
        project = Project.init(temp_project, "cursor", "test-project")

        installer = PluginInstaller(project)
        summary = installer.install(
            {"comprehensive-plugin": PluginSpec(source=f"file:{comprehensive_plugin}")},
            use_lockfile=False,
        )

        assert summary.all_successful

        # Verify rules installed as .cursor/rules/{plugin}-{rule}.mdc
        rule = temp_project / ".cursor" / "rules" / "comprehensive-plugin-code-style.mdc"
        assert rule.exists()

        # Verify frontmatter format
        content = rule.read_text()
        assert "---" in content
        assert "description:" in content
        # Cursor uses globs and alwaysApply in frontmatter
        assert "globs:" in content or "alwaysApply:" in content

    def test_skills_not_installed(self, temp_project: Path, comprehensive_plugin: Path):
        """Cursor: skills are NOT installed (not supported)."""
        project = Project.init(temp_project, "cursor")
        installer = PluginInstaller(project)
        installer.install(
            {"comprehensive-plugin": PluginSpec(source=f"file:{comprehensive_plugin}")},
            use_lockfile=False,
        )

        # Skills should NOT be installed - Cursor doesn't support skills
        skill1 = temp_project / ".cursor" / "rules" / "comprehensive-plugin-code-review.mdc"
        skill2 = temp_project / ".cursor" / "rules" / "comprehensive-plugin-testing.mdc"
        assert not skill1.exists(), "Skills should not be installed in Cursor"
        assert not skill2.exists(), "Skills should not be installed in Cursor"

    def test_ignores_unsupported_features(self, temp_project: Path, comprehensive_plugin: Path):
        """Cursor: ignores unsupported features like skills and subagents."""
        project = Project.init(temp_project, "cursor")
        installer = PluginInstaller(project)
        summary = installer.install(
            {"comprehensive-plugin": PluginSpec(source=f"file:{comprehensive_plugin}")},
            use_lockfile=False,
        )

        # Installation succeeds (unsupported features are logged but not installed)
        assert summary.all_successful

        # Verify no unsupported files were created
        # Subagents directory shouldn't exist for Cursor
        agents_dir = temp_project / ".cursor" / "agents"
        assert not agents_dir.exists()

        # Commands ARE supported - verify they're installed
        commands_dir = temp_project / ".cursor" / "commands"
        assert commands_dir.exists()
        assert any(commands_dir.iterdir())


class TestGitHubCopilotAdapterIntegration:
    """Integration tests for GitHub Copilot adapter."""

    def test_full_installation(self, temp_project: Path, comprehensive_plugin: Path):
        """GitHub Copilot: installs instructions as instruction files."""
        project = Project.init(temp_project, "github-copilot", "test-project")

        installer = PluginInstaller(project)
        summary = installer.install(
            {"comprehensive-plugin": PluginSpec(source=f"file:{comprehensive_plugin}")},
            use_lockfile=False,
        )

        assert summary.all_successful

        # Verify instructions installed as .github/instructions/{name}.instructions.md
        instruction = temp_project / ".github" / "instructions" / "lint.instructions.md"
        assert instruction.exists()

        # Verify content includes applyTo frontmatter since we specified it
        content = instruction.read_text()
        assert "Lint Instructions" in content
        assert "applyTo" in content

    def test_rules_installation(self, temp_project: Path, comprehensive_plugin: Path):
        """GitHub Copilot: installs rules as copilot-instructions.md."""
        project = Project.init(temp_project, "github-copilot")
        installer = PluginInstaller(project)
        installer.install(
            {"comprehensive-plugin": PluginSpec(source=f"file:{comprehensive_plugin}")},
            use_lockfile=False,
        )

        # Verify rules installed in .github/copilot-instructions.md
        rules = temp_project / ".github" / "copilot-instructions.md"
        assert rules.exists()

        content = rules.read_text()
        assert "Code Style Rules" in content


class TestAntigravityAdapterIntegration:
    """Integration tests for Antigravity adapter."""

    def test_full_installation(self, temp_project: Path, comprehensive_plugin: Path):
        """Antigravity: installs skills in universal format."""
        project = Project.init(temp_project, "antigravity", "test-project")

        installer = PluginInstaller(project)
        summary = installer.install(
            {"comprehensive-plugin": PluginSpec(source=f"file:{comprehensive_plugin}")},
            use_lockfile=False,
        )

        assert summary.all_successful

        # Verify skills installed as .agent/skills/{skill}/SKILL.md
        skill1 = temp_project / ".agent" / "skills" / "code-review" / "SKILL.md"
        skill2 = temp_project / ".agent" / "skills" / "testing" / "SKILL.md"
        assert skill1.exists()
        assert skill2.exists()

        # Verify frontmatter format
        content = skill1.read_text()
        assert "---" in content
        assert "name:" in content
        assert "description:" in content

    def test_warns_about_unsupported_features(self, temp_project: Path, comprehensive_plugin: Path):
        """Antigravity: warns about commands and subagents (MCP is supported)."""
        project = Project.init(temp_project, "antigravity")
        installer = PluginInstaller(project)
        summary = installer.install(
            {"comprehensive-plugin": PluginSpec(source=f"file:{comprehensive_plugin}")},
            use_lockfile=False,
        )

        # Installation succeeds but has warnings
        assert summary.all_successful
        warnings = []
        for result in summary.results:
            warnings.extend(result.warnings)
        # Should warn about commands and subagents (MCP is now supported)
        assert len(warnings) >= 2


class TestCodexAdapterIntegration:
    """Integration tests for OpenAI Codex adapter."""

    def test_full_installation(self, temp_project: Path, comprehensive_plugin: Path):
        """Codex: installs skills in .codex/skills format."""
        project = Project.init(temp_project, "codex", "test-project")

        installer = PluginInstaller(project)
        summary = installer.install(
            {"comprehensive-plugin": PluginSpec(source=f"file:{comprehensive_plugin}")},
            use_lockfile=False,
        )

        assert summary.all_successful

        # Verify skills installed as .codex/skills/{skill}/SKILL.md
        skill1 = temp_project / ".codex" / "skills" / "code-review" / "SKILL.md"
        skill2 = temp_project / ".codex" / "skills" / "testing" / "SKILL.md"
        assert skill1.exists()
        assert skill2.exists()

        # Verify frontmatter format
        content = skill1.read_text()
        assert "---" in content
        assert "name:" in content

    def test_agent_file_installation(self, temp_project: Path, comprehensive_plugin: Path):
        """Codex: installs agent_file content to AGENTS.md at project root."""
        project = Project.init(temp_project, "codex")
        installer = PluginInstaller(project)
        installer.install(
            {"comprehensive-plugin": PluginSpec(source=f"file:{comprehensive_plugin}")},
            use_lockfile=False,
        )

        # Verify agent_file content installed as AGENTS.md at project root
        agents_md = temp_project / "AGENTS.md"
        assert agents_md.exists()

        content = agents_md.read_text()
        assert "Agent Instructions" in content
        # Verify plugin markers are present
        assert "dex:plugin:comprehensive-plugin" in content


class TestCrossAdapterConsistency:
    """Tests for consistent behavior across all adapters."""

    @pytest.mark.parametrize(
        "agent_type",
        ["claude-code", "cursor", "github-copilot", "antigravity", "codex"],
    )
    def test_adapter_loads(self, agent_type: str):
        """All registered adapters can be loaded."""
        adapter = get_adapter(agent_type)
        assert adapter is not None
        assert adapter.metadata.name == agent_type

    @pytest.mark.parametrize(
        "agent_type",
        ["claude-code", "cursor", "github-copilot", "antigravity", "codex"],
    )
    def test_project_initialization(self, temp_project: Path, agent_type: str):
        """Projects can be initialized for all adapters."""
        project = Project.init(
            temp_project / agent_type, agent_type, "test-project"  # type: ignore[arg-type]
        )
        assert project.agent == agent_type

    @pytest.mark.parametrize(
        "agent_type",
        ["claude-code", "cursor", "github-copilot", "antigravity", "codex"],
    )
    def test_skill_installation_succeeds(
        self, temp_dir: Path, agent_type: str, comprehensive_plugin: Path
    ):
        """Skill installation succeeds for all adapters."""
        project_dir = temp_dir / f"project-{agent_type}"
        project_dir.mkdir()
        project = Project.init(project_dir, agent_type, "test-project")  # type: ignore[arg-type]

        installer = PluginInstaller(project)
        summary = installer.install(
            {"comprehensive-plugin": PluginSpec(source=f"file:{comprehensive_plugin}")},
            use_lockfile=False,
        )

        assert summary.all_successful, f"Failed for {agent_type}"
        assert summary.success_count >= 1, f"No successful installs for {agent_type}"

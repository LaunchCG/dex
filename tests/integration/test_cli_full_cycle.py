"""Comprehensive CLI integration tests for install/remove cycles.

These tests validate the full install and remove workflow through the CLI,
ensuring that files are properly created on install and removed on uninstall.

Tests cover:
- Full install/remove cycle for all adapters
- --save option for adding plugins to dex.yaml
- --registry option for specifying registry
- MCP config cleanup on removal (Claude Code)
- Manifest tracking and cleanup
"""

import json
from pathlib import Path

import pytest
import yaml
from typer.testing import CliRunner

from dex.cli.main import app


@pytest.fixture
def runner():
    """Get a CLI test runner."""
    return CliRunner()


@pytest.fixture
def comprehensive_plugin(temp_dir: Path) -> Path:
    """Create a comprehensive plugin for testing."""
    plugin_dir = temp_dir / "cli-test-plugin"
    plugin_dir.mkdir()

    manifest = {
        "name": "cli-test-plugin",
        "version": "1.0.0",
        "description": "Plugin for CLI testing",
        "skills": [
            {
                "name": "test-skill",
                "description": "A test skill",
                "context": "./skills/test.md",
            }
        ],
        "commands": [
            {
                "name": "test-command",
                "description": "A test command",
                "context": "./commands/test.md",
            }
        ],
        "sub_agents": [
            {
                "name": "test-agent",
                "description": "A test agent",
                "context": "./agents/test.md",
            }
        ],
        "mcp_servers": [
            {
                "name": "test-mcp-server",
                "type": "remote",
                "source": "npm:@test/mcp-server",
            }
        ],
        "rules": [
            {
                "name": "test-rule",
                "description": "A test rule",
                "context": "./rules/test.md",
            }
        ],
        "instructions": [
            {
                "name": "test-instruction",
                "description": "A test instruction for Python files",
                "context": "./instructions/test.md",
                "applyTo": "**/*.py",
            }
        ],
        "agent_file": {
            "context": "./agent_file/content.md",
        },
    }
    (plugin_dir / "package.json").write_text(json.dumps(manifest, indent=2))

    # Create context files
    skills_dir = plugin_dir / "skills"
    skills_dir.mkdir()
    (skills_dir / "test.md").write_text("# Test Skill\n\nSkill content here.\n")

    commands_dir = plugin_dir / "commands"
    commands_dir.mkdir()
    (commands_dir / "test.md").write_text("# Test Command\n\nCommand content here.\n")

    agents_dir = plugin_dir / "agents"
    agents_dir.mkdir()
    (agents_dir / "test.md").write_text("# Test Agent\n\nAgent content here.\n")

    rules_dir = plugin_dir / "rules"
    rules_dir.mkdir()
    (rules_dir / "test.md").write_text("# Test Rule\n\nRule content here.\n")

    instructions_dir = plugin_dir / "instructions"
    instructions_dir.mkdir()
    (instructions_dir / "test.md").write_text("# Test Instruction\n\nInstruction content here.\n")

    agent_file_dir = plugin_dir / "agent_file"
    agent_file_dir.mkdir()
    (agent_file_dir / "content.md").write_text("# Agent File Content\n\nAgent file content here.\n")

    return plugin_dir


# =============================================================================
# Claude Code Full Cycle Tests
# =============================================================================


class TestClaudeCodeFullCycle:
    """Full install/remove cycle tests for Claude Code."""

    def test_install_creates_all_files(
        self, runner: CliRunner, temp_dir: Path, comprehensive_plugin: Path
    ):
        """Install creates all expected files for Claude Code."""
        project_dir = temp_dir / "claude-install-test"
        project_dir.mkdir()

        # Init project
        runner.invoke(app, ["init", "--agent", "claude-code", "--path", str(project_dir)])

        # Install plugin
        result = runner.invoke(
            app,
            [
                "install",
                "--source",
                f"file:{comprehensive_plugin}",
                "--path",
                str(project_dir),
            ],
        )

        assert result.exit_code == 0, f"Install failed: {result.output}"

        # Verify skill created
        skill_file = project_dir / ".claude" / "skills" / "cli-test-plugin-test-skill" / "SKILL.md"
        assert skill_file.exists(), "Skill file not created"
        assert "# Test Skill" in skill_file.read_text()

        # Verify command created
        command_file = project_dir / ".claude" / "commands" / "cli-test-plugin-test-command.md"
        assert command_file.exists(), "Command file not created"
        assert "# Test Command" in command_file.read_text()

        # Verify agent created
        agent_file = project_dir / ".claude" / "agents" / "cli-test-plugin-test-agent.md"
        assert agent_file.exists(), "Agent file not created"
        assert "# Test Agent" in agent_file.read_text()

        # Verify MCP config updated
        mcp_config = project_dir / ".mcp.json"
        assert mcp_config.exists(), "MCP config not created"
        mcp_data = json.loads(mcp_config.read_text())
        assert "mcpServers" in mcp_data
        assert "test-mcp-server" in mcp_data["mcpServers"]

    def test_uninstall_deletes_all_files(
        self, runner: CliRunner, temp_dir: Path, comprehensive_plugin: Path
    ):
        """Uninstall deletes all files created during install."""
        project_dir = temp_dir / "claude-uninstall-test"
        project_dir.mkdir()

        # Init and install
        runner.invoke(app, ["init", "--agent", "claude-code", "--path", str(project_dir)])
        runner.invoke(
            app,
            [
                "install",
                "--source",
                f"file:{comprehensive_plugin}",
                "--save",
                "--path",
                str(project_dir),
            ],
        )

        # Verify files exist before uninstall
        skill_dir = project_dir / ".claude" / "skills" / "cli-test-plugin-test-skill"
        assert skill_dir.exists(), "Skill dir should exist before uninstall"

        # Uninstall plugin with --remove to also remove from dex.yaml
        result = runner.invoke(
            app, ["uninstall", "cli-test-plugin", "--remove", "--path", str(project_dir)]
        )
        assert result.exit_code == 0, f"Uninstall failed: {result.output}"

        # Verify skill directory removed
        assert not skill_dir.exists(), "Skill directory should be removed"

        # Verify command removed
        command_file = project_dir / ".claude" / "commands" / "cli-test-plugin-test-command.md"
        assert not command_file.exists(), "Command file should be removed"

        # Verify agent removed
        agent_file = project_dir / ".claude" / "agents" / "cli-test-plugin-test-agent.md"
        assert not agent_file.exists(), "Agent file should be removed"

        # Verify MCP server removed from config
        mcp_config = project_dir / ".mcp.json"
        if mcp_config.exists():
            mcp_data = json.loads(mcp_config.read_text())
            if "mcpServers" in mcp_data:
                assert (
                    "test-mcp-server" not in mcp_data["mcpServers"]
                ), "MCP server should be removed from config"

        # Verify removed from dex.yaml (because --remove was used)
        config = yaml.safe_load((project_dir / "dex.yaml").read_text())
        assert "cli-test-plugin" not in config.get("plugins", {})


class TestSaveOption:
    """Tests for --save option."""

    def test_save_adds_plugin_to_sdlc_json(
        self, runner: CliRunner, temp_dir: Path, comprehensive_plugin: Path
    ):
        """--save adds installed plugin to dex.yaml."""
        project_dir = temp_dir / "save-test"
        project_dir.mkdir()

        runner.invoke(app, ["init", "--agent", "claude-code", "--path", str(project_dir)])

        # Install with --save
        result = runner.invoke(
            app,
            [
                "install",
                "--source",
                f"file:{comprehensive_plugin}",
                "--save",
                "--path",
                str(project_dir),
            ],
        )

        assert result.exit_code == 0

        # Verify plugin added to dex.yaml
        config = yaml.safe_load((project_dir / "dex.yaml").read_text())
        assert "cli-test-plugin" in config.get("plugins", {}), "Plugin should be saved"

    def test_install_without_save_does_not_modify_sdlc_json(
        self, runner: CliRunner, temp_dir: Path, comprehensive_plugin: Path
    ):
        """Install without --save does not modify dex.yaml plugins."""
        project_dir = temp_dir / "no-save-test"
        project_dir.mkdir()

        runner.invoke(app, ["init", "--agent", "claude-code", "--path", str(project_dir)])

        # Install without --save
        result = runner.invoke(
            app,
            [
                "install",
                "--source",
                f"file:{comprehensive_plugin}",
                "--path",
                str(project_dir),
            ],
        )

        assert result.exit_code == 0

        # Verify plugin NOT added to dex.yaml
        config = yaml.safe_load((project_dir / "dex.yaml").read_text())
        assert "cli-test-plugin" not in config.get(
            "plugins", {}
        ), "Plugin should not be saved without --save"


# =============================================================================
# Cursor Full Cycle Tests
# =============================================================================


class TestCursorFullCycle:
    """Full install/remove cycle tests for Cursor."""

    def test_install_creates_rule_file(
        self, runner: CliRunner, temp_dir: Path, comprehensive_plugin: Path
    ):
        """Install creates rule file for Cursor."""
        project_dir = temp_dir / "cursor-install-test"
        project_dir.mkdir()

        runner.invoke(app, ["init", "--agent", "cursor", "--path", str(project_dir)])

        result = runner.invoke(
            app,
            [
                "install",
                "--source",
                f"file:{comprehensive_plugin}",
                "--path",
                str(project_dir),
            ],
        )

        assert result.exit_code == 0

        # Verify rule file created (rules are supported, skills are not)
        rule_file = project_dir / ".cursor" / "rules" / "cli-test-plugin-test-rule.mdc"
        assert rule_file.exists(), "Rule file not created"
        content = rule_file.read_text()
        assert "---" in content  # Has frontmatter
        assert "description:" in content
        assert "# Test Rule" in content

        # Verify skill file NOT created (Cursor doesn't support skills)
        skill_file = project_dir / ".cursor" / "rules" / "cli-test-plugin-test-skill.mdc"
        assert not skill_file.exists(), "Skills should not be installed in Cursor"

    def test_remove_deletes_rule_file(
        self, runner: CliRunner, temp_dir: Path, comprehensive_plugin: Path
    ):
        """Remove deletes rule file for Cursor."""
        project_dir = temp_dir / "cursor-remove-test"
        project_dir.mkdir()

        runner.invoke(app, ["init", "--agent", "cursor", "--path", str(project_dir)])
        runner.invoke(
            app,
            [
                "install",
                "--source",
                f"file:{comprehensive_plugin}",
                "--save",
                "--path",
                str(project_dir),
            ],
        )

        rule_file = project_dir / ".cursor" / "rules" / "cli-test-plugin-test-rule.mdc"
        assert rule_file.exists()

        # Remove
        result = runner.invoke(
            app, ["uninstall", "cli-test-plugin", "--remove", "--path", str(project_dir)]
        )
        assert result.exit_code == 0

        assert not rule_file.exists(), "Rule file should be removed"


# =============================================================================
# GitHub Copilot Full Cycle Tests
# =============================================================================


class TestGitHubCopilotFullCycle:
    """Full install/remove cycle tests for GitHub Copilot."""

    def test_install_creates_instruction_file(
        self, runner: CliRunner, temp_dir: Path, comprehensive_plugin: Path
    ):
        """Install creates instruction file for GitHub Copilot."""
        project_dir = temp_dir / "copilot-install-test"
        project_dir.mkdir()

        runner.invoke(app, ["init", "--agent", "github-copilot", "--path", str(project_dir)])

        result = runner.invoke(
            app,
            [
                "install",
                "--source",
                f"file:{comprehensive_plugin}",
                "--path",
                str(project_dir),
            ],
        )

        assert result.exit_code == 0

        # Verify instruction file created (from instructions config)
        instruction_file = (
            project_dir / ".github" / "instructions" / "test-instruction.instructions.md"
        )
        assert instruction_file.exists(), "Instruction file not created"
        assert "# Test Instruction" in instruction_file.read_text()

        # Verify rules file created
        rules_file = project_dir / ".github" / "copilot-instructions.md"
        assert rules_file.exists(), "Rules file not created"
        assert "# Test Rule" in rules_file.read_text()

    def test_remove_deletes_files(
        self, runner: CliRunner, temp_dir: Path, comprehensive_plugin: Path
    ):
        """Remove deletes instruction and rules files."""
        project_dir = temp_dir / "copilot-remove-test"
        project_dir.mkdir()

        runner.invoke(app, ["init", "--agent", "github-copilot", "--path", str(project_dir)])
        runner.invoke(
            app,
            [
                "install",
                "--source",
                f"file:{comprehensive_plugin}",
                "--save",
                "--path",
                str(project_dir),
            ],
        )

        instruction_file = (
            project_dir / ".github" / "instructions" / "test-instruction.instructions.md"
        )
        assert instruction_file.exists()

        result = runner.invoke(
            app, ["uninstall", "cli-test-plugin", "--remove", "--path", str(project_dir)]
        )
        assert result.exit_code == 0

        assert not instruction_file.exists(), "Instruction file should be removed"


# =============================================================================
# Antigravity Full Cycle Tests
# =============================================================================


class TestAntigravityFullCycle:
    """Full install/remove cycle tests for Antigravity."""

    def test_install_creates_skill_file(
        self, runner: CliRunner, temp_dir: Path, comprehensive_plugin: Path
    ):
        """Install creates skill file for Antigravity."""
        project_dir = temp_dir / "antigravity-install-test"
        project_dir.mkdir()

        runner.invoke(app, ["init", "--agent", "antigravity", "--path", str(project_dir)])

        result = runner.invoke(
            app,
            [
                "install",
                "--source",
                f"file:{comprehensive_plugin}",
                "--path",
                str(project_dir),
            ],
        )

        assert result.exit_code == 0

        # Verify skill file created (note: skill name only, not plugin-namespaced)
        skill_file = project_dir / ".agent" / "skills" / "test-skill" / "SKILL.md"
        assert skill_file.exists(), "Skill file not created"
        content = skill_file.read_text()
        assert "---" in content
        assert "name: test-skill" in content

    def test_remove_deletes_skill_directory(
        self, runner: CliRunner, temp_dir: Path, comprehensive_plugin: Path
    ):
        """Remove deletes skill directory for Antigravity."""
        project_dir = temp_dir / "antigravity-remove-test"
        project_dir.mkdir()

        runner.invoke(app, ["init", "--agent", "antigravity", "--path", str(project_dir)])
        runner.invoke(
            app,
            [
                "install",
                "--source",
                f"file:{comprehensive_plugin}",
                "--save",
                "--path",
                str(project_dir),
            ],
        )

        skill_dir = project_dir / ".agent" / "skills" / "test-skill"
        assert skill_dir.exists()

        result = runner.invoke(
            app, ["uninstall", "cli-test-plugin", "--remove", "--path", str(project_dir)]
        )
        assert result.exit_code == 0

        assert not skill_dir.exists(), "Skill directory should be removed"


# =============================================================================
# Codex Full Cycle Tests
# =============================================================================


class TestCodexFullCycle:
    """Full install/remove cycle tests for Codex."""

    def test_install_creates_skill_and_agents_md(
        self, runner: CliRunner, temp_dir: Path, comprehensive_plugin: Path
    ):
        """Install creates skill file and AGENTS.md for Codex."""
        project_dir = temp_dir / "codex-install-test"
        project_dir.mkdir()

        runner.invoke(app, ["init", "--agent", "codex", "--path", str(project_dir)])

        result = runner.invoke(
            app,
            [
                "install",
                "--source",
                f"file:{comprehensive_plugin}",
                "--path",
                str(project_dir),
            ],
        )

        assert result.exit_code == 0

        # Verify skill file created
        skill_file = project_dir / ".codex" / "skills" / "test-skill" / "SKILL.md"
        assert skill_file.exists(), "Skill file not created"

        # Verify AGENTS.md created (from agent_file config)
        agents_file = project_dir / "AGENTS.md"
        assert agents_file.exists(), "AGENTS.md not created"
        assert "# Agent File Content" in agents_file.read_text()

    def test_remove_deletes_files(
        self, runner: CliRunner, temp_dir: Path, comprehensive_plugin: Path
    ):
        """Remove deletes skill directory and AGENTS.md."""
        project_dir = temp_dir / "codex-remove-test"
        project_dir.mkdir()

        runner.invoke(app, ["init", "--agent", "codex", "--path", str(project_dir)])
        runner.invoke(
            app,
            [
                "install",
                "--source",
                f"file:{comprehensive_plugin}",
                "--save",
                "--path",
                str(project_dir),
            ],
        )

        skill_dir = project_dir / ".codex" / "skills" / "test-skill"
        assert skill_dir.exists()

        result = runner.invoke(
            app, ["uninstall", "cli-test-plugin", "--remove", "--path", str(project_dir)]
        )
        assert result.exit_code == 0

        assert not skill_dir.exists(), "Skill directory should be removed"


# =============================================================================
# Version Specifier Tests via CLI
# =============================================================================


class TestCLIVersionSpecifiers:
    """Test version specifier parsing via CLI."""

    def test_install_with_exact_version(self, runner: CliRunner, temp_dir: Path):
        """CLI parses plugin@1.0.0 correctly."""
        project_dir = temp_dir / "version-test"
        project_dir.mkdir()

        runner.invoke(app, ["init", "--agent", "claude-code", "--path", str(project_dir)])

        # This will fail (no registry) but tests parsing
        result = runner.invoke(
            app,
            [
                "install",
                "test-plugin@1.0.0",
                "--path",
                str(project_dir),
            ],
            catch_exceptions=False,
        )

        # Should fail due to no registry, but parsing worked
        # The error should not be about parsing
        assert "invalid" not in result.output.lower() or "version" not in result.output.lower()

    def test_install_with_caret_version(self, runner: CliRunner, temp_dir: Path):
        """CLI parses plugin@^1.0.0 correctly."""
        project_dir = temp_dir / "caret-version-test"
        project_dir.mkdir()

        runner.invoke(app, ["init", "--agent", "claude-code", "--path", str(project_dir)])

        result = runner.invoke(
            app,
            [
                "install",
                "test-plugin@^1.0.0",
                "--path",
                str(project_dir),
            ],
            catch_exceptions=False,
        )

        # Parsing should work - will fail at resolution
        assert "parse" not in result.output.lower()


# =============================================================================
# Registry Option Tests
# =============================================================================


class TestRegistryOption:
    """Test --registry option."""

    def test_registry_option_with_valid_package(
        self, runner: CliRunner, temp_dir: Path, comprehensive_plugin: Path
    ):
        """--registry option successfully installs from specified registry."""
        import tarfile

        project_dir = temp_dir / "registry-option-test"
        project_dir.mkdir()

        # Create a test registry with the plugin
        registry_dir = temp_dir / "test-registry"
        registry_dir.mkdir()
        registry_data = {
            "packages": {
                "cli-test-plugin": {
                    "versions": ["1.0.0"],
                    "latest": "1.0.0",
                }
            }
        }
        (registry_dir / "registry.json").write_text(json.dumps(registry_data))

        # Create tarball
        tarball_path = registry_dir / "cli-test-plugin-1.0.0.tar.gz"
        with tarfile.open(tarball_path, "w:gz") as tar:
            tar.add(comprehensive_plugin, arcname="cli-test-plugin")

        # Init project
        runner.invoke(app, ["init", "--agent", "claude-code", "--path", str(project_dir)])

        # Install with --registry override
        result = runner.invoke(
            app,
            [
                "install",
                "cli-test-plugin@1.0.0",
                "--registry",
                f"file://{registry_dir}",
                "--path",
                str(project_dir),
            ],
        )

        assert result.exit_code == 0, f"Install failed: {result.output}"

        # Verify plugin was installed from the specified registry
        skill_file = project_dir / ".claude" / "skills" / "cli-test-plugin-test-skill" / "SKILL.md"
        assert skill_file.exists(), "Plugin should be installed from registry"

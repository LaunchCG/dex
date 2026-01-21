"""Tests for dex.adapters.github_copilot module."""

from pathlib import Path

import pytest

from dex.adapters.github_copilot import GitHubCopilotAdapter
from dex.config.schemas import (
    CommandConfig,
    InstructionConfig,
    MCPServerConfig,
    PluginManifest,
    SkillConfig,
    SubAgentConfig,
)


@pytest.fixture
def adapter():
    """Get a GitHubCopilotAdapter instance."""
    return GitHubCopilotAdapter()


@pytest.fixture
def sample_manifest():
    """Sample plugin manifest for testing."""
    return PluginManifest(
        name="test-plugin",
        version="1.0.0",
        description="A test plugin",
    )


@pytest.fixture
def sample_skill():
    """Sample skill configuration."""
    return SkillConfig(
        name="test-skill",
        description="A test skill",
        context="./context/skill.md",
    )


@pytest.fixture
def sample_command():
    """Sample command configuration."""
    return CommandConfig(
        name="test-command",
        description="A test command for Python files",
        context="./context/command.md",
    )


@pytest.fixture
def sample_subagent():
    """Sample sub-agent configuration."""
    return SubAgentConfig(
        name="test-agent",
        description="A test agent",
        context="./context/agent.md",
    )


class TestGitHubCopilotAdapterMetadata:
    """Tests for GitHubCopilotAdapter.metadata property."""

    def test_returns_metadata(self, adapter: GitHubCopilotAdapter):
        """Returns AdapterMetadata with correct values."""
        meta = adapter.metadata
        assert meta.name == "github-copilot"
        assert meta.display_name == "GitHub Copilot"
        assert meta.mcp_config_file == ".vscode/mcp.json"


class TestGitHubCopilotAdapterDirectories:
    """Tests for GitHubCopilotAdapter directory methods."""

    def test_get_base_directory(self, adapter: GitHubCopilotAdapter, temp_dir: Path):
        """Returns .github directory."""
        result = adapter.get_base_directory(temp_dir)
        assert result == temp_dir / ".github"

    def test_get_mcp_config_path(self, adapter: GitHubCopilotAdapter, temp_dir: Path):
        """Returns .vscode/mcp.json path."""
        result = adapter.get_mcp_config_path(temp_dir)
        assert result == temp_dir / ".vscode" / "mcp.json"


class TestGitHubCopilotAdapterValidatePluginCompatibility:
    """Tests for GitHubCopilotAdapter.validate_plugin_compatibility()."""

    def test_no_warnings_for_mcp_servers(self, adapter: GitHubCopilotAdapter):
        """No warnings for MCP servers (now supported)."""
        from dex.config.schemas import MCPServerConfig

        manifest = PluginManifest(
            name="test-plugin",
            version="1.0.0",
            description="Plugin with MCP",
            mcp_servers=[MCPServerConfig(name="test-server", type="command", source="npm:test")],
        )

        warnings = adapter.validate_plugin_compatibility(manifest)

        # MCP is now supported - no warnings for MCP servers
        assert not any("MCP" in w for w in warnings)

    def test_warns_about_skills(self, adapter: GitHubCopilotAdapter):
        """Warns when plugin has skills (not directly supported)."""
        manifest = PluginManifest(
            name="test-plugin",
            version="1.0.0",
            description="Plugin with skills",
            skills=[SkillConfig(name="skill", description="Skill", context="./ctx.md")],
        )

        warnings = adapter.validate_plugin_compatibility(manifest)

        # GitHub Copilot now supports skills - no warnings expected
        assert warnings == []

    def test_no_warnings_for_subagents(self, adapter: GitHubCopilotAdapter):
        """No warnings when plugin has subagents (now supported as agents)."""
        manifest = PluginManifest(
            name="test-plugin",
            version="1.0.0",
            description="Plugin with agents",
            sub_agents=[SubAgentConfig(name="agent", description="Agent", context="./ctx.md")],
        )

        warnings = adapter.validate_plugin_compatibility(manifest)

        # GitHub Copilot now supports agents - no warnings expected
        assert warnings == []

    def test_no_warnings_for_commands_only(
        self, adapter: GitHubCopilotAdapter, sample_manifest: PluginManifest
    ):
        """No warnings for plugin with only commands/rules."""
        warnings = adapter.validate_plugin_compatibility(sample_manifest)
        assert warnings == []


class TestGitHubCopilotAdapterPreInstall:
    """Tests for GitHubCopilotAdapter.pre_install()."""

    def test_creates_directories(
        self, adapter: GitHubCopilotAdapter, temp_dir: Path, sample_manifest: PluginManifest
    ):
        """Creates .github and instructions directories."""
        adapter.pre_install(temp_dir, [sample_manifest])

        assert (temp_dir / ".github").exists()
        assert (temp_dir / ".github" / "instructions").exists()

    def test_handles_existing_directories(
        self, adapter: GitHubCopilotAdapter, temp_dir: Path, sample_manifest: PluginManifest
    ):
        """Handles existing directories without error."""
        (temp_dir / ".github" / "instructions").mkdir(parents=True)

        # Should not raise
        adapter.pre_install(temp_dir, [sample_manifest])


class TestGitHubCopilotAdapterMCPConfig:
    """Tests for GitHubCopilotAdapter MCP configuration."""

    def test_generate_mcp_config_npm(
        self, adapter: GitHubCopilotAdapter, temp_dir: Path, sample_manifest: PluginManifest
    ):
        """Generates MCP config for npm source shortcut."""
        from dex.config.schemas import MCPServerConfig

        mcp_server = MCPServerConfig(
            name="test-server",
            type="command",
            source="npm:@example/mcp-server",
        )
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        config = adapter.generate_mcp_config(mcp_server, sample_manifest, temp_dir, source_dir)

        expected = {
            "test-server": {
                "command": "npx",
                "args": ["-y", "@example/mcp-server"],
            }
        }
        assert config == expected

    def test_generate_mcp_config_uvx(
        self, adapter: GitHubCopilotAdapter, temp_dir: Path, sample_manifest: PluginManifest
    ):
        """Generates MCP config for uvx source shortcut."""
        from dex.config.schemas import MCPServerConfig

        mcp_server = MCPServerConfig(
            name="python-server",
            type="command",
            source="uvx:mcp-server-package",
        )
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        config = adapter.generate_mcp_config(mcp_server, sample_manifest, temp_dir, source_dir)

        expected = {
            "python-server": {
                "command": "uvx",
                "args": ["--from", "mcp-server-package"],
            }
        }
        assert config == expected

    def test_merge_mcp_config(self, adapter: GitHubCopilotAdapter):
        """Merges MCP config into mcpServers object."""
        existing: dict[str, object] = {}
        new_entries = {"test-server": {"command": "npx", "args": ["-y", "test"]}}

        result = adapter.merge_mcp_config(existing, new_entries)

        expected = {"mcpServers": {"test-server": {"command": "npx", "args": ["-y", "test"]}}}
        assert result == expected

    def test_generates_config_with_source_and_extra_args(
        self,
        adapter: GitHubCopilotAdapter,
        temp_dir: Path,
        sample_manifest: PluginManifest,
    ):
        """Generates config with uvx source and additional args."""
        mcp_server = MCPServerConfig(
            name="serena",
            type="command",
            source="uvx:git+https://github.com/oraios/serena",
            args=[
                "serena",
                "start-mcp-server",
                "--context",
                "ide-assistant",
                "--project",
                "${PWD}",
            ],
        )
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        config = adapter.generate_mcp_config(
            mcp_server=mcp_server,
            plugin=sample_manifest,
            project_root=temp_dir,
            source_dir=source_dir,
        )

        expected = {
            "serena": {
                "command": "uvx",
                "args": [
                    "--from",
                    "git+https://github.com/oraios/serena",
                    "serena",
                    "start-mcp-server",
                    "--context",
                    "ide-assistant",
                    "--project",
                    "${PWD}",
                ],
            }
        }
        assert config == expected

    def test_generates_http_server_config(
        self,
        adapter: GitHubCopilotAdapter,
        temp_dir: Path,
        sample_manifest: PluginManifest,
    ):
        """Generates config for HTTPS-based MCP server."""
        mcp_server = MCPServerConfig(
            name="atlassian",
            type="http",
            url="https://mcp.atlassian.com/v1/mcp",
        )
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        config = adapter.generate_mcp_config(
            mcp_server=mcp_server,
            plugin=sample_manifest,
            project_root=temp_dir,
            source_dir=source_dir,
        )

        expected = {
            "atlassian": {
                "url": "https://mcp.atlassian.com/v1/mcp",
            }
        }
        assert config == expected

    def test_generates_http_server_config_insecure(
        self,
        adapter: GitHubCopilotAdapter,
        temp_dir: Path,
        sample_manifest: PluginManifest,
    ):
        """Generates config for HTTP-based MCP server (non-HTTPS)."""
        mcp_server = MCPServerConfig(
            name="local-server",
            type="http",
            url="http://localhost:8080/mcp",
        )
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        config = adapter.generate_mcp_config(
            mcp_server=mcp_server,
            plugin=sample_manifest,
            project_root=temp_dir,
            source_dir=source_dir,
        )

        expected = {
            "local-server": {
                "url": "http://localhost:8080/mcp",
            }
        }
        assert config == expected


class TestGitHubCopilotAdapterInstructionFrontmatter:
    """Tests for GitHubCopilotAdapter.generate_instruction_frontmatter()."""

    def test_generates_frontmatter_with_apply_to_and_description(
        self,
        adapter: GitHubCopilotAdapter,
        sample_manifest: PluginManifest,
    ):
        """Generates frontmatter with applyTo and description fields."""
        instruction = InstructionConfig(
            name="test-instruction",
            description="Instructions for Python files",
            context="./instructions/python.md",
            applyTo="**/*.py",
        )

        frontmatter = adapter.generate_instruction_frontmatter(instruction, sample_manifest)

        expected = """---
applyTo: "**/*.py"
description: "Instructions for Python files"
---
"""
        assert frontmatter == expected

    def test_includes_exclude_agent_when_specified(
        self,
        adapter: GitHubCopilotAdapter,
        sample_manifest: PluginManifest,
    ):
        """Includes excludeAgent field when specified."""
        instruction = InstructionConfig(
            name="test-instruction",
            description="Code review instructions",
            context="./instructions/review.md",
            applyTo="**/*.ts",
            excludeAgent="code-review",
        )

        frontmatter = adapter.generate_instruction_frontmatter(instruction, sample_manifest)

        expected = """---
applyTo: "**/*.ts"
description: "Code review instructions"
excludeAgent: "code-review"
---
"""
        assert frontmatter == expected

    def test_handles_list_of_apply_to_patterns(
        self,
        adapter: GitHubCopilotAdapter,
        sample_manifest: PluginManifest,
    ):
        """Joins multiple applyTo patterns with comma."""
        instruction = InstructionConfig(
            name="test-instruction",
            description="Multi-language instructions",
            context="./instructions/multi.md",
            applyTo=["**/*.py", "**/*.js", "**/*.ts"],
        )

        frontmatter = adapter.generate_instruction_frontmatter(instruction, sample_manifest)

        expected = """---
applyTo: "**/*.py,**/*.js,**/*.ts"
description: "Multi-language instructions"
---
"""
        assert frontmatter == expected

    def test_generates_minimal_frontmatter_without_apply_to(
        self,
        adapter: GitHubCopilotAdapter,
        sample_manifest: PluginManifest,
    ):
        """Generates frontmatter with just description when no applyTo."""
        instruction = InstructionConfig(
            name="test-instruction",
            description="General instructions",
            context="./instructions/general.md",
        )

        frontmatter = adapter.generate_instruction_frontmatter(instruction, sample_manifest)

        expected = """---
description: "General instructions"
---
"""
        assert frontmatter == expected

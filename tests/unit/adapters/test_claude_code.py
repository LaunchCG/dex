"""Tests for dex.adapters.claude_code module."""

from pathlib import Path

import pytest

from dex.adapters.claude_code import ClaudeCodeAdapter
from dex.config.schemas import (
    ClaudeSettingsConfig,
    CommandConfig,
    FileTarget,
    MCPServerConfig,
    PluginManifest,
    RuleConfig,
    SkillConfig,
    SubAgentConfig,
)


@pytest.fixture
def adapter():
    """Get a ClaudeCodeAdapter instance."""
    return ClaudeCodeAdapter()


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
        description="A test command",
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


class TestClaudeCodeAdapterMetadata:
    """Tests for ClaudeCodeAdapter.metadata property."""

    def test_returns_metadata(self, adapter: ClaudeCodeAdapter):
        """Returns AdapterMetadata."""
        meta = adapter.metadata
        assert meta.name == "claude-code"
        assert meta.display_name == "Claude Code"
        assert meta.mcp_config_file == ".mcp.json"


class TestClaudeCodeAdapterDirectories:
    """Tests for ClaudeCodeAdapter directory methods."""

    def test_get_base_directory(self, adapter: ClaudeCodeAdapter, temp_dir: Path):
        """Returns .claude directory."""
        result = adapter.get_base_directory(temp_dir)
        assert result == temp_dir / ".claude"

    def test_get_skills_directory(self, adapter: ClaudeCodeAdapter, temp_dir: Path):
        """Returns .claude/skills directory."""
        result = adapter.get_skills_directory(temp_dir)
        assert result == temp_dir / ".claude" / "skills"

    def test_get_mcp_config_path(self, adapter: ClaudeCodeAdapter, temp_dir: Path):
        """Returns .mcp.json path at project root."""
        result = adapter.get_mcp_config_path(temp_dir)
        assert result == temp_dir / ".mcp.json"

    def test_get_commands_directory(self, adapter: ClaudeCodeAdapter, temp_dir: Path):
        """Returns .claude/commands directory (flat structure per Claude Code docs)."""
        result = adapter.get_commands_directory(temp_dir)
        assert result == temp_dir / ".claude" / "commands"

    def test_get_subagents_directory(self, adapter: ClaudeCodeAdapter, temp_dir: Path):
        """Returns .claude/agents directory (flat structure per Claude Code docs)."""
        result = adapter.get_subagents_directory(temp_dir)
        assert result == temp_dir / ".claude" / "agents"


class TestClaudeCodeAdapterPlanSkillInstallation:
    """Tests for ClaudeCodeAdapter.plan_skill_installation()."""

    def test_creates_installation_plan(
        self,
        adapter: ClaudeCodeAdapter,
        temp_dir: Path,
        sample_skill: SkillConfig,
        sample_manifest: PluginManifest,
    ):
        """Creates an installation plan."""
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        plan = adapter.plan_skill_installation(
            skill=sample_skill,
            plugin=sample_manifest,
            rendered_content="# Test Content",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        assert len(plan.directories_to_create) > 0
        assert len(plan.files_to_write) == 1

    def test_skill_file_path(
        self,
        adapter: ClaudeCodeAdapter,
        temp_dir: Path,
        sample_skill: SkillConfig,
        sample_manifest: PluginManifest,
    ):
        """Skill file is created at correct path."""
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        plan = adapter.plan_skill_installation(
            skill=sample_skill,
            plugin=sample_manifest,
            rendered_content="# Test",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        expected_path = temp_dir / ".claude" / "skills" / "test-plugin-test-skill" / "SKILL.md"
        assert plan.files_to_write[0].path == expected_path

    def test_includes_frontmatter(
        self,
        adapter: ClaudeCodeAdapter,
        temp_dir: Path,
        sample_skill: SkillConfig,
        sample_manifest: PluginManifest,
    ):
        """File content includes frontmatter."""
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        plan = adapter.plan_skill_installation(
            skill=sample_skill,
            plugin=sample_manifest,
            rendered_content="# Content",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        content = plan.files_to_write[0].content
        expected = """---
name: test-skill
description: A test skill
version: 1.0.0
---
# Content"""
        assert content == expected


class TestClaudeCodeAdapterPlanCommandInstallation:
    """Tests for ClaudeCodeAdapter.plan_command_installation()."""

    def test_creates_installation_plan(
        self,
        adapter: ClaudeCodeAdapter,
        temp_dir: Path,
        sample_command: CommandConfig,
        sample_manifest: PluginManifest,
    ):
        """Creates an installation plan for command as flat .md file."""
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        plan = adapter.plan_command_installation(
            command=sample_command,
            plugin=sample_manifest,
            rendered_content="# Command",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        assert len(plan.files_to_write) == 1
        # Per Claude Code docs: commands are flat .md files in .claude/commands/
        file_path = str(plan.files_to_write[0].path)
        assert ".claude/commands/" in file_path
        assert file_path.endswith(f"{sample_command.name}.md")


class TestClaudeCodeAdapterPlanSubagentInstallation:
    """Tests for ClaudeCodeAdapter.plan_subagent_installation()."""

    def test_creates_installation_plan(
        self,
        adapter: ClaudeCodeAdapter,
        temp_dir: Path,
        sample_subagent: SubAgentConfig,
        sample_manifest: PluginManifest,
    ):
        """Creates an installation plan for sub-agent as flat .md file."""
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        plan = adapter.plan_subagent_installation(
            subagent=sample_subagent,
            plugin=sample_manifest,
            rendered_content="# Agent",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        assert len(plan.files_to_write) == 1
        # Per Claude Code docs: agents are flat .md files in .claude/agents/
        file_path = str(plan.files_to_write[0].path)
        assert ".claude/agents/" in file_path
        assert file_path.endswith(f"{sample_subagent.name}.md")


class TestClaudeCodeAdapterGenerateMCPConfig:
    """Tests for ClaudeCodeAdapter.generate_mcp_config()."""

    def test_generates_npm_command_config(
        self,
        adapter: ClaudeCodeAdapter,
        temp_dir: Path,
        sample_manifest: PluginManifest,
    ):
        """Generates config for npm source shortcut."""
        mcp_server = MCPServerConfig(
            name="npm-server",
            type="command",
            source="npm:@example/mcp-server",
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
            "npm-server": {
                "command": "npx",
                "args": ["-y", "@example/mcp-server"],
            }
        }
        assert config == expected

    def test_generates_uvx_command_config(
        self,
        adapter: ClaudeCodeAdapter,
        temp_dir: Path,
        sample_manifest: PluginManifest,
    ):
        """Generates config for uvx source shortcut."""
        mcp_server = MCPServerConfig(
            name="uvx-server",
            type="command",
            source="uvx:mcp-package",
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
            "uvx-server": {
                "command": "uvx",
                "args": ["--from", "mcp-package"],
            }
        }
        assert config == expected

    def test_generates_direct_command_config(
        self,
        adapter: ClaudeCodeAdapter,
        temp_dir: Path,
        sample_manifest: PluginManifest,
    ):
        """Generates config for direct command/args."""
        mcp_server = MCPServerConfig(
            name="docker-server",
            type="command",
            command="docker",
            args=["run", "-i", "--rm", "ghcr.io/github/github-mcp-server"],
            env={"GITHUB_TOKEN": "${GITHUB_TOKEN}"},
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
            "docker-server": {
                "command": "docker",
                "args": ["run", "-i", "--rm", "ghcr.io/github/github-mcp-server"],
                "env": {"GITHUB_TOKEN": "${GITHUB_TOKEN}"},
            }
        }
        assert config == expected

    def test_generates_config_with_source_and_extra_args(
        self,
        adapter: ClaudeCodeAdapter,
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
        adapter: ClaudeCodeAdapter,
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
                "type": "http",
                "url": "https://mcp.atlassian.com/v1/mcp",
            }
        }
        assert config == expected

    def test_generates_http_server_config_insecure(
        self,
        adapter: ClaudeCodeAdapter,
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
                "type": "http",
                "url": "http://localhost:8080/mcp",
            }
        }
        assert config == expected


class TestClaudeCodeAdapterMergeMCPConfig:
    """Tests for ClaudeCodeAdapter.merge_mcp_config()."""

    def test_creates_mcp_servers_section(self, adapter: ClaudeCodeAdapter):
        """Creates mcpServers section if not present."""
        existing: dict[str, object] = {}
        new_entries = {"server1": {"command": "node"}}

        result = adapter.merge_mcp_config(existing, new_entries)

        assert "mcpServers" in result
        assert "server1" in result["mcpServers"]

    def test_merges_with_existing_servers(self, adapter: ClaudeCodeAdapter):
        """Merges with existing servers."""
        existing = {"mcpServers": {"existing": {"command": "python"}}}
        new_entries = {"new-server": {"command": "node"}}

        result = adapter.merge_mcp_config(existing, new_entries)

        assert "existing" in result["mcpServers"]
        assert "new-server" in result["mcpServers"]

    def test_preserves_other_settings(self, adapter: ClaudeCodeAdapter):
        """Preserves other settings in config."""
        existing = {"otherSetting": "value", "mcpServers": {}}
        new_entries = {"server": {"command": "node"}}

        result = adapter.merge_mcp_config(existing, new_entries)

        assert result["otherSetting"] == "value"


class TestClaudeCodeAdapterGenerateSkillFrontmatter:
    """Tests for ClaudeCodeAdapter.generate_skill_frontmatter()."""

    def test_generates_yaml_frontmatter(
        self, adapter: ClaudeCodeAdapter, sample_skill: SkillConfig, sample_manifest: PluginManifest
    ):
        """Generates YAML frontmatter per Claude Code docs."""
        frontmatter = adapter.generate_skill_frontmatter(sample_skill, sample_manifest)

        expected = """---
name: test-skill
description: A test skill
version: 1.0.0
---
"""
        assert frontmatter == expected

    def test_includes_metadata(self, adapter: ClaudeCodeAdapter, sample_manifest: PluginManifest):
        """Includes skill metadata as extra frontmatter fields."""
        skill = SkillConfig(
            name="test-skill",
            description="Test skill with metadata",
            context="./context.md",
            metadata={"author": "test", "category": "utility"},
        )

        frontmatter = adapter.generate_skill_frontmatter(skill, sample_manifest)

        expected = """---
name: test-skill
description: Test skill with metadata
version: 1.0.0
author: test
category: utility
---
"""
        assert frontmatter == expected


class TestClaudeCodeAdapterGenerateCommandFrontmatter:
    """Tests for ClaudeCodeAdapter.generate_command_frontmatter()."""

    def test_generates_basic_frontmatter(
        self,
        adapter: ClaudeCodeAdapter,
        sample_command: CommandConfig,
        sample_manifest: PluginManifest,
    ):
        """Generates basic command frontmatter."""
        frontmatter = adapter.generate_command_frontmatter(sample_command, sample_manifest)

        expected = """---
description: A test command
---
"""
        assert frontmatter == expected

    def test_includes_allowed_tools_as_list(
        self, adapter: ClaudeCodeAdapter, sample_manifest: PluginManifest
    ):
        """Includes allowed_tools as comma-separated list."""
        command = CommandConfig(
            name="test-command",
            description="Command with tools",
            context="./context.md",
            allowed_tools=["Bash(pytest:*)", "Bash(python:*)", "Read"],
        )

        frontmatter = adapter.generate_command_frontmatter(command, sample_manifest)

        expected = """---
description: Command with tools
allowed-tools: Bash(pytest:*), Bash(python:*), Read
---
"""
        assert frontmatter == expected

    def test_includes_allowed_tools_as_string(
        self, adapter: ClaudeCodeAdapter, sample_manifest: PluginManifest
    ):
        """Includes allowed_tools when specified as string."""
        command = CommandConfig(
            name="test-command",
            description="Command with tools",
            context="./context.md",
            allowed_tools="Bash(git:*)",
        )

        frontmatter = adapter.generate_command_frontmatter(command, sample_manifest)

        expected = """---
description: Command with tools
allowed-tools: Bash(git:*)
---
"""
        assert frontmatter == expected

    def test_allowed_tools_from_metadata(
        self, adapter: ClaudeCodeAdapter, sample_manifest: PluginManifest
    ):
        """Falls back to metadata for allowed_tools."""
        command = CommandConfig(
            name="test-command",
            description="Command with tools in metadata",
            context="./context.md",
            metadata={"allowed_tools": "Read, Write"},
        )

        frontmatter = adapter.generate_command_frontmatter(command, sample_manifest)

        expected = """---
description: Command with tools in metadata
allowed-tools: Read, Write
---
"""
        assert frontmatter == expected

    def test_no_allowed_tools_when_not_specified(
        self,
        adapter: ClaudeCodeAdapter,
        sample_command: CommandConfig,
        sample_manifest: PluginManifest,
    ):
        """Does not include allowed-tools when not specified."""
        frontmatter = adapter.generate_command_frontmatter(sample_command, sample_manifest)

        expected = """---
description: A test command
---
"""
        assert frontmatter == expected


class TestClaudeCodeAdapterGenerateSubagentFrontmatter:
    """Tests for ClaudeCodeAdapter.generate_subagent_frontmatter()."""

    def test_generates_basic_frontmatter(
        self,
        adapter: ClaudeCodeAdapter,
        sample_subagent: SubAgentConfig,
        sample_manifest: PluginManifest,
    ):
        """Generates basic agent frontmatter."""
        frontmatter = adapter.generate_subagent_frontmatter(sample_subagent, sample_manifest)

        expected = """---
name: test-agent
description: A test agent
model: inherit
color: blue
---
"""
        assert frontmatter == expected

    def test_includes_tools_from_allowed_tools(
        self, adapter: ClaudeCodeAdapter, sample_manifest: PluginManifest
    ):
        """Includes tools when allowed_tools is specified."""
        subagent = SubAgentConfig(
            name="test-agent",
            description="Agent with tools",
            context="./context.md",
            allowed_tools=["Read", "Grep", "Glob"],
        )

        frontmatter = adapter.generate_subagent_frontmatter(subagent, sample_manifest)

        expected = """---
name: test-agent
description: Agent with tools
model: inherit
color: blue
tools: ['Read', 'Grep', 'Glob']
---
"""
        assert frontmatter == expected

    def test_includes_tools_from_metadata(
        self, adapter: ClaudeCodeAdapter, sample_manifest: PluginManifest
    ):
        """Falls back to metadata for tools."""
        subagent = SubAgentConfig(
            name="test-agent",
            description="Agent with tools in metadata",
            context="./context.md",
            metadata={"tools": ["Read", "Write"]},
        )

        frontmatter = adapter.generate_subagent_frontmatter(subagent, sample_manifest)

        expected = """---
name: test-agent
description: Agent with tools in metadata
model: inherit
color: blue
tools: ['Read', 'Write']
---
"""
        assert frontmatter == expected


class TestClaudeCodeAdapterValidatePluginCompatibility:
    """Tests for ClaudeCodeAdapter.validate_plugin_compatibility()."""

    def test_returns_empty_list_for_compatible(
        self, adapter: ClaudeCodeAdapter, sample_manifest: PluginManifest
    ):
        """Returns empty list for compatible plugin."""
        warnings = adapter.validate_plugin_compatibility(sample_manifest)
        assert warnings == []


class TestClaudeCodeAdapterPreInstall:
    """Tests for ClaudeCodeAdapter.pre_install()."""

    def test_creates_directories(
        self, adapter: ClaudeCodeAdapter, temp_dir: Path, sample_manifest: PluginManifest
    ):
        """Creates .claude and skills directories."""
        adapter.pre_install(temp_dir, [sample_manifest])

        assert (temp_dir / ".claude").exists()
        assert (temp_dir / ".claude" / "skills").exists()

    def test_handles_existing_directories(
        self, adapter: ClaudeCodeAdapter, temp_dir: Path, sample_manifest: PluginManifest
    ):
        """Handles existing directories without error."""
        (temp_dir / ".claude" / "skills").mkdir(parents=True)

        # Should not raise
        adapter.pre_install(temp_dir, [sample_manifest])


class TestClaudeCodeAdapterFilesHandling:
    """Tests for file handling in installation plans."""

    def test_adds_files_to_plan(
        self,
        adapter: ClaudeCodeAdapter,
        temp_dir: Path,
        sample_manifest: PluginManifest,
    ):
        """Adds associated files to installation plan."""
        skill = SkillConfig(
            name="test-skill",
            description="Skill with files",
            context="./context.md",
            files=[FileTarget(src="config.json")],
        )
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()
        (source_dir / "config.json").write_text("{}")

        plan = adapter.plan_skill_installation(
            skill=skill,
            plugin=sample_manifest,
            rendered_content="# Test",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        assert len(plan.files_to_copy) > 0

    def test_adds_template_files_to_plan(
        self,
        adapter: ClaudeCodeAdapter,
        temp_dir: Path,
        sample_manifest: PluginManifest,
    ):
        """Adds template files to installation plan for rendering."""
        skill = SkillConfig(
            name="test-skill",
            description="Skill with template files",
            context="./context.md",
            template_files=[FileTarget(src="config.py.j2")],
        )
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()
        (source_dir / "config.py.j2").write_text("# {{ plugin.name }}")

        plan = adapter.plan_skill_installation(
            skill=skill,
            plugin=sample_manifest,
            rendered_content="# Test",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        assert len(plan.template_files_to_render) == 1
        src_path = list(plan.template_files_to_render.keys())[0]
        dest_path = list(plan.template_files_to_render.values())[0]
        assert "config.py.j2" in str(src_path)
        assert "config.py.j2" in str(dest_path)  # dest keeps same name

    def test_adds_template_files_with_custom_dest(
        self,
        adapter: ClaudeCodeAdapter,
        temp_dir: Path,
        sample_manifest: PluginManifest,
    ):
        """Adds template files with custom destination to plan."""
        skill = SkillConfig(
            name="test-skill",
            description="Skill with template files",
            context="./context.md",
            template_files=[FileTarget(src="./config.py.j2", dest="config.py")],
        )
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()
        (source_dir / "config.py.j2").write_text("# {{ plugin.name }}")

        plan = adapter.plan_skill_installation(
            skill=skill,
            plugin=sample_manifest,
            rendered_content="# Test",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        assert len(plan.template_files_to_render) == 1
        src_path = list(plan.template_files_to_render.keys())[0]
        dest_path = list(plan.template_files_to_render.values())[0]
        assert "config.py.j2" in str(src_path)
        assert str(dest_path).endswith("config.py")
        assert "j2" not in str(dest_path)

    def test_handles_both_files_and_template_files(
        self,
        adapter: ClaudeCodeAdapter,
        temp_dir: Path,
        sample_manifest: PluginManifest,
    ):
        """Handles both files and template_files in same skill."""
        skill = SkillConfig(
            name="test-skill",
            description="Skill with both file types",
            context="./context.md",
            files=[FileTarget(src="static.txt")],
            template_files=[FileTarget(src="config.py.j2")],
        )
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()
        (source_dir / "static.txt").write_text("static content")
        (source_dir / "config.py.j2").write_text("# {{ plugin.name }}")

        plan = adapter.plan_skill_installation(
            skill=skill,
            plugin=sample_manifest,
            rendered_content="# Test",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        # Should have both files_to_copy and template_files_to_render
        assert len(plan.files_to_copy) == 1
        assert len(plan.template_files_to_render) == 1
        assert any("static.txt" in str(src) for src in plan.files_to_copy)
        assert any("config.py.j2" in str(src) for src in plan.template_files_to_render)


class TestClaudeCodeAdapterGenerateRuleFrontmatter:
    """Tests for ClaudeCodeAdapter.generate_rule_frontmatter()."""

    def test_returns_empty_when_no_paths_or_metadata(
        self,
        adapter: ClaudeCodeAdapter,
        sample_manifest: PluginManifest,
    ):
        """Returns empty string when no paths or metadata specified."""
        rule = RuleConfig(
            name="test-rule",
            description="A test rule",
            context="./rules/test.md",
        )

        frontmatter = adapter.generate_rule_frontmatter(rule, sample_manifest)

        assert frontmatter == ""

    def test_generates_frontmatter_with_single_path(
        self,
        adapter: ClaudeCodeAdapter,
        sample_manifest: PluginManifest,
    ):
        """Generates YAML frontmatter with single path string."""
        rule = RuleConfig(
            name="test-rule",
            description="A test rule",
            context="./rules/test.md",
            paths="src/**/*.py",
        )

        frontmatter = adapter.generate_rule_frontmatter(rule, sample_manifest)

        expected = """\
---
paths: src/**/*.py
---
"""
        assert frontmatter == expected

    def test_generates_frontmatter_with_paths_list(
        self,
        adapter: ClaudeCodeAdapter,
        sample_manifest: PluginManifest,
    ):
        """Generates YAML frontmatter with list of paths."""
        rule = RuleConfig(
            name="test-rule",
            description="A test rule",
            context="./rules/test.md",
            paths=["tests/**/*.py", "**/*_test.py"],
        )

        frontmatter = adapter.generate_rule_frontmatter(rule, sample_manifest)

        expected = """\
---
paths:
  - tests/**/*.py
  - **/*_test.py
---
"""
        assert frontmatter == expected

    def test_passes_through_metadata(
        self,
        adapter: ClaudeCodeAdapter,
        sample_manifest: PluginManifest,
    ):
        """Passes through metadata fields."""
        rule = RuleConfig(
            name="test-rule",
            description="A test rule",
            context="./rules/test.md",
            metadata={"custom_field": "value", "enabled": True},
        )

        frontmatter = adapter.generate_rule_frontmatter(rule, sample_manifest)

        expected = """\
---
custom_field: value
enabled: true
---
"""
        assert frontmatter == expected

    def test_combines_paths_and_metadata(
        self,
        adapter: ClaudeCodeAdapter,
        sample_manifest: PluginManifest,
    ):
        """Combines paths and metadata in frontmatter."""
        rule = RuleConfig(
            name="test-rule",
            description="A test rule",
            context="./rules/test.md",
            paths=["src/**/*.ts"],
            metadata={"priority": "high"},
        )

        frontmatter = adapter.generate_rule_frontmatter(rule, sample_manifest)

        expected = """\
---
paths:
  - src/**/*.ts
priority: high
---
"""
        assert frontmatter == expected


class TestClaudeCodeAdapterPlatformSpecificFiles:
    """Tests for platform-specific file resolution in Claude Code adapter."""

    def test_resolves_single_platform_file_override(
        self,
        adapter: ClaudeCodeAdapter,
        temp_dir: Path,
        sample_manifest: PluginManifest,
    ):
        """Resolves single platform-specific file override (e.g., script.claude_code.sh)."""
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        # Create default and platform-specific files
        (source_dir / "scripts").mkdir()
        (source_dir / "scripts" / "setup.sh").write_text("#!/bin/bash\n# default")
        (source_dir / "scripts" / "setup.claude_code.sh").write_text("#!/bin/bash\n# claude code")

        skill = SkillConfig(
            name="test-skill",
            description="Test skill with platform file",
            context="./context.md",
            files=[FileTarget(src="scripts/setup.sh")],
        )

        plan = adapter.plan_skill_installation(
            skill=skill,
            plugin=sample_manifest,
            rendered_content="# Test",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        # Should resolve to claude_code-specific file
        assert len(plan.files_to_copy) == 1
        src_path = list(plan.files_to_copy.keys())[0]
        assert "setup.claude_code.sh" in str(src_path)

        # Destination should use original path
        dest_path = list(plan.files_to_copy.values())[0]
        assert "setup.sh" in str(dest_path)
        assert "setup.claude_code.sh" not in str(dest_path)

    def test_resolves_brace_expansion_file_override(
        self,
        adapter: ClaudeCodeAdapter,
        temp_dir: Path,
        sample_manifest: PluginManifest,
    ):
        """Resolves brace expansion file override (e.g., config.{claude_code,cursor}.json)."""
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        # Create default and multi-platform file
        (source_dir / "configs").mkdir()
        (source_dir / "configs" / "settings.json").write_text('{"default": true}')
        (source_dir / "configs" / "settings.{claude_code,cursor}.json").write_text(
            '{"claude_or_cursor": true}'
        )

        skill = SkillConfig(
            name="test-skill",
            description="Test skill with multi-platform file",
            context="./context.md",
            files=[FileTarget(src="configs/settings.json")],
        )

        plan = adapter.plan_skill_installation(
            skill=skill,
            plugin=sample_manifest,
            rendered_content="# Test",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        # Should resolve to multi-platform file
        assert len(plan.files_to_copy) == 1
        src_path = list(plan.files_to_copy.keys())[0]
        assert "{claude_code,cursor}" in str(src_path)

    def test_falls_back_to_default_file(
        self,
        adapter: ClaudeCodeAdapter,
        temp_dir: Path,
        sample_manifest: PluginManifest,
    ):
        """Falls back to default file when no platform override exists."""
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        # Create only default file (no platform-specific override)
        (source_dir / "data").mkdir()
        (source_dir / "data" / "config.yaml").write_text("default: true")

        skill = SkillConfig(
            name="test-skill",
            description="Test skill with default file",
            context="./context.md",
            files=[FileTarget(src="data/config.yaml")],
        )

        plan = adapter.plan_skill_installation(
            skill=skill,
            plugin=sample_manifest,
            rendered_content="# Test",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        # Should use default file
        assert len(plan.files_to_copy) == 1
        src_path = list(plan.files_to_copy.keys())[0]
        assert "config.yaml" in str(src_path)
        assert "claude_code" not in str(src_path)

    def test_prefers_exact_platform_over_brace_expansion(
        self,
        adapter: ClaudeCodeAdapter,
        temp_dir: Path,
        sample_manifest: PluginManifest,
    ):
        """Prefers exact platform match over brace expansion."""
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        # Create all three types of files
        (source_dir / "scripts").mkdir()
        (source_dir / "scripts" / "run.sh").write_text("# default")
        (source_dir / "scripts" / "run.claude_code.sh").write_text("# exact match")
        (source_dir / "scripts" / "run.{claude_code,cursor}.sh").write_text("# multi")

        skill = SkillConfig(
            name="test-skill",
            description="Test skill",
            context="./context.md",
            files=[FileTarget(src="scripts/run.sh")],
        )

        plan = adapter.plan_skill_installation(
            skill=skill,
            plugin=sample_manifest,
            rendered_content="# Test",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        # Should resolve to exact platform match
        assert len(plan.files_to_copy) == 1
        src_path = list(plan.files_to_copy.keys())[0]
        assert "run.claude_code.sh" in str(src_path)
        assert "{claude_code,cursor}" not in str(src_path)


class TestClaudeCodeAdapterGetClaudeSettingsPath:
    """Tests for ClaudeCodeAdapter.get_claude_settings_path()."""

    def test_returns_settings_path(self, adapter: ClaudeCodeAdapter, temp_dir: Path):
        """Returns .claude/settings.json path."""
        result = adapter.get_claude_settings_path(temp_dir)
        assert result == temp_dir / ".claude" / "settings.json"


class TestClaudeCodeAdapterGenerateClaudeSettingsConfig:
    """Tests for ClaudeCodeAdapter.generate_claude_settings_config()."""

    def test_generates_config_with_allow(
        self, adapter: ClaudeCodeAdapter, temp_dir: Path, sample_manifest: PluginManifest
    ):
        """Generates config with allow permissions."""
        claude_settings = ClaudeSettingsConfig(
            allow=["mcp__serena", "mcp__github"],
        )

        config = adapter.generate_claude_settings_config(
            claude_settings, sample_manifest, temp_dir
        )

        expected = {
            "permissions": {
                "allow": ["mcp__serena", "mcp__github"],
            }
        }
        assert config == expected

    def test_generates_config_with_deny(
        self, adapter: ClaudeCodeAdapter, temp_dir: Path, sample_manifest: PluginManifest
    ):
        """Generates config with deny permissions."""
        claude_settings = ClaudeSettingsConfig(
            deny=["Bash(curl:*)", "WebFetch"],
        )

        config = adapter.generate_claude_settings_config(
            claude_settings, sample_manifest, temp_dir
        )

        expected = {
            "permissions": {
                "deny": ["Bash(curl:*)", "WebFetch"],
            }
        }
        assert config == expected

    def test_generates_config_with_both(
        self, adapter: ClaudeCodeAdapter, temp_dir: Path, sample_manifest: PluginManifest
    ):
        """Generates config with both allow and deny permissions."""
        claude_settings = ClaudeSettingsConfig(
            allow=["mcp__serena"],
            deny=["Bash(rm:*)"],
        )

        config = adapter.generate_claude_settings_config(
            claude_settings, sample_manifest, temp_dir
        )

        expected = {
            "permissions": {
                "allow": ["mcp__serena"],
                "deny": ["Bash(rm:*)"],
            }
        }
        assert config == expected

    def test_generates_empty_config_when_no_permissions(
        self, adapter: ClaudeCodeAdapter, temp_dir: Path, sample_manifest: PluginManifest
    ):
        """Returns empty dict when no permissions specified."""
        claude_settings = ClaudeSettingsConfig()

        config = adapter.generate_claude_settings_config(
            claude_settings, sample_manifest, temp_dir
        )

        assert config == {}


class TestClaudeCodeAdapterMergeClaudeSettingsConfig:
    """Tests for ClaudeCodeAdapter.merge_claude_settings_config()."""

    def test_creates_permissions_section(self, adapter: ClaudeCodeAdapter):
        """Creates permissions section if not present."""
        existing: dict[str, object] = {}
        new_entries = {"permissions": {"allow": ["mcp__serena"]}}

        result = adapter.merge_claude_settings_config(existing, new_entries)

        assert "permissions" in result
        assert "allow" in result["permissions"]
        assert "mcp__serena" in result["permissions"]["allow"]

    def test_merges_with_existing_permissions(self, adapter: ClaudeCodeAdapter):
        """Merges with existing permissions."""
        existing = {"permissions": {"allow": ["mcp__existing"]}}
        new_entries = {"permissions": {"allow": ["mcp__new"]}}

        result = adapter.merge_claude_settings_config(existing, new_entries)

        assert "mcp__existing" in result["permissions"]["allow"]
        assert "mcp__new" in result["permissions"]["allow"]

    def test_deduplicates_permissions(self, adapter: ClaudeCodeAdapter):
        """De-duplicates permission entries."""
        existing = {"permissions": {"allow": ["mcp__serena", "mcp__github"]}}
        new_entries = {"permissions": {"allow": ["mcp__serena", "mcp__new"]}}

        result = adapter.merge_claude_settings_config(existing, new_entries)

        # Should have 3 unique entries, not 4
        assert len(result["permissions"]["allow"]) == 3
        assert result["permissions"]["allow"].count("mcp__serena") == 1

    def test_preserves_other_settings(self, adapter: ClaudeCodeAdapter):
        """Preserves other settings in config."""
        existing = {
            "enableAllProjectMcpServers": True,
            "otherSetting": "value",
            "permissions": {"allow": ["existing"]},
        }
        new_entries = {"permissions": {"allow": ["new"]}}

        result = adapter.merge_claude_settings_config(existing, new_entries)

        assert result["enableAllProjectMcpServers"] is True
        assert result["otherSetting"] == "value"

    def test_returns_existing_when_no_new_permissions(self, adapter: ClaudeCodeAdapter):
        """Returns existing config unchanged when no new permissions."""
        existing = {"enableAllProjectMcpServers": True}
        new_entries: dict[str, object] = {}

        result = adapter.merge_claude_settings_config(existing, new_entries)

        assert result == existing

    def test_merges_deny_permissions(self, adapter: ClaudeCodeAdapter):
        """Merges deny permissions correctly."""
        existing = {"permissions": {"deny": ["Bash(rm:*)"]}}
        new_entries = {"permissions": {"deny": ["Bash(curl:*)"]}}

        result = adapter.merge_claude_settings_config(existing, new_entries)

        assert "Bash(rm:*)" in result["permissions"]["deny"]
        assert "Bash(curl:*)" in result["permissions"]["deny"]

    def test_preserves_order(self, adapter: ClaudeCodeAdapter):
        """Preserves order of existing entries while de-duplicating."""
        existing = {"permissions": {"allow": ["a", "b", "c"]}}
        new_entries = {"permissions": {"allow": ["b", "d"]}}

        result = adapter.merge_claude_settings_config(existing, new_entries)

        # Order should be: existing (a, b, c), then new non-duplicates (d)
        assert result["permissions"]["allow"] == ["a", "b", "c", "d"]

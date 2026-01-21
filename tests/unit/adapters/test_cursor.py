"""Tests for dex.adapters.cursor module."""

from pathlib import Path

import pytest

from dex.adapters.cursor import CursorAdapter
from dex.config.schemas import (
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
    """Get a CursorAdapter instance."""
    return CursorAdapter()


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
        description="A test skill for code review",
        context="./context/skill.md",
    )


@pytest.fixture
def sample_rule():
    """Sample rule configuration."""
    return RuleConfig(
        name="test-rule",
        description="A test rule for code standards",
        context="./context/rule.md",
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


class TestCursorAdapterMetadata:
    """Tests for CursorAdapter.metadata property."""

    def test_returns_metadata(self, adapter: CursorAdapter):
        """Returns AdapterMetadata with correct values."""
        meta = adapter.metadata
        assert meta.name == "cursor"
        assert meta.display_name == "Cursor"
        assert meta.mcp_config_file == ".cursor/mcp.json"


class TestCursorAdapterDirectories:
    """Tests for CursorAdapter directory methods."""

    def test_get_base_directory(self, adapter: CursorAdapter, temp_dir: Path):
        """Returns .cursor directory."""
        result = adapter.get_base_directory(temp_dir)
        assert result == temp_dir / ".cursor"

    def test_get_skills_directory(self, adapter: CursorAdapter, temp_dir: Path):
        """Returns .cursor/rules directory (skills not supported, returns fallback)."""
        result = adapter.get_skills_directory(temp_dir)
        assert result == temp_dir / ".cursor" / "rules"

    def test_get_rules_directory(self, adapter: CursorAdapter, temp_dir: Path):
        """Returns .cursor/rules directory."""
        result = adapter.get_rules_directory(temp_dir)
        assert result == temp_dir / ".cursor" / "rules"

    def test_get_mcp_config_path(self, adapter: CursorAdapter, temp_dir: Path):
        """Returns .cursor/mcp.json path."""
        result = adapter.get_mcp_config_path(temp_dir)
        assert result == temp_dir / ".cursor" / "mcp.json"


class TestCursorAdapterSkillsNotSupported:
    """Tests that Cursor does not support skills."""

    def test_plan_skill_installation_returns_empty_plan(
        self,
        adapter: CursorAdapter,
        temp_dir: Path,
        sample_skill: SkillConfig,
        sample_manifest: PluginManifest,
    ):
        """plan_skill_installation returns empty plan (skills not supported)."""
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        plan = adapter.plan_skill_installation(
            skill=sample_skill,
            plugin=sample_manifest,
            rendered_content="# Test Content",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        assert len(plan.directories_to_create) == 0
        assert len(plan.files_to_write) == 0

    def test_generate_skill_frontmatter_returns_empty(
        self, adapter: CursorAdapter, sample_skill: SkillConfig, sample_manifest: PluginManifest
    ):
        """generate_skill_frontmatter returns empty string (skills not supported)."""
        frontmatter = adapter.generate_skill_frontmatter(sample_skill, sample_manifest)
        assert frontmatter == ""


class TestCursorAdapterPlanRuleInstallation:
    """Tests for CursorAdapter.plan_rule_installation()."""

    def test_creates_installation_plan(
        self,
        adapter: CursorAdapter,
        temp_dir: Path,
        sample_rule: RuleConfig,
        sample_manifest: PluginManifest,
    ):
        """Creates an installation plan with MDC file."""
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        plan = adapter.plan_rule_installation(
            rule=sample_rule,
            plugin=sample_manifest,
            rendered_content="# Test Content",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        assert len(plan.directories_to_create) > 0
        assert len(plan.files_to_write) == 1

    def test_rule_file_path_is_mdc(
        self,
        adapter: CursorAdapter,
        temp_dir: Path,
        sample_rule: RuleConfig,
        sample_manifest: PluginManifest,
    ):
        """Rule file is created as .mdc in .cursor/rules/."""
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        plan = adapter.plan_rule_installation(
            rule=sample_rule,
            plugin=sample_manifest,
            rendered_content="# Test",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        # Cursor uses flat .mdc files: .cursor/rules/{plugin}-{rule}.mdc
        expected_path = temp_dir / ".cursor" / "rules" / "test-plugin-test-rule.mdc"
        assert plan.files_to_write[0].path == expected_path

    def test_includes_frontmatter(
        self,
        adapter: CursorAdapter,
        temp_dir: Path,
        sample_rule: RuleConfig,
        sample_manifest: PluginManifest,
    ):
        """File content includes Cursor-specific frontmatter."""
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        plan = adapter.plan_rule_installation(
            rule=sample_rule,
            plugin=sample_manifest,
            rendered_content="# Content",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        content = plan.files_to_write[0].content
        expected = """---
description: A test rule for code standards
globs:
alwaysApply: false
---
# Content"""
        assert content == expected


class TestCursorAdapterGenerateRuleFrontmatter:
    """Tests for CursorAdapter.generate_rule_frontmatter()."""

    def test_generates_cursor_frontmatter(
        self, adapter: CursorAdapter, sample_rule: RuleConfig, sample_manifest: PluginManifest
    ):
        """Generates Cursor MDC frontmatter with description, globs, alwaysApply."""
        frontmatter = adapter.generate_rule_frontmatter(sample_rule, sample_manifest)

        expected = """---
description: A test rule for code standards
globs:
alwaysApply: false
---
"""
        assert frontmatter == expected

    def test_includes_globs_from_config(
        self, adapter: CursorAdapter, sample_manifest: PluginManifest
    ):
        """Includes globs from rule config."""
        rule = RuleConfig(
            name="ts-rule",
            description="TypeScript rule",
            context="./context.md",
            glob="**/*.ts",
        )

        frontmatter = adapter.generate_rule_frontmatter(rule, sample_manifest)

        expected = """---
description: TypeScript rule
globs: **/*.ts
alwaysApply: false
---
"""
        assert frontmatter == expected

    def test_includes_always_apply_from_config(
        self, adapter: CursorAdapter, sample_manifest: PluginManifest
    ):
        """Includes alwaysApply from rule config."""
        rule = RuleConfig(
            name="global-rule",
            description="Always apply this rule",
            context="./context.md",
            always=True,
        )

        frontmatter = adapter.generate_rule_frontmatter(rule, sample_manifest)

        expected = """---
description: Always apply this rule
globs:
alwaysApply: true
---
"""
        assert frontmatter == expected

    def test_includes_globs_and_always_apply(
        self, adapter: CursorAdapter, sample_manifest: PluginManifest
    ):
        """Includes both globs and alwaysApply."""
        rule = RuleConfig(
            name="python-rule",
            description="Python code standards",
            context="./context.md",
            glob="**/*.py",
            always=True,
        )

        frontmatter = adapter.generate_rule_frontmatter(rule, sample_manifest)

        expected = """---
description: Python code standards
globs: **/*.py
alwaysApply: true
---
"""
        assert frontmatter == expected


class TestCursorAdapterValidatePluginCompatibility:
    """Tests for CursorAdapter.validate_plugin_compatibility()."""

    def test_no_warnings_for_mcp_servers(self, adapter: CursorAdapter):
        """No warnings for MCP servers (supported)."""
        from dex.config.schemas import MCPServerConfig

        manifest = PluginManifest(
            name="test-plugin",
            version="1.0.0",
            description="Plugin with MCP",
            mcp_servers=[MCPServerConfig(name="test-server", type="command", source="npm:test")],
        )

        warnings = adapter.validate_plugin_compatibility(manifest)

        # MCP is supported - no warnings for MCP servers
        assert not any("MCP" in w for w in warnings)

    def test_no_warnings_for_rules_only(
        self, adapter: CursorAdapter, sample_manifest: PluginManifest
    ):
        """No warnings for plugin with only rules."""
        warnings = adapter.validate_plugin_compatibility(sample_manifest)
        assert warnings == []


class TestCursorAdapterPreInstall:
    """Tests for CursorAdapter.pre_install()."""

    def test_creates_directories(
        self, adapter: CursorAdapter, temp_dir: Path, sample_manifest: PluginManifest
    ):
        """Creates .cursor, rules, and commands directories."""
        adapter.pre_install(temp_dir, [sample_manifest])

        assert (temp_dir / ".cursor").exists()
        assert (temp_dir / ".cursor" / "rules").exists()
        assert (temp_dir / ".cursor" / "commands").exists()

    def test_handles_existing_directories(
        self, adapter: CursorAdapter, temp_dir: Path, sample_manifest: PluginManifest
    ):
        """Handles existing directories without error."""
        (temp_dir / ".cursor" / "rules").mkdir(parents=True)

        # Should not raise
        adapter.pre_install(temp_dir, [sample_manifest])


class TestCursorAdapterMCPConfig:
    """Tests for CursorAdapter MCP configuration."""

    def test_generate_mcp_config_npm(
        self, adapter: CursorAdapter, temp_dir: Path, sample_manifest: PluginManifest
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

    def test_generate_mcp_config_with_env(
        self, adapter: CursorAdapter, temp_dir: Path, sample_manifest: PluginManifest
    ):
        """Generates MCP config with environment variables."""
        from dex.config.schemas import MCPServerConfig

        mcp_server = MCPServerConfig(
            name="test-server",
            type="command",
            source="npm:@example/mcp-server",
            env={"API_KEY": "${API_KEY}"},
        )
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        config = adapter.generate_mcp_config(mcp_server, sample_manifest, temp_dir, source_dir)

        assert config["test-server"]["env"] == {"API_KEY": "${API_KEY}"}

    def test_merge_mcp_config(self, adapter: CursorAdapter):
        """Merges MCP config into mcpServers object."""
        existing: dict[str, object] = {}
        new_entries = {"test-server": {"command": "npx", "args": ["-y", "test"]}}

        result = adapter.merge_mcp_config(existing, new_entries)

        expected = {"mcpServers": {"test-server": {"command": "npx", "args": ["-y", "test"]}}}
        assert result == expected

    def test_merge_mcp_config_preserves_existing(self, adapter: CursorAdapter):
        """Merge preserves existing MCP servers."""
        existing = {"mcpServers": {"existing-server": {"command": "node", "args": []}}}
        new_entries = {"new-server": {"command": "npx", "args": ["-y", "test"]}}

        result = adapter.merge_mcp_config(existing, new_entries)

        assert "existing-server" in result["mcpServers"]
        assert "new-server" in result["mcpServers"]

    def test_generates_config_with_source_and_extra_args(
        self,
        adapter: CursorAdapter,
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
        adapter: CursorAdapter,
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
        adapter: CursorAdapter,
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


class TestCursorAdapterGetCommandsDirectory:
    """Tests for CursorAdapter.get_commands_directory()."""

    def test_returns_commands_directory(self, adapter: CursorAdapter, temp_dir: Path):
        """Returns .cursor/commands directory."""
        result = adapter.get_commands_directory(temp_dir)
        assert result == temp_dir / ".cursor" / "commands"


class TestCursorAdapterGenerateCommandFrontmatter:
    """Tests for CursorAdapter.generate_command_frontmatter()."""

    def test_returns_empty_string(
        self,
        adapter: CursorAdapter,
        sample_command: CommandConfig,
        sample_manifest: PluginManifest,
    ):
        """Cursor commands do not require frontmatter."""
        frontmatter = adapter.generate_command_frontmatter(sample_command, sample_manifest)
        assert frontmatter == ""


class TestCursorAdapterPlanCommandInstallation:
    """Tests for CursorAdapter.plan_command_installation()."""

    def test_creates_installation_plan(
        self,
        adapter: CursorAdapter,
        temp_dir: Path,
        sample_command: CommandConfig,
        sample_manifest: PluginManifest,
    ):
        """Creates an installation plan with Markdown file."""
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        plan = adapter.plan_command_installation(
            command=sample_command,
            plugin=sample_manifest,
            rendered_content="# Test Content",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        assert len(plan.directories_to_create) > 0
        assert len(plan.files_to_write) == 1

    def test_command_file_path_is_md(
        self,
        adapter: CursorAdapter,
        temp_dir: Path,
        sample_command: CommandConfig,
        sample_manifest: PluginManifest,
    ):
        """Command file is created as .md in .cursor/commands/."""
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        plan = adapter.plan_command_installation(
            command=sample_command,
            plugin=sample_manifest,
            rendered_content="# Test",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        # Cursor uses flat .md files: .cursor/commands/{plugin}-{command}.md
        expected_path = temp_dir / ".cursor" / "commands" / "test-plugin-test-command.md"
        assert plan.files_to_write[0].path == expected_path

    def test_content_is_plain_markdown(
        self,
        adapter: CursorAdapter,
        temp_dir: Path,
        sample_command: CommandConfig,
        sample_manifest: PluginManifest,
    ):
        """File content is plain Markdown without frontmatter."""
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        plan = adapter.plan_command_installation(
            command=sample_command,
            plugin=sample_manifest,
            rendered_content="# Content",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        content = plan.files_to_write[0].content
        expected = "# Content"
        assert content == expected

    def test_copies_associated_files(
        self,
        adapter: CursorAdapter,
        temp_dir: Path,
        sample_manifest: PluginManifest,
    ):
        """Copies associated files to commands directory."""
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()
        (source_dir / "helper.py").write_text("# helper")

        command = CommandConfig(
            name="test-command",
            description="A test command",
            context="./context/command.md",
            files=[FileTarget(src="helper.py")],
        )

        plan = adapter.plan_command_installation(
            command=command,
            plugin=sample_manifest,
            rendered_content="# Content",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        expected_src = source_dir / "helper.py"
        expected_dest = temp_dir / ".cursor" / "commands" / "helper.py"
        assert expected_src in plan.files_to_copy
        assert plan.files_to_copy[expected_src] == expected_dest

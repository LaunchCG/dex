"""Tests for dex.adapters.codex module."""

from pathlib import Path

import pytest

from dex.adapters.codex import CodexAdapter
from dex.config.schemas import (
    CommandConfig,
    PluginManifest,
    RuleConfig,
    SkillConfig,
    SubAgentConfig,
)


@pytest.fixture
def adapter():
    """Get a CodexAdapter instance."""
    return CodexAdapter()


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


class TestCodexAdapterMetadata:
    """Tests for CodexAdapter.metadata property."""

    def test_returns_metadata(self, adapter: CodexAdapter):
        """Returns AdapterMetadata with correct values."""
        meta = adapter.metadata
        assert meta.name == "codex"
        assert meta.display_name == "OpenAI Codex"
        assert meta.mcp_config_file == "~/.codex/config.toml"  # Global TOML config


class TestCodexAdapterDirectories:
    """Tests for CodexAdapter directory methods."""

    def test_get_base_directory(self, adapter: CodexAdapter, temp_dir: Path):
        """Returns .codex directory."""
        result = adapter.get_base_directory(temp_dir)
        assert result == temp_dir / ".codex"

    def test_get_skills_directory(self, adapter: CodexAdapter, temp_dir: Path):
        """Returns .codex/skills directory."""
        result = adapter.get_skills_directory(temp_dir)
        assert result == temp_dir / ".codex" / "skills"

    def test_get_mcp_config_path(self, adapter: CodexAdapter, temp_dir: Path):
        """Returns global ~/.codex/config.toml path."""
        from dex.utils.platform import get_home_directory

        result = adapter.get_mcp_config_path(temp_dir)
        expected = Path(get_home_directory()) / ".codex" / "config.toml"
        assert result == expected


class TestCodexAdapterPlanSkillInstallation:
    """Tests for CodexAdapter.plan_skill_installation()."""

    def test_creates_installation_plan(
        self,
        adapter: CodexAdapter,
        temp_dir: Path,
        sample_skill: SkillConfig,
        sample_manifest: PluginManifest,
    ):
        """Creates an installation plan with SKILL.md file."""
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
        adapter: CodexAdapter,
        temp_dir: Path,
        sample_skill: SkillConfig,
        sample_manifest: PluginManifest,
    ):
        """Skill file is created as SKILL.md in .codex/skills/{skill}/."""
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        plan = adapter.plan_skill_installation(
            skill=sample_skill,
            plugin=sample_manifest,
            rendered_content="# Test",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        # Codex uses: .codex/skills/{skill-name}/SKILL.md
        expected_path = temp_dir / ".codex" / "skills" / "test-skill" / "SKILL.md"
        assert plan.files_to_write[0].path == expected_path

    def test_includes_frontmatter(
        self,
        adapter: CodexAdapter,
        temp_dir: Path,
        sample_skill: SkillConfig,
        sample_manifest: PluginManifest,
    ):
        """File content includes Codex-specific frontmatter."""
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
description: A test skill for code review
---
# Content"""
        assert content == expected


class TestCodexAdapterGenerateSkillFrontmatter:
    """Tests for CodexAdapter.generate_skill_frontmatter()."""

    def test_generates_codex_frontmatter(
        self, adapter: CodexAdapter, sample_skill: SkillConfig, sample_manifest: PluginManifest
    ):
        """Generates Codex frontmatter with name and description."""
        frontmatter = adapter.generate_skill_frontmatter(sample_skill, sample_manifest)

        expected = """---
name: test-skill
description: A test skill for code review
---
"""
        assert frontmatter == expected

    def test_includes_short_description_from_metadata(
        self, adapter: CodexAdapter, sample_manifest: PluginManifest
    ):
        """Includes short-description metadata if provided."""
        skill = SkillConfig(
            name="advanced-skill",
            description="Advanced skill with metadata",
            context="./context.md",
            metadata={"short-description": "Short desc for UI"},
        )

        frontmatter = adapter.generate_skill_frontmatter(skill, sample_manifest)

        expected = """---
name: advanced-skill
description: Advanced skill with metadata
metadata:
  short-description: Short desc for UI
---
"""
        assert frontmatter == expected


class TestCodexAdapterValidatePluginCompatibility:
    """Tests for CodexAdapter.validate_plugin_compatibility()."""

    def test_no_warnings_for_mcp_servers(self, adapter: CodexAdapter):
        """No warnings for MCP servers (Codex supports MCP via global TOML config)."""
        from dex.config.schemas import MCPServerConfig

        manifest = PluginManifest(
            name="test-plugin",
            version="1.0.0",
            description="Plugin with MCP",
            mcp_servers=[MCPServerConfig(name="test-server", type="remote", source="npm:test")],
        )

        warnings = adapter.validate_plugin_compatibility(manifest)

        # Codex supports MCP - no warnings for MCP servers
        assert not any("MCP" in w for w in warnings)

    def test_warns_about_subagents(self, adapter: CodexAdapter):
        """Warns when plugin has subagents."""
        manifest = PluginManifest(
            name="test-plugin",
            version="1.0.0",
            description="Plugin with agents",
            sub_agents=[SubAgentConfig(name="agent", description="Agent", context="./ctx.md")],
        )

        warnings = adapter.validate_plugin_compatibility(manifest)

        assert len(warnings) >= 1
        assert any("subagent" in w.lower() for w in warnings)

    def test_warns_about_commands(self, adapter: CodexAdapter):
        """Warns when plugin has commands (should use skills/AGENTS.md)."""
        manifest = PluginManifest(
            name="test-plugin",
            version="1.0.0",
            description="Plugin with commands",
            commands=[CommandConfig(name="cmd", description="Command", context="./ctx.md")],
        )

        warnings = adapter.validate_plugin_compatibility(manifest)

        assert len(warnings) >= 1
        assert any("command" in w.lower() for w in warnings)

    def test_no_warnings_for_skills_only(
        self, adapter: CodexAdapter, sample_manifest: PluginManifest
    ):
        """No warnings for plugin with only skills."""
        warnings = adapter.validate_plugin_compatibility(sample_manifest)
        assert warnings == []


class TestCodexAdapterPreInstall:
    """Tests for CodexAdapter.pre_install()."""

    def test_creates_directories(
        self, adapter: CodexAdapter, temp_dir: Path, sample_manifest: PluginManifest
    ):
        """Creates .codex and skills directories."""
        adapter.pre_install(temp_dir, [sample_manifest])

        assert (temp_dir / ".codex").exists()
        assert (temp_dir / ".codex" / "skills").exists()

    def test_handles_existing_directories(
        self, adapter: CodexAdapter, temp_dir: Path, sample_manifest: PluginManifest
    ):
        """Handles existing directories without error."""
        (temp_dir / ".codex" / "skills").mkdir(parents=True)

        # Should not raise
        adapter.pre_install(temp_dir, [sample_manifest])


class TestCodexAdapterFilesHandling:
    """Tests for file handling in installation plans."""

    def test_adds_files_to_plan(
        self,
        adapter: CodexAdapter,
        temp_dir: Path,
        sample_manifest: PluginManifest,
    ):
        """Adds associated files (scripts, references) to installation plan."""
        skill = SkillConfig(
            name="test-skill",
            description="Skill with files",
            context="./context.md",
            files=["./scripts/helper.sh", "./references/guide.md"],
        )
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()
        (source_dir / "scripts").mkdir()
        (source_dir / "scripts" / "helper.sh").write_text("#!/bin/bash")
        (source_dir / "references").mkdir()
        (source_dir / "references" / "guide.md").write_text("# Guide")

        plan = adapter.plan_skill_installation(
            skill=skill,
            plugin=sample_manifest,
            rendered_content="# Test",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        assert len(plan.files_to_copy) >= 2
        assert any("helper.sh" in str(src) for src in plan.files_to_copy)
        assert any("guide.md" in str(src) for src in plan.files_to_copy)


class TestCodexAdapterMCPConfig:
    """Tests for CodexAdapter MCP configuration (TOML format)."""

    def test_generate_mcp_config_remote_npm(
        self, adapter: CodexAdapter, temp_dir: Path, sample_manifest: PluginManifest
    ):
        """Generates MCP config for remote npm package."""
        from dex.config.schemas import MCPServerConfig

        mcp_server = MCPServerConfig(
            name="test-server",
            type="remote",
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

    def test_generate_mcp_config_bundled_js(
        self, adapter: CodexAdapter, temp_dir: Path, sample_manifest: PluginManifest
    ):
        """Generates MCP config for bundled JavaScript server."""
        from dex.config.schemas import MCPServerConfig

        mcp_server = MCPServerConfig(
            name="bundled-server",
            type="bundled",
            path="./servers/server.js",
        )
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        config = adapter.generate_mcp_config(mcp_server, sample_manifest, temp_dir, source_dir)

        assert config["bundled-server"]["command"] == "node"
        assert str(source_dir / "servers" / "server.js") in config["bundled-server"]["args"][0]

    def test_generate_mcp_config_with_env(
        self, adapter: CodexAdapter, temp_dir: Path, sample_manifest: PluginManifest
    ):
        """Generates MCP config with environment variables."""
        from dex.config.schemas import MCPServerConfig

        mcp_server = MCPServerConfig(
            name="test-server",
            type="remote",
            source="npm:@example/mcp-server",
            config={"env": {"API_KEY": "${API_KEY}"}},
        )
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        config = adapter.generate_mcp_config(mcp_server, sample_manifest, temp_dir, source_dir)

        assert config["test-server"]["env"] == {"API_KEY": "${API_KEY}"}

    def test_merge_mcp_config(self, adapter: CodexAdapter):
        """Merges MCP config into mcp_servers object (TOML format)."""
        existing: dict[str, object] = {}
        new_entries = {"test-server": {"command": "npx", "args": ["-y", "test"]}}

        result = adapter.merge_mcp_config(existing, new_entries)

        expected = {"mcp_servers": {"test-server": {"command": "npx", "args": ["-y", "test"]}}}
        assert result == expected

    def test_merge_mcp_config_preserves_existing(self, adapter: CodexAdapter):
        """Merge preserves existing MCP servers and other config."""
        existing = {
            "mcp_servers": {"existing-server": {"command": "node", "args": []}},
            "other_setting": "value",
        }
        new_entries = {"new-server": {"command": "npx", "args": ["-y", "test"]}}

        result = adapter.merge_mcp_config(existing, new_entries)

        assert "existing-server" in result["mcp_servers"]
        assert "new-server" in result["mcp_servers"]
        assert result["other_setting"] == "value"


class TestCodexAdapterGenerateRuleFrontmatter:
    """Tests for CodexAdapter.generate_rule_frontmatter()."""

    def test_returns_empty_string(
        self,
        adapter: CodexAdapter,
        sample_manifest: PluginManifest,
    ):
        """Codex uses plain markdown rules without YAML frontmatter."""
        rule = RuleConfig(
            name="test-rule",
            description="A test rule",
            context="./rules/test.md",
        )

        frontmatter = adapter.generate_rule_frontmatter(rule, sample_manifest)

        assert frontmatter == ""

    def test_returns_empty_even_with_metadata(
        self,
        adapter: CodexAdapter,
        sample_manifest: PluginManifest,
    ):
        """Returns empty even when rule has metadata (Codex ignores metadata)."""
        rule = RuleConfig(
            name="test-rule",
            description="A test rule with metadata",
            context="./rules/test.md",
            glob="**/*.py",
            metadata={"category": "testing"},
        )

        frontmatter = adapter.generate_rule_frontmatter(rule, sample_manifest)

        assert frontmatter == ""

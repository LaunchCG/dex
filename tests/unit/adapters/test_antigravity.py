"""Tests for dex.adapters.antigravity module."""

from pathlib import Path

import pytest

from dex.adapters.antigravity import AntigravityAdapter
from dex.config.schemas import (
    CommandConfig,
    FileTarget,
    PluginManifest,
    RuleConfig,
    SkillConfig,
    SubAgentConfig,
)


@pytest.fixture
def adapter():
    """Get an AntigravityAdapter instance."""
    return AntigravityAdapter()


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


class TestAntigravityAdapterMetadata:
    """Tests for AntigravityAdapter.metadata property."""

    def test_returns_metadata(self, adapter: AntigravityAdapter):
        """Returns AdapterMetadata with correct values."""
        meta = adapter.metadata
        assert meta.name == "antigravity"
        assert meta.display_name == "Google Antigravity"
        assert meta.mcp_config_file is None  # Antigravity manages MCP through UI only


class TestAntigravityAdapterDirectories:
    """Tests for AntigravityAdapter directory methods."""

    def test_get_base_directory(self, adapter: AntigravityAdapter, temp_dir: Path):
        """Returns .agent directory (universal agent skills location)."""
        result = adapter.get_base_directory(temp_dir)
        assert result == temp_dir / ".agent"

    def test_get_skills_directory(self, adapter: AntigravityAdapter, temp_dir: Path):
        """Returns .agent/skills directory."""
        result = adapter.get_skills_directory(temp_dir)
        assert result == temp_dir / ".agent" / "skills"

    def test_get_mcp_config_path(self, adapter: AntigravityAdapter, temp_dir: Path):
        """Returns None (Antigravity manages MCP through UI only)."""
        result = adapter.get_mcp_config_path(temp_dir)
        assert result is None


class TestAntigravityAdapterPlanSkillInstallation:
    """Tests for AntigravityAdapter.plan_skill_installation()."""

    def test_creates_installation_plan(
        self,
        adapter: AntigravityAdapter,
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
        adapter: AntigravityAdapter,
        temp_dir: Path,
        sample_skill: SkillConfig,
        sample_manifest: PluginManifest,
    ):
        """Skill file is created as SKILL.md in .agent/skills/{skill}/."""
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        plan = adapter.plan_skill_installation(
            skill=sample_skill,
            plugin=sample_manifest,
            rendered_content="# Test",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        # Antigravity uses: .agent/skills/{skill-name}/SKILL.md
        expected_path = temp_dir / ".agent" / "skills" / "test-skill" / "SKILL.md"
        assert plan.files_to_write[0].path == expected_path

    def test_includes_frontmatter(
        self,
        adapter: AntigravityAdapter,
        temp_dir: Path,
        sample_skill: SkillConfig,
        sample_manifest: PluginManifest,
    ):
        """File content includes Antigravity-specific frontmatter."""
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


class TestAntigravityAdapterGenerateSkillFrontmatter:
    """Tests for AntigravityAdapter.generate_skill_frontmatter()."""

    def test_generates_antigravity_frontmatter(
        self,
        adapter: AntigravityAdapter,
        sample_skill: SkillConfig,
        sample_manifest: PluginManifest,
    ):
        """Generates Antigravity frontmatter with name and description."""
        frontmatter = adapter.generate_skill_frontmatter(sample_skill, sample_manifest)

        expected = """---
name: test-skill
description: A test skill for code review
---
"""
        assert frontmatter == expected

    def test_includes_metadata_fields(
        self, adapter: AntigravityAdapter, sample_manifest: PluginManifest
    ):
        """Includes additional metadata fields."""
        skill = SkillConfig(
            name="advanced-skill",
            description="Advanced skill with metadata",
            context="./context.md",
            metadata={"category": "code-quality", "author": "dex"},
        )

        frontmatter = adapter.generate_skill_frontmatter(skill, sample_manifest)

        expected = """---
name: advanced-skill
description: Advanced skill with metadata
category: code-quality
author: dex
---
"""
        assert frontmatter == expected


class TestAntigravityAdapterValidatePluginCompatibility:
    """Tests for AntigravityAdapter.validate_plugin_compatibility()."""

    def test_warns_about_mcp_servers(self, adapter: AntigravityAdapter):
        """Warns about MCP servers (Antigravity manages MCP through UI only)."""
        from dex.config.schemas import MCPServerConfig

        manifest = PluginManifest(
            name="test-plugin",
            version="1.0.0",
            description="Plugin with MCP",
            mcp_servers=[MCPServerConfig(name="test-server", type="command", source="npm:test")],
        )

        warnings = adapter.validate_plugin_compatibility(manifest)

        # Antigravity doesn't support project-level MCP config - should warn
        assert len(warnings) >= 1
        assert any("MCP" in w for w in warnings)
        assert any("UI" in w for w in warnings)

    def test_warns_about_subagents(self, adapter: AntigravityAdapter):
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

    def test_warns_about_commands(self, adapter: AntigravityAdapter):
        """Warns when plugin has commands (should use skills instead)."""
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
        self, adapter: AntigravityAdapter, sample_manifest: PluginManifest
    ):
        """No warnings for plugin with only skills."""
        warnings = adapter.validate_plugin_compatibility(sample_manifest)
        assert warnings == []


class TestAntigravityAdapterPreInstall:
    """Tests for AntigravityAdapter.pre_install()."""

    def test_creates_directories(
        self, adapter: AntigravityAdapter, temp_dir: Path, sample_manifest: PluginManifest
    ):
        """Creates .agent and skills directories."""
        adapter.pre_install(temp_dir, [sample_manifest])

        assert (temp_dir / ".agent").exists()
        assert (temp_dir / ".agent" / "skills").exists()

    def test_handles_existing_directories(
        self, adapter: AntigravityAdapter, temp_dir: Path, sample_manifest: PluginManifest
    ):
        """Handles existing directories without error."""
        (temp_dir / ".agent" / "skills").mkdir(parents=True)

        # Should not raise
        adapter.pre_install(temp_dir, [sample_manifest])


class TestAntigravityAdapterFilesHandling:
    """Tests for file handling in installation plans."""

    def test_adds_files_to_plan(
        self,
        adapter: AntigravityAdapter,
        temp_dir: Path,
        sample_manifest: PluginManifest,
    ):
        """Adds associated files to installation plan."""
        skill = SkillConfig(
            name="test-skill",
            description="Skill with files",
            context="./context.md",
            files=[FileTarget(src="scripts/helper.sh")],
        )
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()
        (source_dir / "scripts").mkdir()
        (source_dir / "scripts" / "helper.sh").write_text("#!/bin/bash")

        plan = adapter.plan_skill_installation(
            skill=skill,
            plugin=sample_manifest,
            rendered_content="# Test",
            project_root=temp_dir,
            source_dir=source_dir,
        )

        assert len(plan.files_to_copy) > 0
        assert any("helper.sh" in str(src) for src in plan.files_to_copy)


class TestAntigravityAdapterMCPConfig:
    """Tests for AntigravityAdapter MCP configuration (not supported - UI-managed only)."""

    def test_generate_mcp_config_returns_empty(
        self, adapter: AntigravityAdapter, temp_dir: Path, sample_manifest: PluginManifest
    ):
        """Returns empty dict (Antigravity manages MCP through UI only)."""
        from dex.config.schemas import MCPServerConfig

        mcp_server = MCPServerConfig(
            name="test-server",
            type="command",
            source="npm:@example/mcp-server",
        )
        source_dir = temp_dir / "plugin"
        source_dir.mkdir()

        config = adapter.generate_mcp_config(mcp_server, sample_manifest, temp_dir, source_dir)

        # Antigravity doesn't support project-level MCP config
        assert config == {}

    def test_merge_mcp_config_returns_existing(self, adapter: AntigravityAdapter):
        """Returns existing config unchanged (Antigravity doesn't support project-level MCP)."""
        existing = {"some": "config"}
        new_entries = {"test-server": {"command": "npx", "args": ["-y", "test"]}}

        result = adapter.merge_mcp_config(existing, new_entries)

        # Should return existing unchanged
        assert result == existing


class TestAntigravityAdapterGenerateRuleFrontmatter:
    """Tests for AntigravityAdapter.generate_rule_frontmatter()."""

    def test_returns_empty_string(
        self,
        adapter: AntigravityAdapter,
        sample_manifest: PluginManifest,
    ):
        """Antigravity uses plain markdown rules without YAML frontmatter."""
        rule = RuleConfig(
            name="test-rule",
            description="A test rule",
            context="./rules/test.md",
        )

        frontmatter = adapter.generate_rule_frontmatter(rule, sample_manifest)

        assert frontmatter == ""

    def test_returns_empty_even_with_metadata(
        self,
        adapter: AntigravityAdapter,
        sample_manifest: PluginManifest,
    ):
        """Returns empty even when rule has metadata (Antigravity ignores rule metadata)."""
        rule = RuleConfig(
            name="test-rule",
            description="A test rule with metadata",
            context="./rules/test.md",
            glob="**/*.py",
            metadata={"category": "testing"},
        )

        frontmatter = adapter.generate_rule_frontmatter(rule, sample_manifest)

        assert frontmatter == ""

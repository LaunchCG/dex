"""Tests for dex.config.schemas module."""

from pathlib import Path

import pytest
from pydantic import ValidationError

from dex.config.schemas import (
    AdapterMetadata,
    CommandConfig,
    ConditionalContext,
    EnvVariableConfig,
    FileTarget,
    FileToWrite,
    InstallationPlan,
    LockedPlugin,
    LockFile,
    MCPServerConfig,
    PluginManifest,
    PluginSpec,
    ProjectConfig,
    RuleConfig,
    SkillConfig,
    SubAgentConfig,
)


class TestSkillConfig:
    """Tests for SkillConfig model."""

    def test_valid_skill(self):
        """Create a valid skill config."""
        skill = SkillConfig(
            name="test-skill",
            description="A test skill",
            context="./context/skill.md",
        )
        assert skill.name == "test-skill"
        assert skill.description == "A test skill"
        assert skill.context == "./context/skill.md"
        assert skill.files is None
        assert skill.metadata == {}

    def test_skill_with_files(self):
        """Create skill with files."""
        skill = SkillConfig(
            name="test-skill",
            description="Skill with files",
            context="./context.md",
            files=[
                FileTarget(src="tools/file1.txt"),
                FileTarget(src="tools/file2.txt"),
            ],
        )
        assert skill.files is not None
        assert len(skill.files) == 2
        assert skill.files[0].src == "tools/file1.txt"
        assert skill.files[0].dest == "file1.txt"  # Defaults to basename

    def test_skill_with_metadata(self):
        """Create skill with metadata."""
        skill = SkillConfig(
            name="test-skill",
            description="Skill with metadata",
            context="./context.md",
            metadata={"author": "test", "tags": ["utility"]},
        )
        assert skill.metadata["author"] == "test"

    def test_skill_with_template_files(self):
        """Create skill with template_files."""
        skill = SkillConfig(
            name="test-skill",
            description="Skill with template files",
            context="./context.md",
            template_files=[
                FileTarget(src="templates/config.py.j2", dest="config.py"),
                FileTarget(src="templates/settings.yaml.j2", dest="settings.yaml"),
            ],
        )
        assert skill.template_files is not None
        assert len(skill.template_files) == 2
        assert skill.template_files[0].src == "templates/config.py.j2"
        assert skill.template_files[0].dest == "config.py"

    def test_skill_with_files_and_template_files(self):
        """Create skill with both files and template_files."""
        skill = SkillConfig(
            name="test-skill",
            description="Skill with both file types",
            context="./context.md",
            files=[FileTarget(src="static.txt")],
            template_files=[FileTarget(src="config.py.j2")],
        )
        assert skill.files is not None
        assert len(skill.files) == 1
        assert skill.files[0].src == "static.txt"
        assert skill.template_files is not None
        assert len(skill.template_files) == 1
        assert skill.template_files[0].src == "config.py.j2"


class TestCommandConfig:
    """Tests for CommandConfig model."""

    def test_valid_command(self):
        """Create a valid command config."""
        cmd = CommandConfig(
            name="test-cmd",
            description="A test command",
            context="./context.md",
        )
        assert cmd.name == "test-cmd"
        assert cmd.description == "A test command"
        assert cmd.skills == []

    def test_command_with_skills(self):
        """Create command with associated skills."""
        cmd = CommandConfig(
            name="test-cmd",
            description="Command that uses skills",
            context="./context.md",
            skills=["skill1", "skill2"],
        )
        assert cmd.skills == ["skill1", "skill2"]


class TestSubAgentConfig:
    """Tests for SubAgentConfig model."""

    def test_valid_subagent(self):
        """Create a valid sub-agent config."""
        agent = SubAgentConfig(
            name="test-agent",
            description="A test agent",
            context="./context.md",
        )
        assert agent.name == "test-agent"
        assert agent.description == "A test agent"
        assert agent.skills == []
        assert agent.commands == []

    def test_subagent_with_skills_and_commands(self):
        """Create sub-agent with skills and commands."""
        agent = SubAgentConfig(
            name="test-agent",
            description="Agent with capabilities",
            context="./context.md",
            skills=["skill1"],
            commands=["cmd1", "cmd2"],
        )
        assert agent.skills == ["skill1"]
        assert agent.commands == ["cmd1", "cmd2"]


class TestMCPServerConfig:
    """Tests for MCPServerConfig model."""

    def test_valid_bundled_server(self):
        """Create a valid bundled MCP server."""
        server = MCPServerConfig(
            name="test-server",
            type="bundled",
            path="./server.js",
        )
        assert server.name == "test-server"
        assert server.type == "bundled"
        assert server.path == "./server.js"

    def test_valid_remote_server(self):
        """Create a valid remote MCP server."""
        server = MCPServerConfig(
            name="test-server",
            type="remote",
            source="npm:@example/server",
        )
        assert server.type == "remote"
        assert server.source == "npm:@example/server"

    def test_bundled_requires_path(self):
        """Bundled server requires path."""
        with pytest.raises(ValidationError, match="must specify a 'path'"):
            MCPServerConfig(name="test", type="bundled")

    def test_remote_requires_source(self):
        """Remote server requires source."""
        with pytest.raises(ValidationError, match="must specify a 'source'"):
            MCPServerConfig(name="test", type="remote")

    def test_server_with_config(self):
        """Server with additional config."""
        server = MCPServerConfig(
            name="test",
            type="bundled",
            path="./server.js",
            config={"args": ["--port", "8080"], "env": {"DEBUG": "true"}},
        )
        assert server.config["args"] == ["--port", "8080"]

    def test_platform_specific_path(self):
        """Server with platform-specific paths."""
        server = MCPServerConfig(
            name="test",
            type="bundled",
            path={"windows": "./server.exe", "unix": "./server"},
        )
        assert isinstance(server.path, dict)
        assert server.path["windows"] == "./server.exe"


class TestPluginManifest:
    """Tests for PluginManifest model."""

    def test_valid_manifest(self):
        """Create a valid plugin manifest."""
        manifest = PluginManifest(
            name="test-plugin",
            version="1.0.0",
            description="A test plugin",
        )
        assert manifest.name == "test-plugin"
        assert manifest.version == "1.0.0"
        assert manifest.skills == []

    def test_manifest_with_skills(self):
        """Create manifest with skills."""
        manifest = PluginManifest(
            name="test-plugin",
            version="1.0.0",
            description="Test",
            skills=[
                SkillConfig(name="skill1", description="Skill 1", context="./s1.md"),
                SkillConfig(name="skill2", description="Skill 2", context="./s2.md"),
            ],
        )
        assert len(manifest.skills) == 2

    def test_invalid_name_format(self):
        """Invalid plugin name raises error."""
        with pytest.raises(ValidationError, match="Plugin name must start"):
            PluginManifest(
                name="Invalid Plugin",  # Contains space and uppercase
                version="1.0.0",
                description="Test",
            )

    def test_invalid_name_empty(self):
        """Empty plugin name raises error."""
        with pytest.raises(ValidationError, match="cannot be empty"):
            PluginManifest(name="", version="1.0.0", description="Test")

    def test_invalid_version_format(self):
        """Invalid version format raises error."""
        with pytest.raises(ValidationError, match="Invalid semver"):
            PluginManifest(
                name="test-plugin",
                version="invalid",
                description="Test",
            )

    def test_manifest_with_dependencies(self):
        """Manifest with dependencies."""
        manifest = PluginManifest(
            name="test-plugin",
            version="1.0.0",
            description="Test",
            dependencies={"other-plugin": "^1.0.0"},
        )
        assert manifest.dependencies["other-plugin"] == "^1.0.0"

    def test_manifest_with_rules(self):
        """Manifest with rules list."""
        manifest = PluginManifest(
            name="test-plugin",
            version="1.0.0",
            description="Test",
            rules=[
                RuleConfig(
                    name="code-style",
                    description="Code style rules",
                    context="./rules/style.md",
                )
            ],
        )
        assert len(manifest.rules) == 1
        assert manifest.rules[0].name == "code-style"


class TestPluginSpec:
    """Tests for PluginSpec model."""

    def test_spec_with_version(self):
        """Create spec with version."""
        spec = PluginSpec(version="^1.0.0")
        assert spec.version == "^1.0.0"
        assert spec.source is None

    def test_spec_with_source(self):
        """Create spec with source."""
        spec = PluginSpec(source="file:./plugins/local")
        assert spec.source == "file:./plugins/local"
        assert spec.version is None

    def test_spec_requires_version_or_source(self):
        """Spec requires either version or source."""
        with pytest.raises(ValidationError, match="must have either"):
            PluginSpec()

    def test_spec_cannot_have_both(self):
        """Spec cannot have both version and source."""
        with pytest.raises(ValidationError, match="cannot have both"):
            PluginSpec(version="1.0.0", source="file:./local")

    def test_spec_with_registry(self):
        """Spec with registry reference."""
        spec = PluginSpec(version="^1.0.0", registry="custom")
        assert spec.registry == "custom"


class TestProjectConfig:
    """Tests for ProjectConfig model."""

    def test_valid_config(self):
        """Create a valid project config."""
        config = ProjectConfig(agent="claude-code")
        assert config.agent == "claude-code"
        assert config.plugins == {}
        assert config.registries == {}

    def test_config_with_plugins(self):
        """Config with plugins."""
        config = ProjectConfig(
            agent="claude-code",
            plugins={
                "plugin1": "^1.0.0",
                "plugin2": PluginSpec(source="file:./local"),
            },
        )
        assert "plugin1" in config.plugins
        assert config.plugins["plugin1"] == "^1.0.0"

    def test_config_with_registries(self):
        """Config with registries."""
        config = ProjectConfig(
            agent="claude-code",
            registries={
                "local": "file:./registry",
                "remote": "https://example.com/registry",
            },
            default_registry="local",
        )
        assert config.registries["local"] == "file:./registry"

    def test_default_registry_must_exist(self):
        """Default registry must exist in registries."""
        with pytest.raises(ValidationError, match="not found in registries"):
            ProjectConfig(
                agent="claude-code",
                registries={"local": "file:./registry"},
                default_registry="nonexistent",
            )

    def test_valid_agent_types(self):
        """Test valid agent types."""
        from typing import get_args

        from dex.config.schemas import AgentType

        for agent in get_args(AgentType):
            config = ProjectConfig(agent=agent)
            assert config.agent == agent


class TestLockFile:
    """Tests for LockFile model."""

    def test_empty_lockfile(self):
        """Create an empty lock file."""
        lockfile = LockFile(agent="claude-code")
        assert lockfile.version == "1.0"
        assert lockfile.plugins == {}

    def test_lockfile_with_plugins(self):
        """Lock file with locked plugins."""
        lockfile = LockFile(
            agent="claude-code",
            plugins={
                "test-plugin": LockedPlugin(
                    version="1.0.0",
                    resolved="file:///path/to/plugin",
                    integrity="sha512-abc123",
                ),
            },
        )
        assert lockfile.plugins["test-plugin"].version == "1.0.0"


class TestLockedPlugin:
    """Tests for LockedPlugin model."""

    def test_locked_plugin(self):
        """Create a locked plugin entry."""
        locked = LockedPlugin(
            version="1.0.0",
            resolved="file:///path",
            integrity="sha512-abc123",
        )
        assert locked.version == "1.0.0"
        assert locked.dependencies == {}

    def test_locked_plugin_with_deps(self):
        """Locked plugin with dependencies."""
        locked = LockedPlugin(
            version="1.0.0",
            resolved="file:///path",
            integrity="sha512-abc123",
            dependencies={"dep1": "2.0.0"},
        )
        assert locked.dependencies["dep1"] == "2.0.0"


class TestInstallationPlan:
    """Tests for InstallationPlan model."""

    def test_empty_plan(self):
        """Create an empty installation plan."""
        plan = InstallationPlan()
        assert plan.directories_to_create == []
        assert plan.files_to_write == []
        assert plan.files_to_copy == {}
        assert plan.template_files_to_render == {}

    def test_plan_with_template_files_to_render(self):
        """Plan with template files to render."""
        plan = InstallationPlan(
            template_files_to_render={
                Path("/tmp/config.py.j2"): Path("/tmp/output/config.py"),
                Path("/tmp/settings.yaml.j2"): Path("/tmp/output/settings.yaml"),
            }
        )
        assert len(plan.template_files_to_render) == 2
        assert plan.template_files_to_render[Path("/tmp/config.py.j2")] == Path(
            "/tmp/output/config.py"
        )

    def test_plan_with_directories(self):
        """Plan with directories to create."""
        plan = InstallationPlan(directories_to_create=[Path("/tmp/dir1"), Path("/tmp/dir2")])
        assert len(plan.directories_to_create) == 2

    def test_plan_with_files_to_write(self):
        """Plan with files to write."""
        plan = InstallationPlan(
            files_to_write=[FileToWrite(path=Path("/tmp/file.txt"), content="content")]
        )
        assert len(plan.files_to_write) == 1
        assert plan.files_to_write[0].content == "content"


class TestFileToWrite:
    """Tests for FileToWrite model."""

    def test_file_to_write(self):
        """Create a file to write."""
        ftw = FileToWrite(path=Path("/tmp/test.txt"), content="content")
        assert ftw.path == Path("/tmp/test.txt")
        assert ftw.chmod is None

    def test_file_with_chmod(self):
        """File with chmod."""
        ftw = FileToWrite(path=Path("/tmp/script.sh"), content="#!/bin/bash", chmod="755")
        assert ftw.chmod == "755"


class TestAdapterMetadata:
    """Tests for AdapterMetadata model."""

    def test_adapter_metadata(self):
        """Create adapter metadata."""
        meta = AdapterMetadata(
            name="test-adapter",
            display_name="Test Adapter",
            description="A test adapter",
            mcp_config_file="settings.json",
        )
        assert meta.name == "test-adapter"
        assert meta.display_name == "Test Adapter"
        assert meta.mcp_config_file == "settings.json"

    def test_adapter_metadata_no_mcp(self):
        """Create adapter metadata without MCP config file."""
        meta = AdapterMetadata(
            name="no-mcp-adapter",
            display_name="No MCP Adapter",
            description="An adapter without MCP support",
            mcp_config_file=None,
        )
        assert meta.mcp_config_file is None


class TestConditionalContext:
    """Tests for ConditionalContext model."""

    def test_conditional_context(self):
        """Create conditional context."""
        ctx = ConditionalContext(path="./windows.md", **{"if": "platform.os == 'windows'"})
        assert ctx.path == "./windows.md"
        assert ctx.if_ == "platform.os == 'windows'"


class TestEnvVariableConfig:
    """Tests for EnvVariableConfig model."""

    def test_required_env_var(self):
        """Create required env var config."""
        env = EnvVariableConfig(description="API key", required=True)
        assert env.required is True
        assert env.default is None

    def test_optional_env_var_with_default(self):
        """Create optional env var with default."""
        env = EnvVariableConfig(
            description="Log level",
            required=False,
            default="info",
        )
        assert env.required is False
        assert env.default == "info"


class TestFileTarget:
    """Tests for FileTarget model."""

    def test_file_target_with_explicit_dest(self):
        """Create a file target with explicit dest."""
        target = FileTarget(src="tools/file.txt", dest="renamed.txt")
        assert target.src == "tools/file.txt"
        assert target.dest == "renamed.txt"
        assert target.chmod is None

    def test_file_target_dest_defaults_to_basename(self):
        """Dest defaults to basename of src when not specified."""
        target = FileTarget(src="tools/nested/calculator.py")
        assert target.src == "tools/nested/calculator.py"
        assert target.dest == "calculator.py"  # Basename of src

    def test_file_target_with_chmod(self):
        """Create file target with chmod."""
        target = FileTarget(src="scripts/run.sh", dest="run.sh", chmod="755")
        assert target.chmod == "755"

    def test_file_target_from_dict(self):
        """FileTarget can be created from dict (for JSON parsing)."""
        target = FileTarget(**{"src": "schemas/dora.json", "dest": "schema.json"})
        assert target.src == "schemas/dora.json"
        assert target.dest == "schema.json"

"""Shared fixtures for Dex tests."""

import json
import shutil
import tempfile
from collections.abc import Generator
from pathlib import Path

import pytest

from dex.config.schemas import (
    CommandConfig,
    MCPServerConfig,
    PluginManifest,
    PluginSpec,
    ProjectConfig,
    SkillConfig,
    SubAgentConfig,
)
from dex.core.project import Project


@pytest.fixture
def temp_dir() -> Generator[Path, None, None]:
    """Create a temporary directory."""
    path = Path(tempfile.mkdtemp(prefix="dex_test_"))
    yield path
    if path.exists():
        shutil.rmtree(path)


@pytest.fixture
def temp_project(temp_dir: Path) -> Path:
    """Create a temporary project directory."""
    project_dir = temp_dir / "test-project"
    project_dir.mkdir()
    return project_dir


@pytest.fixture
def sample_skill_config() -> SkillConfig:
    """Sample skill configuration for testing."""
    return SkillConfig(
        name="test-skill",
        description="A test skill for unit tests",
        context="./context/skill.md",
        files=["./files/config.json"],
        metadata={"author": "test"},
    )


@pytest.fixture
def sample_command_config() -> CommandConfig:
    """Sample command configuration for testing."""
    return CommandConfig(
        name="test-command",
        description="A test command for unit tests",
        context="./context/command.md",
        files=None,
        metadata={},
    )


@pytest.fixture
def sample_subagent_config() -> SubAgentConfig:
    """Sample sub-agent configuration for testing."""
    return SubAgentConfig(
        name="test-agent",
        description="A test sub-agent for unit tests",
        context="./context/agent.md",
        files=None,
        metadata={},
    )


@pytest.fixture
def sample_mcp_server_bundled() -> MCPServerConfig:
    """Sample bundled MCP server configuration."""
    return MCPServerConfig(
        name="test-mcp",
        type="bundled",
        path="./servers/server.js",
        config={"env": {"API_KEY": "${API_KEY}"}},
    )


@pytest.fixture
def sample_mcp_server_remote() -> MCPServerConfig:
    """Sample remote MCP server configuration."""
    return MCPServerConfig(
        name="test-remote-mcp",
        type="remote",
        source="npm:@example/mcp-server",
        version="1.0.0",
    )


@pytest.fixture
def sample_plugin_manifest(sample_skill_config: SkillConfig) -> PluginManifest:
    """Sample plugin manifest for testing."""
    return PluginManifest(
        name="test-plugin",
        version="1.0.0",
        description="A test plugin",
        skills=[sample_skill_config],
        commands=[],
        sub_agents=[],
        mcp_servers=[],
        dependencies={},
    )


@pytest.fixture
def sample_plugin_manifest_full(
    sample_skill_config: SkillConfig,
    sample_command_config: CommandConfig,
    sample_subagent_config: SubAgentConfig,
    sample_mcp_server_bundled: MCPServerConfig,
) -> PluginManifest:
    """Sample plugin manifest with all component types."""
    return PluginManifest(
        name="full-plugin",
        version="2.0.0",
        description="A full-featured test plugin",
        skills=[sample_skill_config],
        commands=[sample_command_config],
        sub_agents=[sample_subagent_config],
        mcp_servers=[sample_mcp_server_bundled],
        dependencies={"other-plugin": "^1.0.0"},
    )


@pytest.fixture
def sample_project_config() -> ProjectConfig:
    """Sample project configuration for testing."""
    return ProjectConfig(
        agent="claude-code",
        project_name="test-project",
        plugins={
            "test-plugin": "^1.0.0",
            "other-plugin": PluginSpec(source="file:./plugins/other"),
        },
        registries={"local": "file:./registry"},
        default_registry="local",
    )


@pytest.fixture
def temp_registry(temp_dir: Path) -> Path:
    """Create a temporary registry with test plugins."""
    registry_dir = temp_dir / "registry"
    registry_dir.mkdir()

    # Create registry.json
    registry_data = {
        "packages": {
            "test-plugin": {
                "versions": ["1.0.0", "1.1.0", "2.0.0"],
                "latest": "2.0.0",
            },
            "other-plugin": {
                "versions": ["0.9.0", "1.0.0"],
                "latest": "1.0.0",
            },
        }
    }
    with open(registry_dir / "registry.json", "w") as f:
        json.dump(registry_data, f)

    return registry_dir


@pytest.fixture
def temp_plugin_dir(temp_dir: Path) -> Path:
    """Create a temporary plugin directory with package.json."""
    plugin_dir = temp_dir / "test-plugin"
    plugin_dir.mkdir()

    # Create package.json
    manifest = {
        "name": "test-plugin",
        "version": "1.0.0",
        "description": "A test plugin",
        "skills": [
            {
                "name": "test-skill",
                "description": "A test skill",
                "context": "./context/skill.md",
            }
        ],
    }
    with open(plugin_dir / "package.json", "w") as f:
        json.dump(manifest, f)

    # Create context file
    context_dir = plugin_dir / "context"
    context_dir.mkdir()
    (context_dir / "skill.md").write_text("# Test Skill\n\nThis is a test skill.")

    return plugin_dir


@pytest.fixture
def initialized_project(temp_project: Path) -> Project:
    """Project with dex.yaml initialized."""
    project = Project.init(temp_project, "claude-code", project_name="test-project")
    return project


@pytest.fixture
def project_with_plugins(initialized_project: Project, temp_registry: Path) -> Project:
    """Project with plugins and registry configured."""
    initialized_project.add_plugin("test-plugin", PluginSpec(version="^1.0.0"))
    initialized_project._config.registries["local"] = f"file://{temp_registry}"
    initialized_project._config.default_registry = "local"
    initialized_project.save()
    return initialized_project

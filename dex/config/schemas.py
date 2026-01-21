"""Pydantic schemas for Dex configuration files.

This module defines the data models for:
- dex.yaml (project configuration)
- dex.lock (lockfile)
- package.json (plugin manifest)
"""

from pathlib import Path
from typing import Any, Literal

from pydantic import BaseModel, Field, field_validator, model_validator

# =============================================================================
# Common Types
# =============================================================================

AgentType = Literal["claude-code", "cursor", "codex", "antigravity", "github-copilot"]
PlatformOS = Literal["windows", "linux", "macos", "unix"]
MCPServerType = Literal["command", "http"]
ComponentType = Literal["skill", "command", "sub_agent", "instruction", "rule", "prompt"]


# =============================================================================
# File Specification Models
# =============================================================================


class FileTarget(BaseModel):
    """A file with source path and optional destination.

    - src: Path relative to plugin root (required)
    - dest: Destination filename (optional, defaults to basename of src)
    - chmod: File permissions (optional, e.g., "755")
    """

    src: str
    dest: str | None = None
    chmod: str | None = None

    @model_validator(mode="after")
    def set_default_dest(self) -> "FileTarget":
        """Default dest to basename of src if not specified."""
        if self.dest is None:
            # Use Path to get just the filename
            from pathlib import PurePosixPath

            self.dest = PurePosixPath(self.src).name
        return self


# Type alias for file specifications - always a list of FileTarget
FileSpec = list[FileTarget]

# Template files use the same spec - they're rendered through Jinja2 before writing
TemplateFileSpec = list[FileTarget]


# =============================================================================
# Context File Models
# =============================================================================


class ConditionalContext(BaseModel):
    """A context file with a condition."""

    path: str
    if_: str = Field(alias="if")

    model_config = {"populate_by_name": True}


# Context can be a single path, list of paths, or list with conditions
ContextSpec = str | list[str | ConditionalContext]


# =============================================================================
# Component Models (Skills, Commands, Sub-Agents)
# =============================================================================


class SkillConfig(BaseModel):
    """Skill definition within a plugin."""

    name: str
    description: str
    context: ContextSpec
    files: FileSpec | None = None
    template_files: TemplateFileSpec | None = None
    adapters: list[str] | None = None  # If set, only install on these adapters
    metadata: dict[str, Any] = Field(default_factory=dict)


class CommandConfig(BaseModel):
    """Command definition within a plugin."""

    name: str
    description: str
    context: ContextSpec
    skills: list[str] = Field(default_factory=list)  # Skills this command uses
    allowed_tools: list[str] | str | None = None  # Tools this command can use (e.g., "Bash(uv:*)")
    files: FileSpec | None = None
    template_files: TemplateFileSpec | None = None
    adapters: list[str] | None = None  # If set, only install on these adapters
    metadata: dict[str, Any] = Field(default_factory=dict)


class SubAgentConfig(BaseModel):
    """Sub-agent definition within a plugin."""

    name: str
    description: str
    context: ContextSpec
    skills: list[str] = Field(default_factory=list)  # Skills this agent can use
    commands: list[str] = Field(default_factory=list)  # Commands this agent can use
    allowed_tools: list[str] | str | None = None  # Tools this agent can use
    files: FileSpec | None = None
    template_files: TemplateFileSpec | None = None
    adapters: list[str] | None = None  # If set, only install on these adapters
    metadata: dict[str, Any] = Field(default_factory=dict)


# =============================================================================
# First-Class Platform-Specific Components
# =============================================================================


class InstructionConfig(BaseModel):
    """Instruction definition (GitHub Copilot .instructions.md files).

    Instructions are path-specific guidance that can be scoped to
    specific file patterns using the applyTo field.
    """

    model_config = {"populate_by_name": True}

    name: str
    description: str
    context: ContextSpec
    apply_to: str | list[str] | None = Field(
        default=None, alias="applyTo"
    )  # File glob patterns (e.g., "**/*.py")
    exclude_agent: str | None = Field(
        default=None, alias="excludeAgent"
    )  # Agent to exclude (e.g., "code-review")
    files: FileSpec | None = None
    template_files: TemplateFileSpec | None = None
    adapters: list[str] | None = None  # If set, only install on these adapters
    metadata: dict[str, Any] = Field(default_factory=dict)


class RuleConfig(BaseModel):
    """Rule definition (Cursor .cursor/rules/, Copilot copilot-instructions.md).

    Rules are project-wide or scoped guidelines that inform AI behavior.
    """

    name: str
    description: str
    context: ContextSpec
    glob: str | None = None  # File pattern scope (Cursor .mdc files)
    paths: str | list[str] | None = None  # File pattern scope (Claude Code)
    always: bool = False  # Always apply this rule (Cursor)
    files: FileSpec | None = None
    template_files: TemplateFileSpec | None = None
    adapters: list[str] | None = None  # If set, only install on these adapters
    metadata: dict[str, Any] = Field(default_factory=dict)


class PromptConfig(BaseModel):
    """Prompt definition (reusable prompts/system instructions).

    Prompts are reusable text snippets that can be referenced or
    invoked by the AI assistant.
    """

    name: str
    description: str
    context: ContextSpec
    trigger: str | None = None  # Optional trigger phrase
    files: FileSpec | None = None
    template_files: TemplateFileSpec | None = None
    adapters: list[str] | None = None  # If set, only install on these adapters
    metadata: dict[str, Any] = Field(default_factory=dict)


# =============================================================================
# Claude Settings Models
# =============================================================================


class ClaudeSettingsConfig(BaseModel):
    """Claude Code settings configuration within a plugin.

    Allows plugins to contribute permission patterns to settings.json.
    Only permissions are managed - other settings are user-controlled.
    """

    allow: list[str] = Field(default_factory=list)
    deny: list[str] = Field(default_factory=list)


# =============================================================================
# MCP Server Models
# =============================================================================


class MCPServerConfig(BaseModel):
    """MCP server configuration within a plugin.

    Two types supported:
    - command: Runs a local command (stdio transport)
    - http: Connects to an HTTP endpoint

    For command type, you can either specify command/args directly,
    or use a source shortcut (npm:, uvx:, pip:) that expands to command/args.
    """

    name: str
    description: str = ""
    type: MCPServerType

    # For command type
    command: str | None = None
    args: list[str] = Field(default_factory=list)
    env: dict[str, str] = Field(default_factory=dict)
    source: str | None = None  # Shortcut: npm:, uvx:, pip: expand to command/args

    # For http type
    url: str | None = None

    @model_validator(mode="after")
    def validate_server_type(self) -> "MCPServerConfig":
        """Validate fields based on server type."""
        if self.type == "command":
            if not self.source and not self.command:
                raise ValueError("Command MCP servers must specify 'source' or 'command'")
        elif self.type == "http" and not self.url:
            raise ValueError("HTTP MCP servers must specify 'url'")
        return self


# =============================================================================
# Environment Variable Models
# =============================================================================


class EnvVariableConfig(BaseModel):
    """Environment variable declaration."""

    description: str
    required: bool = True
    default: str | None = None


# =============================================================================
# Agent File Config
# =============================================================================


class AgentFileConfig(BaseModel):
    """Content to inject into the main agent instruction file.

    This configures content that should be appended to shared agent files
    like CLAUDE.md (Claude Code) or AGENTS.md (Codex). Content is managed
    using markers to allow multiple plugins to contribute to the same file.
    """

    context: ContextSpec  # Path(s) to markdown content to inject
    files: FileSpec | None = None  # Optional additional files to copy
    template_files: TemplateFileSpec | None = None  # Optional files to render with Jinja2
    metadata: dict[str, Any] = Field(default_factory=dict)

    @property
    def name(self) -> str:
        """Agent file config uses a fixed name for context building."""
        return "agent_file"


# =============================================================================
# Plugin Manifest (package.json)
# =============================================================================


class PluginManifest(BaseModel):
    """Plugin manifest (package.json) schema.

    This defines the structure of a plugin's package.json file.
    """

    name: str
    version: str
    description: str

    # Core components (Claude Code native)
    skills: list[SkillConfig] = Field(default_factory=list)
    commands: list[CommandConfig] = Field(default_factory=list)
    sub_agents: list[SubAgentConfig] = Field(default_factory=list)

    # First-class platform components
    instructions: list[InstructionConfig] = Field(default_factory=list)  # GitHub Copilot
    rules: list[RuleConfig] = Field(default_factory=list)  # Cursor, Copilot
    prompts: list[PromptConfig] = Field(default_factory=list)  # Various platforms

    # Infrastructure
    mcp_servers: list[MCPServerConfig] = Field(default_factory=list)

    # Claude Code settings (permissions)
    claude_settings: ClaudeSettingsConfig | None = None

    # Agent file injection (for CLAUDE.md, AGENTS.md, etc.)
    agent_file: AgentFileConfig | None = None

    # First-class file resources (copied to plugin directory)
    files: FileSpec = Field(default_factory=list)
    # First-class template files (rendered with Jinja2 before writing)
    template_files: TemplateFileSpec = Field(default_factory=list)

    dependencies: dict[str, str] = Field(default_factory=dict)
    env_variables: dict[str, EnvVariableConfig] = Field(default_factory=dict)
    metadata: dict[str, Any] = Field(default_factory=dict)

    @field_validator("name")
    @classmethod
    def validate_name(cls, v: str) -> str:
        """Validate plugin name format."""
        if not v:
            raise ValueError("Plugin name cannot be empty")
        # Allow alphanumeric, hyphens, and underscores
        import re

        if not re.match(r"^[a-z0-9][a-z0-9_-]*$", v):
            raise ValueError(
                "Plugin name must start with alphanumeric and contain only "
                "lowercase letters, numbers, hyphens, and underscores"
            )
        return v

    @field_validator("version")
    @classmethod
    def validate_version(cls, v: str) -> str:
        """Validate semver format."""
        import re

        # Basic semver pattern
        pattern = r"^\d+\.\d+\.\d+(-[a-zA-Z0-9.-]+)?(\+[a-zA-Z0-9.-]+)?$"
        if not re.match(pattern, v):
            raise ValueError(f"Invalid semver version: {v}")
        return v


# =============================================================================
# Plugin Spec (within dex.yaml)
# =============================================================================


class PluginSpec(BaseModel):
    """Plugin specification in project configuration.

    Can specify version, registry, or direct source.
    """

    version: str | None = None
    registry: str | None = None
    source: str | None = None

    @model_validator(mode="after")
    def validate_spec(self) -> "PluginSpec":
        """Validate that either version/registry or source is specified."""
        has_version = self.version is not None
        has_source = self.source is not None
        if not has_version and not has_source:
            raise ValueError("Plugin spec must have either 'version' or 'source'")
        if has_version and has_source:
            raise ValueError("Plugin spec cannot have both 'version' and 'source'")
        return self


# =============================================================================
# Project Configuration (dex.yaml)
# =============================================================================


class ProjectConfig(BaseModel):
    """Project configuration (dex.yaml) schema."""

    agent: AgentType
    project_name: str | None = None
    registries: dict[str, str] = Field(default_factory=dict)
    default_registry: str | None = None
    plugins: dict[str, str | PluginSpec] = Field(default_factory=dict)
    config: dict[str, Any] = Field(default_factory=dict)

    @model_validator(mode="after")
    def validate_default_registry(self) -> "ProjectConfig":
        """Validate that default_registry exists in registries."""
        if self.default_registry and self.default_registry not in self.registries:
            raise ValueError(f"default_registry '{self.default_registry}' not found in registries")
        return self


# =============================================================================
# Lock File (dex.lock)
# =============================================================================


class LockedPlugin(BaseModel):
    """A locked plugin entry in the lock file."""

    version: str
    resolved: str
    integrity: str
    dependencies: dict[str, str] = Field(default_factory=dict)


class LockFile(BaseModel):
    """Lock file (dex.lock) schema."""

    version: str = "1.0"
    agent: AgentType
    plugins: dict[str, LockedPlugin] = Field(default_factory=dict)


# =============================================================================
# Installation Plan Models
# =============================================================================


class FileToWrite(BaseModel):
    """Represents a file to be written during installation."""

    path: Path
    content: str
    chmod: str | None = None


class InstallationPlan(BaseModel):
    """Plan for installing a plugin component.

    This is returned by adapters and executed by the installer.
    """

    directories_to_create: list[Path] = Field(default_factory=list)
    files_to_write: list[FileToWrite] = Field(default_factory=list)
    files_to_copy: dict[Path, Path] = Field(default_factory=dict)  # src -> dest
    template_files_to_render: dict[Path, Path] = Field(default_factory=dict)  # src -> dest
    mcp_config_updates: dict[str, Any] = Field(default_factory=dict)

    model_config = {"arbitrary_types_allowed": True}


# =============================================================================
# Adapter Metadata
# =============================================================================


class AdapterMetadata(BaseModel):
    """Metadata about a platform adapter.

    Adapters declare what they support by implementing the corresponding
    plan_*_installation methods. The base class returns empty plans for
    unsupported types, so adapters only override what they actually handle.
    """

    name: str
    display_name: str
    description: str
    mcp_config_file: str | None = None  # e.g., ".mcp.json", None if no MCP support


# =============================================================================
# Dex Manifest (.dex/manifest.json)
# =============================================================================


class PluginFiles(BaseModel):
    """Files managed by a single plugin."""

    files: list[str] = Field(default_factory=list)  # Relative to project root
    directories: list[str] = Field(default_factory=list)  # Relative to project root
    mcp_servers: list[str] = Field(default_factory=list)  # Server names added to .mcp.json
    claude_settings_allow: list[str] = Field(default_factory=list)  # Permission allow patterns
    claude_settings_deny: list[str] = Field(default_factory=list)  # Permission deny patterns


class DexManifest(BaseModel):
    """Manifest tracking all files managed by dex.

    Stored at .dex/manifest.json in the project root.
    """

    version: str = "1.0"
    plugins: dict[str, PluginFiles] = Field(default_factory=dict)

    def add_file(self, plugin_name: str, file_path: str) -> None:
        """Record a file as managed by a plugin."""
        if plugin_name not in self.plugins:
            self.plugins[plugin_name] = PluginFiles()
        if file_path not in self.plugins[plugin_name].files:
            self.plugins[plugin_name].files.append(file_path)

    def add_directory(self, plugin_name: str, dir_path: str) -> None:
        """Record a directory as managed by a plugin."""
        if plugin_name not in self.plugins:
            self.plugins[plugin_name] = PluginFiles()
        if dir_path not in self.plugins[plugin_name].directories:
            self.plugins[plugin_name].directories.append(dir_path)

    def add_mcp_server(self, plugin_name: str, server_name: str) -> None:
        """Record an MCP server as added by a plugin."""
        if plugin_name not in self.plugins:
            self.plugins[plugin_name] = PluginFiles()
        if server_name not in self.plugins[plugin_name].mcp_servers:
            self.plugins[plugin_name].mcp_servers.append(server_name)

    def get_plugin_files(self, plugin_name: str) -> PluginFiles | None:
        """Get all files managed by a plugin."""
        return self.plugins.get(plugin_name)

    def remove_plugin(self, plugin_name: str) -> PluginFiles | None:
        """Remove a plugin from the manifest and return its files."""
        return self.plugins.pop(plugin_name, None)

    def get_mcp_servers_to_remove(self, plugin_name: str) -> list[str]:
        """Get MCP servers that should be removed when a plugin is uninstalled.

        Only returns servers that are not used by any other plugin.
        """
        plugin_files = self.plugins.get(plugin_name)
        if not plugin_files:
            return []

        # Get servers used by other plugins
        other_servers: set[str] = set()
        for name, files in self.plugins.items():
            if name != plugin_name:
                other_servers.update(files.mcp_servers)

        # Return servers only used by this plugin
        return [s for s in plugin_files.mcp_servers if s not in other_servers]

    def add_claude_settings_allow(self, plugin_name: str, pattern: str) -> None:
        """Record a permission allow pattern as added by a plugin."""
        if plugin_name not in self.plugins:
            self.plugins[plugin_name] = PluginFiles()
        if pattern not in self.plugins[plugin_name].claude_settings_allow:
            self.plugins[plugin_name].claude_settings_allow.append(pattern)

    def add_claude_settings_deny(self, plugin_name: str, pattern: str) -> None:
        """Record a permission deny pattern as added by a plugin."""
        if plugin_name not in self.plugins:
            self.plugins[plugin_name] = PluginFiles()
        if pattern not in self.plugins[plugin_name].claude_settings_deny:
            self.plugins[plugin_name].claude_settings_deny.append(pattern)

    def get_claude_settings_to_remove(self, plugin_name: str) -> dict[str, list[str]]:
        """Get settings that should be removed when uninstalling a plugin.

        Returns dict with keys: 'allow', 'deny'
        Only returns items not used by any other plugin.
        """
        plugin_files = self.plugins.get(plugin_name)
        if not plugin_files:
            return {"allow": [], "deny": []}

        # Get patterns used by other plugins
        other_allow: set[str] = set()
        other_deny: set[str] = set()
        for name, files in self.plugins.items():
            if name != plugin_name:
                other_allow.update(files.claude_settings_allow)
                other_deny.update(files.claude_settings_deny)

        return {
            "allow": [p for p in plugin_files.claude_settings_allow if p not in other_allow],
            "deny": [p for p in plugin_files.claude_settings_deny if p not in other_deny],
        }

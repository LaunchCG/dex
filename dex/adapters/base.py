"""Abstract base class for platform adapters.

All platform-specific logic is encapsulated within adapters that implement
this interface. The core system delegates ALL platform decisions to adapters.
"""

import logging
from abc import ABC, abstractmethod
from pathlib import Path
from typing import Any

from dex.config.schemas import (
    AdapterMetadata,
    AgentFileConfig,
    ClaudeSettingsConfig,
    CommandConfig,
    FileTarget,
    InstallationPlan,
    InstructionConfig,
    MCPServerConfig,
    PluginManifest,
    PromptConfig,
    RuleConfig,
    SkillConfig,
    SubAgentConfig,
)
from dex.template.context_resolver import find_platform_specific_file

logger = logging.getLogger(__name__)


class PlatformAdapter(ABC):
    """Abstract base class for platform adapters.

    Platform adapters handle all platform-specific aspects of plugin installation:
    - Directory structure and file locations
    - Skill/command/subagent installation planning
    - MCP server configuration
    - Frontmatter generation
    - Template variable provisioning

    Subclasses must implement all abstract methods to support a new platform.
    """

    # =========================================================================
    # Metadata
    # =========================================================================

    @property
    @abstractmethod
    def metadata(self) -> AdapterMetadata:
        """Get adapter metadata.

        Returns:
            AdapterMetadata describing this adapter's capabilities
        """
        ...

    # =========================================================================
    # Directory Structure
    # =========================================================================

    @abstractmethod
    def get_base_directory(self, project_root: Path) -> Path:
        """Get the base directory for this platform's configuration.

        For example:
        - Claude Code: .claude
        - Cursor: .cursor

        Args:
            project_root: Path to the project root

        Returns:
            Path to the platform's base configuration directory
        """
        ...

    @abstractmethod
    def get_skills_directory(self, project_root: Path) -> Path:
        """Get the directory where skills are installed.

        Args:
            project_root: Path to the project root

        Returns:
            Path to the skills directory
        """
        ...

    def get_skill_install_directory(
        self,
        skill: SkillConfig,
        plugin: PluginManifest,
        project_root: Path,
    ) -> Path:
        """Get the directory where a specific skill will be installed.

        This method returns the actual directory path used for installation,
        which is used to compute the context_root template variable.

        Default implementation uses: skills_dir / skill.name
        Override if the platform uses a different naming convention.

        Args:
            skill: The skill configuration
            plugin: The plugin manifest
            project_root: Path to the project root

        Returns:
            Path to the skill's installation directory
        """
        return self.get_skills_directory(project_root) / skill.name

    @abstractmethod
    def get_mcp_config_path(self, project_root: Path) -> Path | None:
        """Get the path to the MCP configuration file.

        Args:
            project_root: Path to the project root

        Returns:
            Path to the MCP config file (e.g., .mcp.json for Claude Code), or None if
            the platform doesn't support MCP configuration.
        """
        ...

    def get_commands_directory(self, project_root: Path) -> Path:
        """Get the directory where commands are installed.

        Default implementation returns the same as skills directory.
        Override if the platform stores commands separately.

        Args:
            project_root: Path to the project root

        Returns:
            Path to the commands directory
        """
        return self.get_skills_directory(project_root)

    def get_subagents_directory(self, project_root: Path) -> Path:
        """Get the directory where sub-agents are installed.

        Default implementation returns the same as skills directory.
        Override if the platform stores sub-agents separately.

        Args:
            project_root: Path to the project root

        Returns:
            Path to the sub-agents directory
        """
        return self.get_skills_directory(project_root)

    def get_instructions_directory(self, project_root: Path) -> Path:
        """Get the directory where instructions are installed.

        Default implementation returns the base directory.
        Override if the platform supports instructions.

        Args:
            project_root: Path to the project root

        Returns:
            Path to the instructions directory
        """
        return self.get_base_directory(project_root)

    def get_rules_directory(self, project_root: Path) -> Path:
        """Get the directory where rules are installed.

        Default implementation returns the base directory.
        Override if the platform supports rules.

        Args:
            project_root: Path to the project root

        Returns:
            Path to the rules directory
        """
        return self.get_base_directory(project_root)

    def get_prompts_directory(self, project_root: Path) -> Path:
        """Get the directory where prompts are installed.

        Default implementation returns the base directory.
        Override if the platform supports prompts.

        Args:
            project_root: Path to the project root

        Returns:
            Path to the prompts directory
        """
        return self.get_base_directory(project_root)

    def get_files_directory(self, project_root: Path, plugin_name: str) -> Path:
        """Get the directory where manifest-level files are installed.

        Default implementation returns a plugin-specific subdirectory under files/.
        These are files declared at the manifest level, not associated with
        any specific component (skill, command, etc.).

        Args:
            project_root: Path to the project root
            plugin_name: Name of the plugin

        Returns:
            Path to the plugin's files directory
        """
        return self.get_base_directory(project_root) / "files" / plugin_name

    def get_agent_file_path(self, project_root: Path) -> Path | None:
        """Get the path to the main agent instruction file.

        This is the file where plugin content is injected using markers
        (e.g., CLAUDE.md for Claude Code, AGENTS.md for Codex).

        Default implementation returns None (no agent file support).
        Override if the platform supports agent file content injection.

        Args:
            project_root: Path to the project root

        Returns:
            Path to the agent file, or None if not supported
        """
        return None

    # =========================================================================
    # Installation Planning
    # =========================================================================

    def plan_skill_installation(
        self,
        skill: SkillConfig,
        plugin: PluginManifest,
        rendered_content: str,
        project_root: Path,
        source_dir: Path,
    ) -> InstallationPlan:
        """Plan the installation of a skill.

        Default implementation returns empty plan (not supported).
        Override in adapters that support skills.
        """
        logger.info(
            "%s does not support skills - skipping '%s'",
            self.metadata.display_name,
            skill.name,
        )
        return InstallationPlan(directories_to_create=[], files_to_write=[])

    def plan_command_installation(
        self,
        command: CommandConfig,
        plugin: PluginManifest,
        rendered_content: str,
        project_root: Path,
        source_dir: Path,
    ) -> InstallationPlan:
        """Plan the installation of a command.

        Default implementation returns empty plan (not supported).
        Override in adapters that support commands.
        """
        logger.info(
            "%s does not support commands - skipping '%s'",
            self.metadata.display_name,
            command.name,
        )
        return InstallationPlan(directories_to_create=[], files_to_write=[])

    def plan_subagent_installation(
        self,
        subagent: SubAgentConfig,
        plugin: PluginManifest,
        rendered_content: str,
        project_root: Path,
        source_dir: Path,
    ) -> InstallationPlan:
        """Plan the installation of a sub-agent.

        Default implementation returns empty plan (not supported).
        Override in adapters that support sub-agents.
        """
        logger.info(
            "%s does not support sub-agents - skipping '%s'",
            self.metadata.display_name,
            subagent.name,
        )
        return InstallationPlan(directories_to_create=[], files_to_write=[])

    def plan_instruction_installation(
        self,
        instruction: InstructionConfig,
        plugin: PluginManifest,
        rendered_content: str,
        project_root: Path,
        source_dir: Path,
    ) -> InstallationPlan:
        """Plan the installation of an instruction.

        Default implementation returns empty plan (not supported).
        Override in adapters that support instructions.
        """
        logger.info(
            "%s does not support instructions - skipping '%s'",
            self.metadata.display_name,
            instruction.name,
        )
        return InstallationPlan(directories_to_create=[], files_to_write=[])

    def plan_rule_installation(
        self,
        rule: RuleConfig,
        plugin: PluginManifest,
        rendered_content: str,
        project_root: Path,
        source_dir: Path,
    ) -> InstallationPlan:
        """Plan the installation of a rule.

        Default implementation returns empty plan (not supported).
        Override in adapters that support rules.
        """
        logger.info(
            "%s does not support rules - skipping '%s'",
            self.metadata.display_name,
            rule.name,
        )
        return InstallationPlan(directories_to_create=[], files_to_write=[])

    def plan_prompt_installation(
        self,
        prompt: PromptConfig,
        plugin: PluginManifest,
        rendered_content: str,
        project_root: Path,
        source_dir: Path,
    ) -> InstallationPlan:
        """Plan the installation of a prompt.

        Default implementation returns empty plan (not supported).
        Override in adapters that support prompts.
        """
        logger.info(
            "%s does not support prompts - skipping '%s'",
            self.metadata.display_name,
            prompt.name,
        )
        return InstallationPlan(directories_to_create=[], files_to_write=[])

    def plan_agent_file_installation(
        self,
        agent_file_config: AgentFileConfig,
        plugin: PluginManifest,
        rendered_content: str,
        project_root: Path,
        source_dir: Path,
    ) -> InstallationPlan:
        """Plan the installation of content into the agent file.

        Default implementation returns empty plan (not supported).
        Override in adapters that support agent file content injection.
        """
        logger.info(
            "%s does not support agent_file - skipping for plugin '%s'",
            self.metadata.display_name,
            plugin.name,
        )
        return InstallationPlan(directories_to_create=[], files_to_write=[])

    # =========================================================================
    # File Handling
    # =========================================================================

    def _add_files_to_plan(
        self,
        plan: InstallationPlan,
        files: list[FileTarget] | None,
        source_dir: Path,
        dest_dir: Path,
        render_as_template: bool = False,
    ) -> None:
        """Add file copy/render operations to an installation plan.

        Args:
            plan: The installation plan to update
            files: List of FileTarget objects (src required, dest defaults to basename)
            source_dir: Plugin root directory (src paths are relative to this)
            dest_dir: Destination directory for files
            render_as_template: If True, add to template_files_to_render instead of files_to_copy
        """
        if not files:
            return

        # Add to appropriate plan field
        target_dict = plan.template_files_to_render if render_as_template else plan.files_to_copy

        for file_target in files:
            src_path = file_target.src
            dest_path = file_target.dest  # Guaranteed non-None by validator

            # Strip leading ./ if present
            if src_path.startswith("./"):
                src_path = src_path[2:]
            if dest_path and dest_path.startswith("./"):
                dest_path = dest_path[2:]

            # Resolve platform-specific file override
            resolved_path = find_platform_specific_file(source_dir, src_path, self.metadata.name)
            src = source_dir / resolved_path
            # Use dest_path for destination (allows renaming)
            # dest_path is guaranteed non-None by FileTarget validator
            assert dest_path is not None
            dest = dest_dir / dest_path

            if src.exists():
                target_dict[src] = dest
                # Ensure parent directory is created
                if dest.parent not in plan.directories_to_create:
                    plan.directories_to_create.append(dest.parent)

    # =========================================================================
    # MCP Configuration
    # =========================================================================

    @abstractmethod
    def generate_mcp_config(
        self,
        mcp_server: MCPServerConfig,
        plugin: PluginManifest,
        project_root: Path,
        source_dir: Path,
    ) -> dict[str, Any]:
        """Generate MCP server configuration for this platform.

        Args:
            mcp_server: MCP server configuration from the plugin manifest
            plugin: The plugin manifest
            project_root: Path to the project root
            source_dir: Path to the plugin source directory

        Returns:
            Dictionary representing the MCP config entry for this server
        """
        ...

    @abstractmethod
    def merge_mcp_config(
        self,
        existing_config: dict[str, Any],
        new_entries: dict[str, Any],
    ) -> dict[str, Any]:
        """Merge new MCP entries into existing configuration.

        This handles platform-specific config structure.

        Args:
            existing_config: Current MCP configuration
            new_entries: New MCP server entries to add

        Returns:
            Merged configuration
        """
        ...

    # =========================================================================
    # Claude Settings Configuration
    # =========================================================================

    def get_claude_settings_path(self, project_root: Path) -> Path | None:
        """Get path to Claude settings file.

        Default implementation returns None (not supported).
        Override in adapters that support Claude settings.

        Args:
            project_root: Path to the project root

        Returns:
            Path to the settings file, or None if not supported
        """
        return None

    def generate_claude_settings_config(
        self,
        claude_settings: ClaudeSettingsConfig,
        plugin: PluginManifest,
        project_root: Path,
    ) -> dict[str, Any]:
        """Generate settings entries for this platform.

        Default implementation returns empty dict (not supported).
        Override in adapters that support Claude settings.

        Args:
            claude_settings: Claude settings configuration from the plugin manifest
            plugin: The plugin manifest
            project_root: Path to the project root

        Returns:
            Dictionary representing the settings config entries
        """
        return {}

    def merge_claude_settings_config(
        self,
        existing_config: dict[str, Any],
        new_entries: dict[str, Any],
    ) -> dict[str, Any]:
        """Merge new settings into existing configuration.

        Default implementation returns existing_config unchanged.
        Override in adapters that support Claude settings.

        Args:
            existing_config: Current settings configuration
            new_entries: New settings entries to add

        Returns:
            Merged configuration
        """
        return existing_config

    # =========================================================================
    # Frontmatter Generation
    # =========================================================================

    def generate_skill_frontmatter(
        self,
        skill: SkillConfig,
        plugin: PluginManifest,
    ) -> str:
        """Generate frontmatter/header for a skill file.

        Default implementation returns empty string (no frontmatter).
        Override in adapters that support skills with frontmatter.

        Args:
            skill: Skill configuration
            plugin: The plugin manifest

        Returns:
            Frontmatter string to prepend to skill content
        """
        return ""

    def generate_command_frontmatter(
        self,
        command: CommandConfig,
        plugin: PluginManifest,
    ) -> str:
        """Generate frontmatter/header for a command file.

        Default implementation uses skill frontmatter format.

        Args:
            command: Command configuration
            plugin: The plugin manifest

        Returns:
            Frontmatter string to prepend to command content
        """
        # Create a skill-like config for default behavior
        skill = SkillConfig(
            name=command.name,
            description=command.description,
            context=command.context,
            files=command.files,
            metadata=command.metadata,
        )
        return self.generate_skill_frontmatter(skill, plugin)

    def generate_subagent_frontmatter(
        self,
        subagent: SubAgentConfig,
        plugin: PluginManifest,
    ) -> str:
        """Generate frontmatter/header for a sub-agent file.

        Default implementation uses skill frontmatter format.

        Args:
            subagent: Sub-agent configuration
            plugin: The plugin manifest

        Returns:
            Frontmatter string to prepend to sub-agent content
        """
        # Create a skill-like config for default behavior
        skill = SkillConfig(
            name=subagent.name,
            description=subagent.description,
            context=subagent.context,
            files=subagent.files,
            metadata=subagent.metadata,
        )
        return self.generate_skill_frontmatter(skill, plugin)

    def generate_instruction_frontmatter(
        self,
        instruction: InstructionConfig,
        plugin: PluginManifest,
    ) -> str:
        """Generate frontmatter/header for an instruction file.

        Default implementation returns empty string.
        Override in adapters that support instructions.

        Args:
            instruction: Instruction configuration
            plugin: The plugin manifest

        Returns:
            Frontmatter string to prepend to instruction content
        """
        return ""

    def generate_rule_frontmatter(
        self,
        rule: RuleConfig,
        plugin: PluginManifest,
    ) -> str:
        """Generate frontmatter/header for a rule file.

        Default implementation returns empty string.
        Override in adapters that support rules.

        Args:
            rule: Rule configuration
            plugin: The plugin manifest

        Returns:
            Frontmatter string to prepend to rule content
        """
        return ""

    def generate_prompt_frontmatter(
        self,
        prompt: PromptConfig,
        plugin: PluginManifest,
    ) -> str:
        """Generate frontmatter/header for a prompt file.

        Default implementation returns empty string.
        Override in adapters that support prompts.

        Args:
            prompt: Prompt configuration
            plugin: The plugin manifest

        Returns:
            Frontmatter string to prepend to prompt content
        """
        return ""

    # =========================================================================
    # Template Variables
    # =========================================================================

    def get_template_variables(
        self,
        project_root: Path,
        plugin: PluginManifest | None = None,
    ) -> dict[str, Any]:
        """Get template variables specific to this platform.

        Override only if the adapter needs to provide platform-specific
        variables beyond what the context builder provides.

        Returns:
            Empty dict by default
        """
        return {}

    # =========================================================================
    # Validation
    # =========================================================================

    @abstractmethod
    def validate_plugin_compatibility(
        self,
        plugin: PluginManifest,
    ) -> list[str]:
        """Validate that a plugin is compatible with this platform.

        Args:
            plugin: Plugin manifest to validate

        Returns:
            List of warning messages (empty if fully compatible)
        """
        ...

    # =========================================================================
    # Lifecycle Hooks
    # =========================================================================

    def pre_install(  # noqa: B027
        self,
        project_root: Path,
        plugins: list[PluginManifest],
    ) -> None:
        """Hook called before any plugins are installed.

        Override to perform platform-specific setup.

        Args:
            project_root: Path to the project root
            plugins: List of plugins about to be installed
        """

    def post_install(  # noqa: B027
        self,
        project_root: Path,
        installed: list[PluginManifest],
    ) -> None:
        """Hook called after all plugins are installed.

        Override to perform platform-specific cleanup or finalization.

        Args:
            project_root: Path to the project root
            installed: List of successfully installed plugins
        """

    def pre_uninstall(  # noqa: B027
        self,
        project_root: Path,
        plugins: list[str],
    ) -> None:
        """Hook called before plugins are uninstalled.

        Args:
            project_root: Path to the project root
            plugins: List of plugin names to be uninstalled
        """

    def post_uninstall(  # noqa: B027
        self,
        project_root: Path,
        removed: list[str],
    ) -> None:
        """Hook called after plugins are uninstalled.

        Args:
            project_root: Path to the project root
            removed: List of successfully removed plugin names
        """

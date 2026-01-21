"""Cursor IDE platform adapter.

Cursor uses:
- .cursor/rules/ directory with MDC files for rules (YAML frontmatter with
  description, globs, and alwaysApply fields)
- .cursor/commands/ directory with plain Markdown files for commands
  (no frontmatter required)
"""

from pathlib import Path
from typing import Any

from dex.adapters import register_adapter
from dex.adapters.base import PlatformAdapter
from dex.config.schemas import (
    AdapterMetadata,
    CommandConfig,
    FileToWrite,
    InstallationPlan,
    MCPServerConfig,
    PluginManifest,
    RuleConfig,
)


@register_adapter("cursor")
class CursorAdapter(PlatformAdapter):
    """Adapter for Cursor IDE.

    Cursor supports rules, commands, and MCP servers.

    Directory structure:
    .cursor/
    ├── rules/
    │   └── {plugin}-{rule}.mdc     # Rules as MDC files
    └── commands/
        └── {plugin}-{command}.md   # Commands as plain Markdown

    MDC frontmatter fields (rules only):
    - description: When the rule should apply (for intelligent selection)
    - globs: File patterns to auto-attach (e.g., "**/*.ts")
    - alwaysApply: Boolean, if true applies to every chat session

    Commands are plain Markdown files without frontmatter, triggered with / prefix.
    """

    @property
    def metadata(self) -> AdapterMetadata:
        return AdapterMetadata(
            name="cursor",
            display_name="Cursor",
            description="Cursor IDE with AI assistance",
            mcp_config_file=".cursor/mcp.json",
        )

    # =========================================================================
    # Directory Structure
    # =========================================================================

    def get_base_directory(self, project_root: Path) -> Path:
        return project_root / ".cursor"

    def get_skills_directory(self, project_root: Path) -> Path:
        """Cursor does not support skills. Returns rules directory as fallback."""
        return self.get_base_directory(project_root) / "rules"

    def get_rules_directory(self, project_root: Path) -> Path:
        """Rules are stored in .cursor/rules/."""
        return self.get_base_directory(project_root) / "rules"

    def get_commands_directory(self, project_root: Path) -> Path:
        """Commands are stored in .cursor/commands/."""
        return self.get_base_directory(project_root) / "commands"

    def get_mcp_config_path(self, project_root: Path) -> Path:
        """MCP config at .cursor/mcp.json."""
        return self.get_base_directory(project_root) / "mcp.json"

    # =========================================================================
    # Installation Planning
    # =========================================================================

    def plan_rule_installation(
        self,
        rule: RuleConfig,
        plugin: PluginManifest,
        rendered_content: str,
        project_root: Path,
        source_dir: Path,
    ) -> InstallationPlan:
        """Plan rule installation for Cursor.

        Creates:
        .cursor/rules/{plugin}-{rule}.mdc
        """
        rules_dir = self.get_rules_directory(project_root)

        # Cursor uses flat MDC files with plugin-rule naming
        rule_file = rules_dir / f"{plugin.name}-{rule.name}.mdc"

        # Generate frontmatter
        frontmatter = self.generate_rule_frontmatter(rule, plugin)
        full_content = frontmatter + rendered_content

        plan = InstallationPlan(
            directories_to_create=[rules_dir],
            files_to_write=[FileToWrite(path=rule_file, content=full_content)],
        )

        # Add associated files to rules directory
        self._add_files_to_plan(plan, rule.files, source_dir, rules_dir)
        self._add_files_to_plan(
            plan, rule.template_files, source_dir, rules_dir, render_as_template=True
        )

        return plan

    def plan_command_installation(
        self,
        command: CommandConfig,
        plugin: PluginManifest,
        rendered_content: str,
        project_root: Path,
        source_dir: Path,
    ) -> InstallationPlan:
        """Plan command installation for Cursor.

        Creates: .cursor/commands/{plugin}-{command}.md
        """
        commands_dir = self.get_commands_directory(project_root)
        command_file = commands_dir / f"{plugin.name}-{command.name}.md"

        frontmatter = self.generate_command_frontmatter(command, plugin)
        full_content = frontmatter + rendered_content

        plan = InstallationPlan(
            directories_to_create=[commands_dir],
            files_to_write=[FileToWrite(path=command_file, content=full_content)],
        )

        self._add_files_to_plan(plan, command.files, source_dir, commands_dir)
        self._add_files_to_plan(
            plan, command.template_files, source_dir, commands_dir, render_as_template=True
        )
        return plan

    # Note: _add_files_to_plan is inherited from PlatformAdapter base class

    # =========================================================================
    # MCP Configuration
    # =========================================================================

    def generate_mcp_config(
        self,
        mcp_server: MCPServerConfig,
        plugin: PluginManifest,
        project_root: Path,
        source_dir: Path,
    ) -> dict[str, Any]:
        """Generate MCP config entry for Cursor.

        Cursor uses .cursor/mcp.json with mcpServers object.
        """
        config: dict[str, Any] = {}

        if mcp_server.type == "command":
            # Command-based server (stdio transport)
            if mcp_server.source:
                # Expand source shortcut to command/args
                source = mcp_server.source
                if source.startswith("npm:"):
                    package_name = source[4:]
                    config["command"] = "npx"
                    config["args"] = ["-y", package_name]
                elif source.startswith("uvx:"):
                    package_source = source[4:]
                    config["command"] = "uvx"
                    config["args"] = ["--from", package_source]
                elif source.startswith("pip:"):
                    package_name = source[4:]
                    config["command"] = package_name
                    config["args"] = []
            else:
                # Direct command/args
                config["command"] = mcp_server.command
                config["args"] = list(mcp_server.args) if mcp_server.args else []

            # Add extra args from config
            if mcp_server.args and mcp_server.source:
                config["args"] = config.get("args", []) + list(mcp_server.args)

            # Add env if provided
            if mcp_server.env:
                config["env"] = dict(mcp_server.env)

        elif mcp_server.type == "http":
            # HTTP-based server - Cursor uses just url field
            config["url"] = mcp_server.url

        return {mcp_server.name: config}

    def merge_mcp_config(
        self,
        existing_config: dict[str, Any],
        new_entries: dict[str, Any],
    ) -> dict[str, Any]:
        """Merge MCP entries into Cursor .cursor/mcp.json format."""
        if "mcpServers" not in existing_config:
            existing_config["mcpServers"] = {}

        existing_config["mcpServers"].update(new_entries)

        return existing_config

    # =========================================================================
    # Frontmatter Generation
    # =========================================================================

    def generate_rule_frontmatter(
        self,
        rule: RuleConfig,
        plugin: PluginManifest,
    ) -> str:
        """Generate Cursor MDC frontmatter for rules.

        Cursor rules use YAML frontmatter with:
        - description: When this rule applies (for intelligent selection)
        - globs: File patterns to auto-attach
        - alwaysApply: Whether to apply to every session
        """
        lines = [
            "---",
            f"description: {rule.description}",
            f"globs: {rule.glob}" if rule.glob else "globs:",
            f"alwaysApply: {'true' if rule.always else 'false'}",
            "---",
            "",  # Blank line after frontmatter
        ]

        return "\n".join(lines)

    def generate_command_frontmatter(
        self,
        command: CommandConfig,
        plugin: PluginManifest,
    ) -> str:
        """Cursor commands do not require frontmatter."""
        return ""

    # =========================================================================
    # Validation
    # =========================================================================

    def validate_plugin_compatibility(
        self,
        plugin: PluginManifest,
    ) -> list[str]:
        """Validate plugin compatibility with Cursor.

        Cursor only supports rules and MCP servers.
        """
        return []

    # =========================================================================
    # Lifecycle Hooks
    # =========================================================================

    def pre_install(
        self,
        project_root: Path,
        plugins: list[PluginManifest],
    ) -> None:
        """Ensure .cursor directories exist before installation."""
        base_dir = self.get_base_directory(project_root)
        base_dir.mkdir(parents=True, exist_ok=True)
        self.get_rules_directory(project_root).mkdir(parents=True, exist_ok=True)
        self.get_commands_directory(project_root).mkdir(parents=True, exist_ok=True)

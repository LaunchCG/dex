"""Cursor IDE platform adapter.

Cursor uses .cursor/rules/ directory with MDC files for rules/skills.
MDC files have YAML frontmatter with description, globs, and alwaysApply fields.
"""

from pathlib import Path
from typing import Any

from dex.adapters import register_adapter
from dex.adapters.base import PlatformAdapter
from dex.config.schemas import (
    AdapterMetadata,
    FileToWrite,
    InstallationPlan,
    MCPServerConfig,
    PlatformFiles,
    PluginManifest,
    RuleConfig,
)
from dex.template.context_resolver import find_platform_specific_file
from dex.utils.platform import get_os, is_unix


@register_adapter("cursor")
class CursorAdapter(PlatformAdapter):
    """Adapter for Cursor IDE.

    Cursor supports rules and MCP servers only.

    Directory structure:
    .cursor/
    └── rules/
        └── {plugin}-{rule}.mdc     # Rules as MDC files

    MDC frontmatter fields:
    - description: When the rule should apply (for intelligent selection)
    - globs: File patterns to auto-attach (e.g., "**/*.ts")
    - alwaysApply: Boolean, if true applies to every chat session
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

        return plan

    def _add_files_to_plan(
        self,
        plan: InstallationPlan,
        files: Any,
        source_dir: Path,
        dest_dir: Path,
    ) -> None:
        """Add file copy operations to an installation plan."""
        if files is None:
            return

        current_os = get_os()
        files_to_copy: list[str] = []

        if isinstance(files, list):
            files_to_copy = [str(f) for f in files]
        elif isinstance(files, PlatformFiles):
            files_to_copy.extend(files.common)
            platform_files = files.platform
            if current_os in platform_files:
                files_to_copy.extend(platform_files[current_os])
            if is_unix() and "unix" in platform_files:
                files_to_copy.extend(platform_files["unix"])
        elif isinstance(files, dict):
            if "common" in files or "platform" in files:
                files_to_copy.extend(files.get("common", []))
                platform_files = files.get("platform", {})
                if current_os in platform_files:
                    files_to_copy.extend(platform_files[current_os])
                if is_unix() and "unix" in platform_files:
                    files_to_copy.extend(platform_files["unix"])
            else:
                files_to_copy = list(files.keys()) if files else []

        for file_path in files_to_copy:
            if file_path.startswith("./"):
                file_path = file_path[2:]
            # Resolve platform-specific file override
            resolved_path = find_platform_specific_file(source_dir, file_path, self.metadata.name)
            src = source_dir / resolved_path
            # Preserve directory structure within dest (use original path for dest)
            dest = dest_dir / file_path
            if src.exists():
                plan.files_to_copy[src] = dest
                if dest.parent not in plan.directories_to_create:
                    plan.directories_to_create.append(dest.parent)

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

        if mcp_server.type == "bundled":
            path = mcp_server.path
            if isinstance(path, dict):
                current_os = get_os()
                if current_os in path:
                    path = path[current_os]
                elif is_unix() and "unix" in path:
                    path = path["unix"]
                else:
                    path = next(iter(path.values()))

            if path and path.startswith("./"):
                path = path[2:]

            server_path = source_dir / path if path else None

            if server_path and server_path.suffix == ".js":
                config["command"] = "node"
                config["args"] = [str(server_path)]
            elif server_path and server_path.suffix == ".py":
                config["command"] = "python"
                config["args"] = [str(server_path)]
            elif server_path:
                config["command"] = str(server_path)
                config["args"] = []

        elif mcp_server.type == "remote":
            source = mcp_server.source
            if source:
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

        if mcp_server.config:
            if "args" in mcp_server.config:
                existing_args = config.get("args", [])
                config["args"] = existing_args + mcp_server.config["args"]
            if "env" in mcp_server.config:
                config["env"] = mcp_server.config["env"]
            if "command" in mcp_server.config:
                config["command"] = mcp_server.config["command"]

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

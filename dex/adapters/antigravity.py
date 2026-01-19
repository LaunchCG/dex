"""Google Antigravity platform adapter.

Antigravity uses:
- .agent/skills/ directory with SKILL.md files for skills
- .agent/mcp.json for MCP server configuration

This follows the open Agent Skills standard shared with Claude Code and Codex.
"""

from pathlib import Path
from typing import Any

from dex.adapters import register_adapter
from dex.adapters.base import PlatformAdapter
from dex.config.schemas import (  # First-class types; Core types
    AdapterMetadata,
    CommandConfig,
    FileToWrite,
    InstallationPlan,
    MCPServerConfig,
    PlatformFiles,
    PluginManifest,
    RuleConfig,
    SkillConfig,
    SubAgentConfig,
)
from dex.template.context_resolver import find_platform_specific_file
from dex.utils.platform import get_os, is_unix


@register_adapter("antigravity")
class AntigravityAdapter(PlatformAdapter):
    """Adapter for Google Antigravity.

    Directory structure:
    .agent/
    └── skills/
        └── {skill-name}/
            ├── SKILL.md
            ├── scripts/      (optional)
            ├── references/   (optional)
            └── assets/       (optional)

    Antigravity doesn't support:
    - Commands (use skills instead)
    - Subagents

    SKILL.md frontmatter:
    - name: Skill identifier
    - description: When to use this skill
    """

    @property
    def metadata(self) -> AdapterMetadata:
        return AdapterMetadata(
            name="antigravity",
            display_name="Google Antigravity",
            description="Google's agentic development platform",
            mcp_config_file=None,  # Antigravity manages MCP through UI only
        )

    # =========================================================================
    # Directory Structure
    # =========================================================================

    def get_base_directory(self, project_root: Path) -> Path:
        """Universal .agent directory for cross-platform compatibility."""
        return project_root / ".agent"

    def get_skills_directory(self, project_root: Path) -> Path:
        return self.get_base_directory(project_root) / "skills"

    def get_commands_directory(self, project_root: Path) -> Path:
        """Antigravity doesn't have commands - returns skills directory."""
        return self.get_skills_directory(project_root)

    def get_subagents_directory(self, project_root: Path) -> Path:
        """Antigravity doesn't have subagents."""
        return self.get_base_directory(project_root)

    def get_rules_directory(self, project_root: Path) -> Path:
        """Rules directory at .agent/rules/."""
        return self.get_base_directory(project_root) / "rules"

    def get_mcp_config_path(self, project_root: Path) -> Path | None:
        """Antigravity manages MCP through UI only - no project-level config file."""
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
        """Plan skill installation for Antigravity.

        Creates:
        .agent/skills/{skill-name}/SKILL.md
        .agent/skills/{skill-name}/[scripts/references/assets...]
        """
        skill_dir = self.get_skills_directory(project_root) / skill.name

        # Generate frontmatter
        frontmatter = self.generate_skill_frontmatter(skill, plugin)
        full_content = frontmatter + rendered_content

        plan = InstallationPlan(
            directories_to_create=[skill_dir],
            files_to_write=[FileToWrite(path=skill_dir / "SKILL.md", content=full_content)],
        )

        # Add associated files
        self._add_files_to_plan(plan, skill.files, source_dir, skill_dir)

        return plan

    def plan_command_installation(
        self,
        command: CommandConfig,
        plugin: PluginManifest,
        rendered_content: str,
        project_root: Path,
        source_dir: Path,
    ) -> InstallationPlan:
        """Antigravity doesn't support commands - returns empty plan."""
        return InstallationPlan(directories_to_create=[], files_to_write=[])

    def plan_subagent_installation(
        self,
        subagent: SubAgentConfig,
        plugin: PluginManifest,
        rendered_content: str,
        project_root: Path,
        source_dir: Path,
    ) -> InstallationPlan:
        """Antigravity doesn't support subagents - returns empty plan."""
        return InstallationPlan(directories_to_create=[], files_to_write=[])

    def plan_rule_installation(
        self,
        rule: RuleConfig,
        plugin: PluginManifest,
        rendered_content: str,
        project_root: Path,
        source_dir: Path,
    ) -> InstallationPlan:
        """Plan rule installation for Antigravity.

        Creates:
        .agent/rules/{rule-name}.md
        """
        rules_dir = self.get_rules_directory(project_root)
        rule_file = rules_dir / f"{rule.name}.md"

        # Generate frontmatter
        frontmatter = self.generate_rule_frontmatter(rule, plugin)
        full_content = frontmatter + rendered_content

        plan = InstallationPlan(
            directories_to_create=[rules_dir],
            files_to_write=[FileToWrite(path=rule_file, content=full_content)],
        )

        # Add associated files
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
    # MCP Configuration (Not Supported - Antigravity uses UI-managed config)
    # =========================================================================

    def generate_mcp_config(
        self,
        mcp_server: MCPServerConfig,
        plugin: PluginManifest,
        project_root: Path,
        source_dir: Path,
    ) -> dict[str, Any]:
        """Antigravity manages MCP through UI only - not project-level files."""
        return {}

    def merge_mcp_config(
        self,
        existing_config: dict[str, Any],
        new_entries: dict[str, Any],
    ) -> dict[str, Any]:
        """Antigravity manages MCP through UI only - not project-level files."""
        return existing_config

    # =========================================================================
    # Frontmatter Generation
    # =========================================================================

    def generate_skill_frontmatter(
        self,
        skill: SkillConfig,
        plugin: PluginManifest,
    ) -> str:
        """Generate Antigravity SKILL.md frontmatter.

        Antigravity skills use YAML frontmatter with:
        - name: Skill identifier
        - description: When to use this skill
        """
        lines = [
            "---",
            f"name: {skill.name}",
            f"description: {skill.description}",
        ]

        # Add any additional metadata
        if skill.metadata:
            for key, value in skill.metadata.items():
                lines.append(f"{key}: {value}")

        lines.append("---")
        lines.append("")  # Blank line after frontmatter

        return "\n".join(lines)

    def generate_command_frontmatter(
        self,
        command: CommandConfig,
        plugin: PluginManifest,
    ) -> str:
        """Antigravity doesn't support commands."""
        return ""

    def generate_subagent_frontmatter(
        self,
        subagent: SubAgentConfig,
        plugin: PluginManifest,
    ) -> str:
        """Antigravity doesn't support subagents."""
        return ""

    def generate_rule_frontmatter(
        self,
        rule: RuleConfig,
        plugin: PluginManifest,
    ) -> str:
        """Generate Antigravity rule frontmatter.

        Antigravity uses plain markdown rules without YAML frontmatter.
        The content is written directly without any frontmatter.
        """
        return ""

    # =========================================================================
    # Validation
    # =========================================================================

    def validate_plugin_compatibility(
        self,
        plugin: PluginManifest,
    ) -> list[str]:
        """Validate plugin compatibility with Antigravity."""
        warnings: list[str] = []

        if plugin.mcp_servers:
            warnings.append(
                f"Plugin '{plugin.name}' has MCP servers - Antigravity manages MCP "
                "through its UI only, not project-level config files"
            )

        if plugin.commands:
            warnings.append(
                f"Plugin '{plugin.name}' has commands which are not supported by "
                "Antigravity (consider converting to skills)"
            )

        if plugin.sub_agents:
            warnings.append(
                f"Plugin '{plugin.name}' has subagents which are not supported by " "Antigravity"
            )

        return warnings

    # =========================================================================
    # Lifecycle Hooks
    # =========================================================================

    def pre_install(
        self,
        project_root: Path,
        plugins: list[PluginManifest],
    ) -> None:
        """Ensure .agent directories exist before installation."""
        base_dir = self.get_base_directory(project_root)
        base_dir.mkdir(parents=True, exist_ok=True)
        self.get_skills_directory(project_root).mkdir(parents=True, exist_ok=True)
        self.get_rules_directory(project_root).mkdir(parents=True, exist_ok=True)

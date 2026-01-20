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
    PluginManifest,
    RuleConfig,
    SkillConfig,
    SubAgentConfig,
)


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
        self._add_files_to_plan(
            plan, skill.template_files, source_dir, skill_dir, render_as_template=True
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
        self._add_files_to_plan(
            plan, rule.template_files, source_dir, rules_dir, render_as_template=True
        )

        return plan

    # Note: _add_files_to_plan is inherited from PlatformAdapter base class

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

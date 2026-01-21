"""OpenAI Codex platform adapter.

Codex uses:
- .codex/skills/ directory with SKILL.md files for skills
- AGENTS.md files for project-wide instructions
- ~/.codex/config.toml for MCP server configuration (TOML format, global)
"""

from pathlib import Path
from typing import Any

from dex.adapters import register_adapter
from dex.adapters.base import PlatformAdapter
from dex.config.schemas import (  # First-class types this adapter supports; Core types this adapter supports
    AdapterMetadata,
    AgentFileConfig,
    CommandConfig,
    FileToWrite,
    InstallationPlan,
    MCPServerConfig,
    PluginManifest,
    RuleConfig,
    SkillConfig,
    SubAgentConfig,
)
from dex.utils.markers import insert_plugin_section
from dex.utils.platform import get_home_directory


@register_adapter("codex")
class CodexAdapter(PlatformAdapter):
    """Adapter for OpenAI Codex CLI.

    Directory structure:
    .codex/
    └── skills/
        └── {skill-name}/
            ├── SKILL.md
            ├── scripts/      (optional)
            ├── references/   (optional)
            └── assets/       (optional)
    AGENTS.md                 # Project-wide rules (RuleConfig)
    ~/.codex/config.toml      # Global MCP config (TOML format)

    SKILL.md frontmatter:
    - name: Skill identifier
    - description: When to use this skill
    - allowed-tools: List of tools the skill can use
    - license: License information (e.g., "MIT")
    - compatibility: Version compatibility info
    - metadata.short-description: Optional UI description

    MCP config format (TOML):
    [mcp_servers.<name>]
    command = "npx"
    args = ["-y", "@example/mcp-server"]
    env = { KEY = "value" }
    """

    @property
    def metadata(self) -> AdapterMetadata:
        return AdapterMetadata(
            name="codex",
            display_name="OpenAI Codex",
            description="OpenAI's Codex CLI coding agent",
            mcp_config_file="~/.codex/config.toml",  # Global TOML config
        )

    # =========================================================================
    # Directory Structure
    # =========================================================================

    def get_base_directory(self, project_root: Path) -> Path:
        return project_root / ".codex"

    def get_skills_directory(self, project_root: Path) -> Path:
        return self.get_base_directory(project_root) / "skills"

    def get_rules_directory(self, project_root: Path) -> Path:
        """Rules go to .codex/rules/ directory (not AGENTS.md which is user-maintained)."""
        return self.get_base_directory(project_root) / "rules"

    def get_mcp_config_path(self, project_root: Path) -> Path:
        """Codex uses global ~/.codex/config.toml for MCP configuration."""
        return Path(get_home_directory()) / ".codex" / "config.toml"

    def get_agent_file_path(self, project_root: Path) -> Path:
        """Get the path to AGENTS.md for content injection."""
        return project_root / "AGENTS.md"

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
        """Plan skill installation for Codex.

        Creates:
        .codex/skills/{skill-name}/SKILL.md
        .codex/skills/{skill-name}/[scripts/references/assets...]
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

    def plan_rule_installation(
        self,
        rule: RuleConfig,
        plugin: PluginManifest,
        rendered_content: str,
        project_root: Path,
        source_dir: Path,
    ) -> InstallationPlan:
        """Plan rule installation for Codex.

        Rules go to .codex/rules/{rule-name}.md (not AGENTS.md).
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

    def plan_agent_file_installation(
        self,
        agent_file_config: AgentFileConfig,
        plugin: PluginManifest,
        rendered_content: str,
        project_root: Path,
        source_dir: Path,
    ) -> InstallationPlan:
        """Plan agent file content installation for Codex.

        Injects plugin content into AGENTS.md using marker-based management.
        Each plugin's content is wrapped in markers:
        <!-- dex:plugin:{plugin-name}:start -->
        ... content ...
        <!-- dex:plugin:{plugin-name}:end -->

        This allows multiple plugins to contribute to AGENTS.md without
        conflicting with each other or user content.
        """
        agent_file_path = self.get_agent_file_path(project_root)

        # Read existing content if file exists
        existing_content = ""
        if agent_file_path.exists():
            existing_content = agent_file_path.read_text(encoding="utf-8")

        # Insert or update the plugin's section using markers
        new_content = insert_plugin_section(
            existing_content,
            plugin.name,
            rendered_content,
        )

        plan = InstallationPlan(
            directories_to_create=[],
            files_to_write=[FileToWrite(path=agent_file_path, content=new_content)],
        )

        # Add associated files if specified
        if agent_file_config.files:
            self._add_files_to_plan(
                plan,
                agent_file_config.files,
                source_dir,
                project_root,  # Agent file associated files go to project root
            )
        if agent_file_config.template_files:
            self._add_files_to_plan(
                plan,
                agent_file_config.template_files,
                source_dir,
                project_root,
                render_as_template=True,
            )

        return plan

    # Note: _add_files_to_plan is inherited from PlatformAdapter base class

    # =========================================================================
    # MCP Configuration (Global TOML at ~/.codex/config.toml)
    # =========================================================================

    def generate_mcp_config(
        self,
        mcp_server: MCPServerConfig,
        plugin: PluginManifest,
        project_root: Path,
        source_dir: Path,
    ) -> dict[str, Any]:
        """Generate MCP config entry for Codex.

        Codex uses TOML format with [mcp_servers.<name>] tables:
        [mcp_servers.example]
        command = "npx"
        args = ["-y", "@example/mcp-server"]
        env = { KEY = "value" }

        For HTTP servers:
        [mcp_servers.example]
        url = "https://example.com/mcp"
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
            # HTTP-based server - Codex uses just url field
            config["url"] = mcp_server.url

        return {mcp_server.name: config}

    def merge_mcp_config(
        self,
        existing_config: dict[str, Any],
        new_entries: dict[str, Any],
    ) -> dict[str, Any]:
        """Merge MCP entries into Codex config.toml format.

        Codex uses [mcp_servers.<name>] structure in TOML.
        """
        if "mcp_servers" not in existing_config:
            existing_config["mcp_servers"] = {}

        existing_config["mcp_servers"].update(new_entries)

        return existing_config

    # =========================================================================
    # Frontmatter Generation
    # =========================================================================

    def generate_skill_frontmatter(
        self,
        skill: SkillConfig,
        plugin: PluginManifest,
    ) -> str:
        """Generate Codex SKILL.md frontmatter.

        Codex skills use YAML frontmatter with:
        - name: Skill identifier
        - description: When to use this skill
        - allowed-tools: List of tools the skill can use
        - license: License information
        - compatibility: Version compatibility info
        - metadata.short-description: Optional UI description
        """
        lines = [
            "---",
            f"name: {skill.name}",
            f"description: {skill.description}",
        ]

        # Add allowed-tools if specified
        allowed_tools = skill.metadata.get("allowed-tools")
        if allowed_tools:
            if isinstance(allowed_tools, list):
                lines.append("allowed-tools:")
                for tool in allowed_tools:
                    lines.append(f"  - {tool}")
            else:
                lines.append(f"allowed-tools: {allowed_tools}")

        # Add license if specified
        license_info = skill.metadata.get("license")
        if license_info:
            lines.append(f"license: {license_info}")

        # Add compatibility if specified
        compatibility = skill.metadata.get("compatibility")
        if compatibility:
            lines.append(f"compatibility: {compatibility}")

        # Check for short-description in metadata
        short_desc = skill.metadata.get("short-description")
        if short_desc:
            lines.append("metadata:")
            lines.append(f"  short-description: {short_desc}")

        lines.append("---")
        lines.append("")  # Blank line after frontmatter

        return "\n".join(lines)

    def generate_command_frontmatter(
        self,
        command: CommandConfig,
        plugin: PluginManifest,
    ) -> str:
        """Codex doesn't support commands."""
        return ""

    def generate_subagent_frontmatter(
        self,
        subagent: SubAgentConfig,
        plugin: PluginManifest,
    ) -> str:
        """Codex doesn't support subagents."""
        return ""

    def generate_rule_frontmatter(
        self,
        rule: RuleConfig,
        plugin: PluginManifest,
    ) -> str:
        """Generate Codex rule frontmatter.

        Codex uses plain AGENTS.md files without YAML frontmatter.
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
        """Validate plugin compatibility with Codex."""
        warnings: list[str] = []

        # MCP servers are supported via global ~/.codex/config.toml (no warning needed)

        if plugin.commands:
            warnings.append(
                f"Plugin '{plugin.name}' has commands which are not supported by Codex "
                "(consider using skills or AGENTS.md rules)"
            )

        if plugin.sub_agents:
            warnings.append(
                f"Plugin '{plugin.name}' has subagents which are not supported by Codex"
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
        """Ensure .codex directories exist before installation."""
        base_dir = self.get_base_directory(project_root)
        base_dir.mkdir(parents=True, exist_ok=True)
        self.get_skills_directory(project_root).mkdir(parents=True, exist_ok=True)
        self.get_rules_directory(project_root).mkdir(parents=True, exist_ok=True)

"""Claude Code platform adapter.

This is the reference implementation for platform adapters.
It handles Claude Code's specific directory structure and configuration format.
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
from dex.utils.platform import get_os, is_unix


@register_adapter("claude-code")
class ClaudeCodeAdapter(PlatformAdapter):
    """Adapter for Claude Code.

    Directory structure (per official Claude Code documentation):
    .mcp.json                     # MCP server configuration (project root)
    CLAUDE.md                     # Project-wide rules (RuleConfig)
    .claude/
    ├── settings.json             # Permissions (allow/deny)
    ├── {plugin-name}.local.md    # Plugin-specific rules (optional)
    ├── commands/
    │   └── {command-name}.md     # Slash commands (flat files)
    ├── agents/
    │   └── {agent-name}.md       # Sub-agents (flat files)
    └── skills/
        └── {plugin-name}/
            └── {skill-name}/
                ├── SKILL.md
                └── [supporting files]
    """

    @property
    def metadata(self) -> AdapterMetadata:
        return AdapterMetadata(
            name="claude-code",
            display_name="Claude Code",
            description="Anthropic's Claude Code CLI agent",
            mcp_config_file=".mcp.json",
        )

    # =========================================================================
    # Directory Structure
    # =========================================================================

    def get_base_directory(self, project_root: Path) -> Path:
        return project_root / ".claude"

    def get_skills_directory(self, project_root: Path) -> Path:
        return self.get_base_directory(project_root) / "skills"

    def get_skill_install_directory(
        self,
        skill: SkillConfig,
        plugin: PluginManifest,
        project_root: Path,
    ) -> Path:
        """Get the skill install directory using Claude Code's naming convention.

        Claude Code uses: .claude/skills/{plugin-name}-{skill-name}/
        """
        return self.get_skills_directory(project_root) / f"{plugin.name}-{skill.name}"

    def get_commands_directory(self, project_root: Path) -> Path:
        """Commands are flat .md files in .claude/commands/."""
        return self.get_base_directory(project_root) / "commands"

    def get_subagents_directory(self, project_root: Path) -> Path:
        """Agents are flat .md files in .claude/agents/."""
        return self.get_base_directory(project_root) / "agents"

    def get_rules_directory(self, project_root: Path) -> Path:
        """Rules go to .claude/rules/ directory."""
        return self.get_base_directory(project_root) / "rules"

    def get_mcp_config_path(self, project_root: Path) -> Path:
        return project_root / ".mcp.json"

    def get_agent_file_path(self, project_root: Path) -> Path:
        """Get the path to CLAUDE.md for content injection."""
        return project_root / "CLAUDE.md"

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
        """Plan skill installation for Claude Code.

        Creates:
        .claude/skills/{plugin-name}-{skill-name}/SKILL.md
        .claude/skills/{plugin-name}-{skill-name}/[files...]
        """
        skill_dir = self.get_skills_directory(project_root) / f"{plugin.name}-{skill.name}"

        # Generate frontmatter
        frontmatter = self.generate_skill_frontmatter(skill, plugin)
        full_content = frontmatter + rendered_content

        plan = InstallationPlan(
            directories_to_create=[skill_dir],
            files_to_write=[FileToWrite(path=skill_dir / "SKILL.md", content=full_content)],
        )

        # Add associated files
        self._add_files_to_plan(plan, skill.files, source_dir, skill_dir)
        self._add_files_to_plan(plan, skill.template_files, source_dir, skill_dir, render_as_template=True)

        return plan

    def plan_command_installation(
        self,
        command: CommandConfig,
        plugin: PluginManifest,
        rendered_content: str,
        project_root: Path,
        source_dir: Path,
    ) -> InstallationPlan:
        """Plan command installation for Claude Code.

        Per Claude Code docs, commands are flat .md files:
        .claude/commands/{plugin}-{command}.md

        The command name becomes a slash command: /{plugin}-{command}
        """
        commands_dir = self.get_commands_directory(project_root)
        command_file = commands_dir / f"{plugin.name}-{command.name}.md"

        frontmatter = self.generate_command_frontmatter(command, plugin)
        full_content = frontmatter + rendered_content

        plan = InstallationPlan(
            directories_to_create=[commands_dir],
            files_to_write=[FileToWrite(path=command_file, content=full_content)],
        )

        # Commands don't have associated files in Claude Code's flat structure
        # Supporting files would need to be referenced differently

        return plan

    def plan_subagent_installation(
        self,
        subagent: SubAgentConfig,
        plugin: PluginManifest,
        rendered_content: str,
        project_root: Path,
        source_dir: Path,
    ) -> InstallationPlan:
        """Plan sub-agent installation for Claude Code.

        Per Claude Code docs, agents are flat .md files:
        .claude/agents/{plugin}-{agent}.md

        The agent can be triggered by the main Claude instance.
        """
        agents_dir = self.get_subagents_directory(project_root)
        agent_file = agents_dir / f"{plugin.name}-{subagent.name}.md"

        frontmatter = self.generate_subagent_frontmatter(subagent, plugin)
        full_content = frontmatter + rendered_content

        plan = InstallationPlan(
            directories_to_create=[agents_dir],
            files_to_write=[FileToWrite(path=agent_file, content=full_content)],
        )

        # Agents don't have associated files in Claude Code's flat structure
        # Supporting files would need to be referenced differently

        return plan

    def plan_rule_installation(
        self,
        rule: RuleConfig,
        plugin: PluginManifest,
        rendered_content: str,
        project_root: Path,
        source_dir: Path,
    ) -> InstallationPlan:
        """Plan rule installation for Claude Code.

        Rules go to .claude/rules/{plugin}-{rule}.md with optional path scoping.
        """
        rules_dir = self.get_rules_directory(project_root)
        rule_file = rules_dir / f"{plugin.name}-{rule.name}.md"

        # Generate frontmatter with path scoping
        frontmatter = self.generate_rule_frontmatter(rule, plugin)
        full_content = frontmatter + rendered_content

        plan = InstallationPlan(
            directories_to_create=[rules_dir],
            files_to_write=[FileToWrite(path=rule_file, content=full_content)],
        )

        # Add associated files
        self._add_files_to_plan(plan, rule.files, source_dir, rules_dir)
        self._add_files_to_plan(plan, rule.template_files, source_dir, rules_dir, render_as_template=True)

        return plan

    def plan_agent_file_installation(
        self,
        agent_file_config: AgentFileConfig,
        plugin: PluginManifest,
        rendered_content: str,
        project_root: Path,
        source_dir: Path,
    ) -> InstallationPlan:
        """Plan agent file content installation for Claude Code.

        Injects plugin content into CLAUDE.md using marker-based management.
        Each plugin's content is wrapped in markers:
        <!-- dex:plugin:{plugin-name}:start -->
        ... content ...
        <!-- dex:plugin:{plugin-name}:end -->

        This allows multiple plugins to contribute to CLAUDE.md without
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
    # MCP Configuration
    # =========================================================================

    def generate_mcp_config(
        self,
        mcp_server: MCPServerConfig,
        plugin: PluginManifest,
        project_root: Path,
        source_dir: Path,
    ) -> dict[str, Any]:
        """Generate MCP config entry for Claude Code.

        Claude Code uses .mcp.json at project root with mcpServers object.
        """
        config: dict[str, Any] = {}

        if mcp_server.type == "bundled":
            # Bundled server - resolve path
            path = mcp_server.path
            if isinstance(path, dict):
                # Platform-specific paths
                current_os = get_os()
                if current_os in path:
                    path = path[current_os]
                elif is_unix() and "unix" in path:
                    path = path["unix"]
                else:
                    # Fallback to first available
                    path = next(iter(path.values()))

            # Resolve path relative to source directory
            if path and path.startswith("./"):
                path = path[2:]

            # Store path relative to project root for now
            # The actual command will be determined by the file type
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
            # Remote server - determine command based on source prefix
            source = mcp_server.source
            if source:
                if source.startswith("npm:"):
                    # npm package: npx -y package-name
                    package_name = source[4:]
                    config["command"] = "npx"
                    config["args"] = ["-y", package_name]
                elif source.startswith("uvx:"):
                    # uvx package: uvx --from source package-command
                    package_source = source[4:]
                    config["command"] = "uvx"
                    config["args"] = ["--from", package_source]
                elif source.startswith("pip:"):
                    # pip package installed globally
                    package_name = source[4:]
                    config["command"] = package_name
                    config["args"] = []

        # Add additional config (args, env, etc.)
        if mcp_server.config:
            if "args" in mcp_server.config:
                existing_args = config.get("args", [])
                config["args"] = existing_args + mcp_server.config["args"]
            if "env" in mcp_server.config:
                config["env"] = mcp_server.config["env"]
            if "command" in mcp_server.config:
                # Allow explicit override of command
                config["command"] = mcp_server.config["command"]

        return {mcp_server.name: config}

    def merge_mcp_config(
        self,
        existing_config: dict[str, Any],
        new_entries: dict[str, Any],
    ) -> dict[str, Any]:
        """Merge MCP entries into Claude Code .mcp.json format."""
        # Claude Code stores MCP servers under "mcpServers"
        if "mcpServers" not in existing_config:
            existing_config["mcpServers"] = {}

        # Merge new entries
        existing_config["mcpServers"].update(new_entries)

        return existing_config

    # =========================================================================
    # Frontmatter Generation
    # =========================================================================

    def generate_skill_frontmatter(
        self,
        skill: SkillConfig,
        plugin: PluginManifest,
    ) -> str:
        """Generate Claude Code skill frontmatter.

        Per Claude Code docs, SKILL.md files use YAML frontmatter with:
        - name: Skill identifier
        - description: When to use this skill
        - version: Skill version
        """
        lines = [
            "---",
            f"name: {skill.name}",
            f"description: {skill.description}",
            f"version: {plugin.version}",
        ]

        # Add any metadata (allows custom fields like 'plugin')
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
        """Generate Claude Code command frontmatter.

        Per Claude Code docs, command .md files use YAML frontmatter with:
        - description: Brief description shown to user
        - argument-hint: (optional) Hint for arguments like [arg1] [arg2]
        - allowed-tools: (optional) Tools this command can use
        - model: (optional) Model to use (e.g., sonnet, haiku)
        """
        lines = [
            "---",
            f"description: {command.description}",
        ]

        # Add optional argument hint from metadata
        if command.metadata.get("argument_hint"):
            lines.append(f"argument-hint: {command.metadata['argument_hint']}")

        # Add allowed tools - from command config or metadata
        allowed_tools = command.allowed_tools or command.metadata.get("allowed_tools")
        if allowed_tools:
            if isinstance(allowed_tools, list):
                allowed_tools = ", ".join(allowed_tools)
            lines.append(f"allowed-tools: {allowed_tools}")

        # Add model preference from metadata
        if command.metadata.get("model"):
            lines.append(f"model: {command.metadata['model']}")

        lines.append("---")
        lines.append("")  # Blank line after frontmatter

        return "\n".join(lines)

    def generate_subagent_frontmatter(
        self,
        subagent: SubAgentConfig,
        plugin: PluginManifest,
    ) -> str:
        """Generate Claude Code agent frontmatter.

        Per Claude Code docs, agent .md files use YAML frontmatter with:
        - name: Agent identifier
        - description: When to use, with <example> blocks
        - model: inherit or specific model
        - color: Agent color (blue, green, etc.)
        - tools: (optional) List of tools the agent can use
        """
        # Build description with examples if provided
        description = subagent.description
        if subagent.metadata.get("examples"):
            # Examples should already be formatted in the description
            pass

        lines = [
            "---",
            f"name: {subagent.name}",
            f"description: {description}",
            f"model: {subagent.metadata.get('model', 'inherit')}",
            f"color: {subagent.metadata.get('color', 'blue')}",
        ]

        # Add tools if specified - from config or metadata
        tools = subagent.allowed_tools or subagent.metadata.get("tools")
        if tools:
            tools_str = str(tools) if isinstance(tools, list) else tools
            lines.append(f"tools: {tools_str}")

        # Passthrough any additional metadata
        for key, value in subagent.metadata.items():
            if key not in ("examples", "model", "color", "tools"):
                if isinstance(value, bool):
                    lines.append(f"{key}: {'true' if value else 'false'}")
                else:
                    lines.append(f"{key}: {value}")

        lines.append("---")
        lines.append("")  # Blank line after frontmatter

        return "\n".join(lines)

    def generate_rule_frontmatter(
        self,
        rule: RuleConfig,
        plugin: PluginManifest,
    ) -> str:
        """Generate Claude Code rule frontmatter for CLAUDE.md.

        Rules use YAML frontmatter with `paths` field for file pattern scope.
        """
        # If no paths and no metadata, no frontmatter needed
        if not rule.paths and not rule.metadata:
            return ""

        lines = ["---"]

        # Add paths field if specified (Claude Code specific)
        if rule.paths:
            if isinstance(rule.paths, list):
                lines.append("paths:")
                for path in rule.paths:
                    lines.append(f"  - {path}")
            else:
                lines.append(f"paths: {rule.paths}")

        # Passthrough all metadata
        for key, value in rule.metadata.items():
            if isinstance(value, bool):
                lines.append(f"{key}: {'true' if value else 'false'}")
            elif isinstance(value, list):
                lines.append(f"{key}:")
                for item in value:
                    lines.append(f"  - {item}")
            else:
                lines.append(f"{key}: {value}")

        lines.append("---")
        lines.append("")  # Blank line after frontmatter

        return "\n".join(lines)

    # =========================================================================
    # Validation
    # =========================================================================

    def validate_plugin_compatibility(
        self,
        plugin: PluginManifest,
    ) -> list[str]:
        """Validate plugin compatibility with Claude Code."""
        warnings: list[str] = []

        # Claude Code does not support instructions (GitHub Copilot feature)
        if plugin.instructions:
            warnings.append(
                f"Claude Code does not support 'instructions' - "
                f"{len(plugin.instructions)} instruction(s) will be skipped"
            )

        # Claude Code does not support prompts (GitHub Copilot feature)
        if plugin.prompts:
            warnings.append(
                f"Claude Code does not support 'prompts' - "
                f"{len(plugin.prompts)} prompt(s) will be skipped"
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
        """Ensure .claude directories exist before installation."""
        base_dir = self.get_base_directory(project_root)
        base_dir.mkdir(parents=True, exist_ok=True)

        # Ensure all component directories exist
        self.get_skills_directory(project_root).mkdir(parents=True, exist_ok=True)
        self.get_commands_directory(project_root).mkdir(parents=True, exist_ok=True)
        self.get_subagents_directory(project_root).mkdir(parents=True, exist_ok=True)

    def post_install(
        self,
        project_root: Path,
        installed: list[PluginManifest],
    ) -> None:
        """Update .mcp.json after installation if needed."""
        # The installer handles MCP config updates, so this is mainly
        # for any additional cleanup or validation
        pass

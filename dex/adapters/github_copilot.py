"""GitHub Copilot platform adapter.

GitHub Copilot uses .github/ directory for various AI configurations:
- .github/copilot-instructions.md - Repository-wide instructions
- .github/instructions/*.instructions.md - Path-specific instructions
- .github/skills/*/SKILL.md - Skills for structured capabilities
- .github/agents/*.agent.md - Custom agent configurations
- .github/prompts/*.prompt.md - Reusable prompts
"""

from pathlib import Path
from typing import Any

from dex.adapters import register_adapter
from dex.adapters.base import PlatformAdapter
from dex.config.schemas import (  # First-class types this adapter supports; Core types this adapter supports
    AdapterMetadata,
    FileToWrite,
    InstallationPlan,
    InstructionConfig,
    MCPServerConfig,
    PluginManifest,
    PromptConfig,
    RuleConfig,
    SkillConfig,
    SubAgentConfig,
)
from dex.utils.platform import get_os, is_unix


@register_adapter("github-copilot")
class GitHubCopilotAdapter(PlatformAdapter):
    """Adapter for GitHub Copilot.

    Directory structure:
    .github/
    ├── copilot-instructions.md          # Repository-wide instructions (from RuleConfig)
    ├── instructions/
    │   └── {name}.instructions.md       # Path-specific instructions (InstructionConfig)
    ├── skills/
    │   └── {skill-name}/
    │       └── SKILL.md                 # Skills (SkillConfig)
    ├── agents/
    │   └── {agent-name}.agent.md        # Custom agents (SubAgentConfig)
    └── prompts/
        └── {prompt-name}.prompt.md      # Reusable prompts (PromptConfig)

    Path-specific instructions frontmatter:
    - applyTo: Glob pattern for files (e.g., "**/*.py")
    - excludeAgent: "code-review" or "coding-agent" to exclude

    Agent frontmatter:
    - name: Agent identifier
    - description: What the agent does

    Skill frontmatter:
    - name: Skill identifier
    - description: When to use this skill

    Prompt frontmatter:
    - name: Prompt identifier
    - description: What the prompt does
    """

    @property
    def metadata(self) -> AdapterMetadata:
        return AdapterMetadata(
            name="github-copilot",
            display_name="GitHub Copilot",
            description="GitHub Copilot AI assistant",
            mcp_config_file=".vscode/mcp.json",
        )

    # =========================================================================
    # Directory Structure
    # =========================================================================

    def get_base_directory(self, project_root: Path) -> Path:
        return project_root / ".github"

    def get_skills_directory(self, project_root: Path) -> Path:
        """Skills directory at .github/skills/."""
        return self.get_base_directory(project_root) / "skills"

    def get_commands_directory(self, project_root: Path) -> Path:
        """Commands not supported - returns base directory."""
        return self.get_base_directory(project_root)

    def get_subagents_directory(self, project_root: Path) -> Path:
        """Agents directory at .github/agents/."""
        return self.get_base_directory(project_root) / "agents"

    def get_instructions_directory(self, project_root: Path) -> Path:
        """Instructions directory at .github/instructions/."""
        return self.get_base_directory(project_root) / "instructions"

    def get_rules_directory(self, project_root: Path) -> Path:
        """Rules go to .github/ as copilot-instructions.md."""
        return self.get_base_directory(project_root)

    def get_prompts_directory(self, project_root: Path) -> Path:
        """Prompts directory at .github/prompts/."""
        return self.get_base_directory(project_root) / "prompts"

    def get_mcp_config_path(self, project_root: Path) -> Path:
        """MCP config at .vscode/mcp.json."""
        return project_root / ".vscode" / "mcp.json"

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
        """Plan skill installation for GitHub Copilot.

        Creates:
        .github/skills/{skill-name}/SKILL.md
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
        self._add_files_to_plan(plan, skill.template_files, source_dir, skill_dir, render_as_template=True)

        return plan

    def plan_instruction_installation(
        self,
        instruction: InstructionConfig,
        plugin: PluginManifest,
        rendered_content: str,
        project_root: Path,
        source_dir: Path,
    ) -> InstallationPlan:
        """Plan instruction installation for GitHub Copilot.

        Creates:
        .github/instructions/{name}.instructions.md
        """
        instructions_dir = self.get_instructions_directory(project_root)
        instruction_file = instructions_dir / f"{instruction.name}.instructions.md"

        # Generate frontmatter
        frontmatter = self.generate_instruction_frontmatter(instruction, plugin)
        full_content = frontmatter + rendered_content

        plan = InstallationPlan(
            directories_to_create=[instructions_dir],
            files_to_write=[FileToWrite(path=instruction_file, content=full_content)],
        )

        # Add associated files
        self._add_files_to_plan(plan, instruction.files, source_dir, instructions_dir)
        self._add_files_to_plan(plan, instruction.template_files, source_dir, instructions_dir, render_as_template=True)

        return plan

    def plan_subagent_installation(
        self,
        subagent: SubAgentConfig,
        plugin: PluginManifest,
        rendered_content: str,
        project_root: Path,
        source_dir: Path,
    ) -> InstallationPlan:
        """Plan subagent installation for GitHub Copilot.

        Creates:
        .github/agents/{agent-name}.agent.md
        """
        agents_dir = self.get_subagents_directory(project_root)
        agent_file = agents_dir / f"{subagent.name}.agent.md"

        # Generate frontmatter
        frontmatter = self.generate_subagent_frontmatter(subagent, plugin)
        full_content = frontmatter + rendered_content

        plan = InstallationPlan(
            directories_to_create=[agents_dir],
            files_to_write=[FileToWrite(path=agent_file, content=full_content)],
        )

        # Add associated files
        self._add_files_to_plan(plan, subagent.files, source_dir, agents_dir)
        self._add_files_to_plan(plan, subagent.template_files, source_dir, agents_dir, render_as_template=True)

        return plan

    # Note: _add_files_to_plan is inherited from PlatformAdapter base class

    def plan_rule_installation(
        self,
        rule: RuleConfig,
        plugin: PluginManifest,
        rendered_content: str,
        project_root: Path,
        source_dir: Path,
    ) -> InstallationPlan:
        """Plan rule installation for GitHub Copilot.

        Creates .github/copilot-instructions.md with repository-wide instructions.
        Multiple rules are appended to the same file.
        """
        rules_file = self.get_rules_directory(project_root) / "copilot-instructions.md"

        # Generate frontmatter
        frontmatter = self.generate_rule_frontmatter(rule, plugin)
        full_content = frontmatter + rendered_content

        return InstallationPlan(
            directories_to_create=[self.get_rules_directory(project_root)],
            files_to_write=[FileToWrite(path=rules_file, content=full_content)],
        )

    def plan_prompt_installation(
        self,
        prompt: PromptConfig,
        plugin: PluginManifest,
        rendered_content: str,
        project_root: Path,
        source_dir: Path,
    ) -> InstallationPlan:
        """Plan prompt installation for GitHub Copilot.

        Creates:
        .github/prompts/{prompt-name}.prompt.md
        """
        prompts_dir = self.get_prompts_directory(project_root)
        prompt_file = prompts_dir / f"{prompt.name}.prompt.md"

        # Generate frontmatter
        frontmatter = self.generate_prompt_frontmatter(prompt, plugin)
        full_content = frontmatter + rendered_content

        plan = InstallationPlan(
            directories_to_create=[prompts_dir],
            files_to_write=[FileToWrite(path=prompt_file, content=full_content)],
        )

        # Add associated files
        self._add_files_to_plan(plan, prompt.files, source_dir, prompts_dir)
        self._add_files_to_plan(plan, prompt.template_files, source_dir, prompts_dir, render_as_template=True)

        return plan

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
        """Generate MCP config entry for GitHub Copilot.

        GitHub Copilot uses .github/mcp.json with mcpServers object.
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
        """Merge MCP entries into GitHub Copilot .github/mcp.json format."""
        if "mcpServers" not in existing_config:
            existing_config["mcpServers"] = {}

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
        """Generate GitHub Copilot skill frontmatter.

        Skills use YAML frontmatter with:
        - name: Skill identifier
        - description: When to use this skill
        """
        lines = [
            "---",
            f"name: {skill.name}",
            f"description: {skill.description}",
            "---",
            "",  # Blank line after frontmatter
        ]

        return "\n".join(lines)

    def generate_instruction_frontmatter(
        self,
        instruction: InstructionConfig,
        plugin: PluginManifest,
    ) -> str:
        """Generate GitHub Copilot instruction frontmatter.

        Instructions use YAML frontmatter with:
        - applyTo: Glob pattern for auto-attachment
        - description: Brief description of what the instructions do
        - excludeAgent: "code-review" or "coding-agent" to exclude
        """
        apply_to = instruction.apply_to
        exclude_agent = instruction.exclude_agent

        # Build frontmatter lines
        lines = ["---"]

        if apply_to:
            # Handle list of patterns
            if isinstance(apply_to, list):
                apply_to = ",".join(apply_to)
            lines.append(f'applyTo: "{apply_to}"')

        # Always include description if available
        if instruction.description:
            lines.append(f'description: "{instruction.description}"')

        if exclude_agent:
            lines.append(f'excludeAgent: "{exclude_agent}"')

        lines.append("---")
        lines.append("")  # Blank line after frontmatter

        return "\n".join(lines)

    def generate_rule_frontmatter(
        self,
        rule: RuleConfig,
        plugin: PluginManifest,
    ) -> str:
        """Generate GitHub Copilot rule frontmatter.

        Rules for copilot-instructions.md don't typically need frontmatter.
        """
        return ""

    def generate_prompt_frontmatter(
        self,
        prompt: PromptConfig,
        plugin: PluginManifest,
    ) -> str:
        """Generate GitHub Copilot prompt frontmatter.

        Prompts use YAML frontmatter with:
        - name: Prompt identifier
        - description: What the prompt does
        """
        lines = [
            "---",
            f"name: {prompt.name}",
            f"description: {prompt.description}",
        ]

        if prompt.trigger:
            lines.append(f"trigger: {prompt.trigger}")

        lines.append("---")
        lines.append("")  # Blank line after frontmatter

        return "\n".join(lines)

    def generate_subagent_frontmatter(
        self,
        subagent: SubAgentConfig,
        plugin: PluginManifest,
    ) -> str:
        """Generate GitHub Copilot agent frontmatter.

        Agents use YAML frontmatter with:
        - name: Agent identifier
        - description: What the agent does
        """
        lines = [
            "---",
            f"name: {subagent.name}",
            f"description: {subagent.description}",
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
        """Validate plugin compatibility with GitHub Copilot."""
        return []

    # =========================================================================
    # Lifecycle Hooks
    # =========================================================================

    def pre_install(
        self,
        project_root: Path,
        plugins: list[PluginManifest],
    ) -> None:
        """Ensure .github directories exist before installation."""
        base_dir = self.get_base_directory(project_root)
        base_dir.mkdir(parents=True, exist_ok=True)
        self.get_skills_directory(project_root).mkdir(parents=True, exist_ok=True)
        self.get_subagents_directory(project_root).mkdir(parents=True, exist_ok=True)
        self.get_instructions_directory(project_root).mkdir(parents=True, exist_ok=True)
        self.get_prompts_directory(project_root).mkdir(parents=True, exist_ok=True)

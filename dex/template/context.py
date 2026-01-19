"""Template context builder for Dex."""

import os
from pathlib import Path
from typing import Any

from dex.config.schemas import (
    AgentFileConfig,
    CommandConfig,
    InstructionConfig,
    PluginManifest,
    PromptConfig,
    RuleConfig,
    SkillConfig,
    SubAgentConfig,
)
from dex.utils.platform import get_arch, get_os

# Type alias for all component types
ComponentConfig = (
    SkillConfig
    | CommandConfig
    | SubAgentConfig
    | RuleConfig
    | InstructionConfig
    | PromptConfig
    | AgentFileConfig
)


class TemplateContext:
    """Builder for template rendering context.

    This class constructs the context dictionary used when rendering
    Jinja2 templates in context files (.md files).
    """

    def __init__(self) -> None:
        """Initialize an empty context."""
        self._context: dict[str, Any] = {}

    def with_platform(self) -> "TemplateContext":
        """Add platform information to the context.

        Adds:
        - platform.os: "windows" | "linux" | "macos"
        - platform.arch: "x64" | "arm64" | etc.
        """
        self._context["platform"] = {
            "os": get_os(),
            "arch": get_arch(),
        }
        return self

    def with_agent(self, agent_name: str, **kwargs: Any) -> "TemplateContext":
        """Add agent information to the context.

        Args:
            agent_name: Name of the agent (e.g., "claude-code")
            **kwargs: Additional agent properties

        Adds:
        - agent.name: Agent identifier
        - agent.*: Any additional properties
        """
        self._context["agent"] = {
            "name": agent_name,
            **kwargs,
        }
        return self

    def with_environment(
        self, project_root: Path, project_name: str | None = None
    ) -> "TemplateContext":
        """Add environment information to the context.

        Args:
            project_root: Path to the project root directory
            project_name: Optional project name (defaults to directory name)

        Adds:
        - env.project.root: Project root path
        - env.project.name: Project name
        - env.home: User home directory
        - env.*: Environment variables
        """
        if project_name is None:
            project_name = project_root.name

        # Build env dict with environment variables
        env_dict: dict[str, Any] = {
            "project": {
                "root": str(project_root),
                "name": project_name,
            },
            "home": os.path.expanduser("~"),
        }

        # Add all environment variables
        for key, value in os.environ.items():
            # Don't overwrite project or home
            if key not in ("project", "home"):
                env_dict[key] = value

        self._context["env"] = env_dict
        return self

    def with_plugin(self, plugin: PluginManifest) -> "TemplateContext":
        """Add plugin information to the context.

        Args:
            plugin: Plugin manifest

        Adds:
        - plugin.name: Plugin name
        - plugin.version: Plugin version
        - plugin.description: Plugin description
        - plugin.dependencies: List of dependency names
        """
        plugin_ctx: dict[str, Any] = {
            "name": plugin.name,
            "version": plugin.version,
            "description": plugin.description,
            "dependencies": list(plugin.dependencies.keys()),
        }

        self._context["plugin"] = plugin_ctx
        return self

    def with_component(
        self,
        component: ComponentConfig,
        component_type: str,
        context_root: str | None = None,
    ) -> "TemplateContext":
        """Add component information to the context.

        Args:
            component: Component configuration
            component_type: Type of component ("skill", "command", "sub_agent", etc.)
            context_root: Root directory for the component relative to project root
                         (e.g., ".claude/skills/plugin-name/skill-name/")

        Adds:
        - component.name: Component name
        - component.type: Component type
        - context.root: Root directory for the component (if provided)
        """
        self._context["component"] = {
            "name": component.name,
            "type": component_type,
        }
        if context_root:
            if "context" not in self._context:
                self._context["context"] = {}
            self._context["context"]["root"] = context_root
        return self

    def with_custom(self, key: str, value: Any) -> "TemplateContext":
        """Add a custom value to the context.

        Args:
            key: Context key (supports dot notation like "foo.bar")
            value: Value to add

        Returns:
            Self for chaining
        """
        parts = key.split(".")
        target = self._context

        for part in parts[:-1]:
            if part not in target:
                target[part] = {}
            target = target[part]

        target[parts[-1]] = value
        return self

    def merge(self, other: dict[str, Any]) -> "TemplateContext":
        """Merge another dictionary into the context.

        Args:
            other: Dictionary to merge

        Returns:
            Self for chaining
        """
        self._deep_merge(self._context, other)
        return self

    def _deep_merge(self, base: dict[str, Any], override: dict[str, Any]) -> None:
        """Deep merge override into base."""
        for key, value in override.items():
            if key in base and isinstance(base[key], dict) and isinstance(value, dict):
                self._deep_merge(base[key], value)
            else:
                base[key] = value

    def build(self) -> dict[str, Any]:
        """Build and return the final context dictionary.

        Returns:
            Complete context dictionary for template rendering
        """
        return self._context.copy()


def build_context(
    project_root: Path,
    agent_name: str,
    plugin: PluginManifest | None = None,
    component: ComponentConfig | None = None,
    component_type: str | None = None,
    project_name: str | None = None,
    adapter_variables: dict[str, Any] | None = None,
    context_root: str | None = None,
) -> dict[str, Any]:
    """Build a complete template context.

    This is a convenience function that creates a fully populated context.

    Args:
        project_root: Path to the project root
        agent_name: Name of the target agent
        plugin: Optional plugin manifest
        component: Optional component being rendered
        component_type: Type of the component (if component is provided)
        project_name: Optional project name
        adapter_variables: Optional adapter-specific variables
        context_root: Root directory for the component relative to project root
                     (e.g., ".claude/skills/plugin-name/skill-name/")

    Returns:
        Complete context dictionary
    """
    ctx = TemplateContext()
    ctx.with_platform()
    ctx.with_agent(agent_name)
    ctx.with_environment(project_root, project_name)

    if plugin:
        ctx.with_plugin(plugin)

    if component and component_type:
        ctx.with_component(component, component_type, context_root)

    if adapter_variables:
        ctx.merge(adapter_variables)

    return ctx.build()

"""Plugin installation orchestrator.

This module contains the PluginInstaller which coordinates the installation
of plugins. It delegates all platform-specific decisions to the adapter.
"""

import json
import logging
import shutil
import tomllib
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

import tomli_w

from dex.adapters import get_adapter
from dex.adapters.base import PlatformAdapter
from dex.config.parser import load_plugin_manifest
from dex.config.schemas import (
    AgentFileConfig,
    CommandConfig,
    ConditionalContext,
    InstallationPlan,
    InstructionConfig,
    PluginManifest,
    PluginSpec,
    PromptConfig,
    RuleConfig,
    SkillConfig,
    SubAgentConfig,
)
from dex.core.lockfile import LockFileManager
from dex.core.manifest import ManifestManager
from dex.core.project import Project
from dex.registry.base import ResolvedPackage
from dex.registry.factory import create_registry_client, normalize_source
from dex.template.context import build_context
from dex.template.context_resolver import resolve_context_spec
from dex.template.engine import TemplateRenderError, render_file
from dex.utils.filesystem import ensure_directory, write_text_file

logger = logging.getLogger("dex.installer")


class InstallError(Exception):
    """Error during plugin installation."""

    def __init__(self, message: str, plugin_name: str | None = None):
        self.plugin_name = plugin_name
        super().__init__(message)


class FileConflictError(InstallError):
    """Error when install would overwrite unmanaged files."""

    def __init__(self, conflicting_files: list[str], plugin_name: str | None = None):
        self.conflicting_files = conflicting_files
        message = (
            f"Cannot install: {len(conflicting_files)} file(s) would be overwritten that "
            f"are not managed by dex. Use --force to overwrite.\n"
            f"  Conflicting files:\n" + "\n".join(f"    - {f}" for f in conflicting_files[:5])
        )
        if len(conflicting_files) > 5:
            message += f"\n    ... and {len(conflicting_files) - 5} more"
        super().__init__(message, plugin_name)


@dataclass
class InstallResult:
    """Result of a plugin installation."""

    plugin_name: str
    version: str
    success: bool
    message: str = ""
    warnings: list[str] = field(default_factory=list)


@dataclass
class InstallSummary:
    """Summary of an installation operation."""

    results: list[InstallResult] = field(default_factory=list)
    env_warnings: list[str] = field(default_factory=list)

    @property
    def success_count(self) -> int:
        return sum(1 for r in self.results if r.success)

    @property
    def failure_count(self) -> int:
        return sum(1 for r in self.results if not r.success)

    @property
    def all_successful(self) -> bool:
        return all(r.success for r in self.results)


class PluginInstaller:
    """Orchestrates plugin installation.

    The installer coordinates the installation process but delegates
    all platform-specific decisions to the adapter. There is NO
    platform-specific logic in this class.
    """

    def __init__(self, project: Project, force: bool = False):
        """Initialize the installer.

        Args:
            project: The project to install plugins into
            force: If True, overwrite existing unmanaged files without error
        """
        self.project = project
        self.force = force
        self.adapter: PlatformAdapter = get_adapter(project.agent)
        self.lockfile_manager = LockFileManager(project.root, project.agent)
        self.manifest_manager = ManifestManager(project.root)
        self._temp_dirs: list[Path] = []
        self._current_plugin: str | None = None  # Track which plugin is being installed
        self._backup_dir: Path | None = None  # Track backup directory for rollback
        self._backed_up_files: list[tuple[Path, Path]] = (
            []
        )  # (original, backup) pairs  # Track which plugin is being installed

    def install(
        self,
        plugin_specs: dict[str, str | PluginSpec] | None = None,
        use_lockfile: bool = True,
        update_lockfile: bool = True,
    ) -> InstallSummary:
        """Install plugins.

        Args:
            plugin_specs: Dictionary of plugin specs to install, or None to use project config
            use_lockfile: Whether to use locked versions from dex.lock
            update_lockfile: Whether to update the lock file after installation

        Returns:
            InstallSummary with results for each plugin
        """
        if plugin_specs is None:
            plugin_specs = self.project.plugins

        if not plugin_specs:
            logger.info("No plugins to install")
            return InstallSummary()

        logger.info("Starting installation of %d plugin(s)", len(plugin_specs))

        # Load lock file if using it
        if use_lockfile:
            self.lockfile_manager.load()
            logger.debug("Loaded lock file")

        # Resolve all plugins
        resolved_plugins: list[tuple[str, PluginSpec, ResolvedPackage]] = []
        for name, spec in plugin_specs.items():
            if isinstance(spec, str):
                spec = PluginSpec(version=spec)

            logger.info("Resolving plugin: %s", name)
            resolved = self._resolve_plugin(name, spec, use_lockfile)
            if resolved:
                logger.debug(
                    "Resolved %s to version %s from %s",
                    name,
                    resolved.version,
                    resolved.resolved_url,
                )
                resolved_plugins.append((name, spec, resolved))

        # Collect manifests for pre-install hook
        manifests: list[PluginManifest] = []
        for _name, _spec, resolved in resolved_plugins:
            manifest = self._fetch_and_load_manifest(resolved)
            if manifest:
                manifests.append(manifest)

        # Pre-install hook
        self.adapter.pre_install(self.project.root, manifests)

        summary = InstallSummary()

        # Install each plugin
        mcp_configs: dict[str, Any] = {}
        installed_manifests: list[PluginManifest] = []

        for name, spec, resolved in resolved_plugins:
            result = self._install_single_plugin(name, spec, resolved, mcp_configs)
            summary.results.append(result)

            if result.success and update_lockfile:
                # Lock the installed version
                self.lockfile_manager.lock_plugin(
                    name=name,
                    version=resolved.version,
                    resolved_url=resolved.resolved_url,
                    integrity=resolved.integrity,
                )

            if result.success:
                manifest = self._fetch_and_load_manifest(resolved)
                if manifest:
                    installed_manifests.append(manifest)

        # Update MCP config if any servers were installed
        if mcp_configs:
            logger.info("Updating MCP config with %d server(s)", len(mcp_configs))
            self._update_mcp_config(mcp_configs)

        # Post-install hook
        self.adapter.post_install(self.project.root, installed_manifests)

        # Save lock file
        if update_lockfile:
            self.lockfile_manager.save()
            logger.debug("Saved lock file")

        # Save manifest
        self.manifest_manager.save()
        logger.debug("Saved manifest")

        # Collect environment variable warnings
        summary.env_warnings = self._collect_env_warnings(installed_manifests)

        # Cleanup temp directories
        self._cleanup_temp_dirs()

        logger.info(
            "Installation complete: %d succeeded, %d failed",
            summary.success_count,
            summary.failure_count,
        )

        return summary

    def _resolve_plugin(
        self,
        name: str,
        spec: PluginSpec,
        use_lockfile: bool,
    ) -> ResolvedPackage | None:
        """Resolve a plugin specification to a downloadable package."""
        # Check for locked version - only applies to registry-based resolution (not source)
        # Only use locked version if:
        # 1. No explicit version specified (use locked)
        # 2. Explicit version matches locked version (use locked for stability)
        # If user specifies a different version, they want to upgrade/downgrade
        if use_lockfile and not spec.source:
            locked_version = self.lockfile_manager.get_locked_version(name)
            if locked_version and (not spec.version or spec.version == locked_version):
                spec = PluginSpec(
                    version=locked_version,
                    registry=spec.registry,
                )

        # Handle direct source - use "package" mode since source points to a single package
        if spec.source:
            source = normalize_source(spec.source)
            client = create_registry_client(source, mode="package")
            return client.resolve_package(name, spec.version or "latest")

        # Handle registry-based resolution
        version = spec.version or "latest"
        registry_url: str | None = None

        # Check if spec.registry is a direct URL (starts with file://, http://, s3://, az://, git+, etc.)
        if spec.registry and (
            spec.registry.startswith("file://")
            or spec.registry.startswith("http://")
            or spec.registry.startswith("https://")
            or spec.registry.startswith("s3://")
            or spec.registry.startswith("az://")
            or spec.registry.startswith("git+")
        ):
            registry_url = spec.registry
            logger.debug("Using direct registry URL: %s", registry_url)
        else:
            # Look up registry by name
            registry_url = self.project.get_registry_url(spec.registry)

        if registry_url:
            client = create_registry_client(registry_url)
            return client.resolve_package(name, version)

        return None

    def _fetch_and_load_manifest(self, resolved: ResolvedPackage) -> PluginManifest | None:
        """Fetch a package and load its manifest."""
        if resolved.local_path and resolved.local_path.is_dir():
            # Direct local directory
            try:
                return load_plugin_manifest(resolved.local_path)
            except Exception:
                return None

        if resolved.local_path and resolved.local_path.is_file():
            # Local tarball - extract it
            import tempfile

            from dex.utils.filesystem import extract_tarball

            temp_dir = Path(tempfile.mkdtemp(prefix="dex_"))
            self._temp_dirs.append(temp_dir)

            try:
                plugin_dir = extract_tarball(resolved.local_path, temp_dir)
                return load_plugin_manifest(plugin_dir)
            except Exception:
                return None

        if resolved.resolved_url:
            # Fetch from remote
            import tempfile

            temp_dir = Path(tempfile.mkdtemp(prefix="dex_"))
            self._temp_dirs.append(temp_dir)

            try:
                client = create_registry_client(resolved.resolved_url)
                plugin_dir = client.fetch_package(resolved, temp_dir)
                return load_plugin_manifest(plugin_dir)
            except Exception:
                return None

        # No local path or URL - can't fetch
        return None

    def _install_single_plugin(
        self,
        name: str,
        spec: PluginSpec,
        resolved: ResolvedPackage,
        mcp_configs: dict[str, Any],
    ) -> InstallResult:
        """Install a single plugin."""
        # Track which plugin we're installing for manifest
        self._current_plugin = name
        # Clear backups from previous plugin install
        self._backed_up_files.clear()

        try:
            # Get the plugin source directory
            if resolved.local_path and resolved.local_path.is_dir():
                source_dir = resolved.local_path
            elif resolved.local_path and resolved.local_path.is_file():
                # Local tarball - extract it directly
                import tempfile

                from dex.utils.filesystem import extract_tarball

                temp_dir = Path(tempfile.mkdtemp(prefix="dex_"))
                self._temp_dirs.append(temp_dir)

                source_dir = extract_tarball(resolved.local_path, temp_dir)
            elif resolved.resolved_url:
                # Fetch from remote
                import tempfile

                from dex.utils.filesystem import extract_tarball

                temp_dir = Path(tempfile.mkdtemp(prefix="dex_"))
                self._temp_dirs.append(temp_dir)

                # Create a client from the resolved URL and fetch
                client = create_registry_client(resolved.resolved_url)
                source_dir = client.fetch_package(resolved, temp_dir)
            else:
                raise InstallError(
                    f"Cannot fetch package: no local path or URL for {name}",
                    plugin_name=name,
                )

            # Load manifest
            manifest = load_plugin_manifest(source_dir)

            # Validate compatibility
            warnings = self.adapter.validate_plugin_compatibility(manifest)

            # Get adapter template variables
            adapter_vars = self.adapter.get_template_variables(self.project.root, manifest)

            # Get old files from dex manifest for cleanup and conflict exclusion during reinstall
            old_files = self._get_old_plugin_files(name)

            # Clear old manifest entry so we start fresh
            self.manifest_manager.remove_plugin(name)

            # Install skills
            for skill in manifest.skills:
                self._install_skill(skill, manifest, source_dir, adapter_vars, old_files)

            # Install commands
            for command in manifest.commands:
                self._install_command(command, manifest, source_dir, adapter_vars, old_files)

            # Install sub-agents
            for subagent in manifest.sub_agents:
                self._install_subagent(subagent, manifest, source_dir, adapter_vars, old_files)

            # Install rules (list[RuleConfig])
            for rule in manifest.rules:
                self._install_rule(rule, manifest, source_dir, adapter_vars, old_files)

            # Install instructions (list[InstructionConfig])
            for instruction in manifest.instructions:
                self._install_instruction(
                    instruction, manifest, source_dir, adapter_vars, old_files
                )

            # Install prompts (list[PromptConfig])
            for prompt in manifest.prompts:
                self._install_prompt(prompt, manifest, source_dir, adapter_vars, old_files)

            # Install agent file content if specified
            if manifest.agent_file:
                self._install_agent_file(
                    manifest.agent_file, manifest, source_dir, adapter_vars, old_files
                )

            # Collect MCP configs and track in manifest
            for mcp_server in manifest.mcp_servers:
                mcp_entry = self.adapter.generate_mcp_config(
                    mcp_server, manifest, self.project.root, source_dir
                )
                mcp_configs.update(mcp_entry)
                # Track which plugin added this server
                self.manifest_manager.add_mcp_server(name, mcp_server.name)

            # Clean up old files that are no longer used (after install so we know new files)
            self._cleanup_old_files(name, old_files)

            # Clean up backups on success
            self._cleanup_backups()

            return InstallResult(
                plugin_name=name,
                version=resolved.version,
                success=True,
                message=f"Installed {name}@{resolved.version}",
                warnings=warnings,
            )

        except FileConflictError:
            # Re-raise conflict errors without rollback (nothing was written)
            raise

        except Exception as e:
            # Rollback any changes on failure
            self._rollback_backups()

            return InstallResult(
                plugin_name=name,
                version=resolved.version,
                success=False,
                message=str(e),
            )
        finally:
            self._current_plugin = None

    def _get_old_plugin_files(self, plugin_name: str) -> set[str]:
        """Get set of files from old plugin installation for cleanup comparison."""
        old_plugin_files = self.manifest_manager.get_plugin_files(plugin_name)
        if not old_plugin_files:
            return set()
        return set(old_plugin_files.files)

    def _cleanup_old_files(self, plugin_name: str, old_files: set[str]) -> None:
        """Remove files that were in old installation but not in new one."""
        if not old_files:
            return

        # Get current files from manifest (just installed)
        new_plugin_files = self.manifest_manager.get_plugin_files(plugin_name)
        new_files = set(new_plugin_files.files) if new_plugin_files else set()

        # Files to remove: in old but not in new
        files_to_remove = old_files - new_files

        for rel_path in files_to_remove:
            file_path = self.project.root / rel_path
            if file_path.exists():
                file_path.unlink()
                # Try to remove empty parent directories
                try:
                    parent = file_path.parent
                    while parent != self.project.root:
                        if parent.is_dir() and not any(parent.iterdir()):
                            parent.rmdir()
                            parent = parent.parent
                        else:
                            break
                except OSError:
                    pass

    def _install_skill(
        self,
        skill: SkillConfig,
        manifest: PluginManifest,
        source_dir: Path,
        adapter_vars: dict[str, Any],
        old_files: set[str] | None = None,
    ) -> None:
        """Install a single skill."""
        # Get the context root directory from adapter
        skill_dir = self.adapter.get_skill_install_directory(skill, manifest, self.project.root)
        try:
            context_root = str(skill_dir.relative_to(self.project.root)) + "/"
        except ValueError:
            context_root = str(skill_dir) + "/"

        # Build template context
        context = build_context(
            project_root=self.project.root,
            agent_name=self.project.agent,
            plugin=manifest,
            component=skill,
            component_type="skill",
            project_name=self.project.project_name,
            adapter_variables=adapter_vars,
            context_root=context_root,
        )

        # Render the context file
        rendered_content = self._render_context(skill.context, source_dir, context)

        # Get installation plan from adapter
        plan = self.adapter.plan_skill_installation(
            skill=skill,
            plugin=manifest,
            rendered_content=rendered_content,
            project_root=self.project.root,
            source_dir=source_dir,
        )

        # Execute the plan
        self._execute_plan(plan, old_files)

    def _install_command(
        self,
        command: CommandConfig,
        manifest: PluginManifest,
        source_dir: Path,
        adapter_vars: dict[str, Any],
        old_files: set[str] | None = None,
    ) -> None:
        """Install a single command."""
        # Get the context root directory from adapter (commands are flat)
        commands_dir = self.adapter.get_commands_directory(self.project.root)
        try:
            context_root = str(commands_dir.relative_to(self.project.root)) + "/"
        except ValueError:
            context_root = str(commands_dir) + "/"

        context = build_context(
            project_root=self.project.root,
            agent_name=self.project.agent,
            plugin=manifest,
            component=command,
            component_type="command",
            project_name=self.project.project_name,
            adapter_variables=adapter_vars,
            context_root=context_root,
        )

        rendered_content = self._render_context(command.context, source_dir, context)

        plan = self.adapter.plan_command_installation(
            command=command,
            plugin=manifest,
            rendered_content=rendered_content,
            project_root=self.project.root,
            source_dir=source_dir,
        )

        self._execute_plan(plan, old_files)

    def _install_subagent(
        self,
        subagent: SubAgentConfig,
        manifest: PluginManifest,
        source_dir: Path,
        adapter_vars: dict[str, Any],
        old_files: set[str] | None = None,
    ) -> None:
        """Install a single sub-agent."""
        # Get the context root directory from adapter (sub-agents are flat)
        subagents_dir = self.adapter.get_subagents_directory(self.project.root)
        try:
            context_root = str(subagents_dir.relative_to(self.project.root)) + "/"
        except ValueError:
            context_root = str(subagents_dir) + "/"

        context = build_context(
            project_root=self.project.root,
            agent_name=self.project.agent,
            plugin=manifest,
            component=subagent,
            component_type="sub_agent",
            project_name=self.project.project_name,
            adapter_variables=adapter_vars,
            context_root=context_root,
        )

        rendered_content = self._render_context(subagent.context, source_dir, context)

        plan = self.adapter.plan_subagent_installation(
            subagent=subagent,
            plugin=manifest,
            rendered_content=rendered_content,
            project_root=self.project.root,
            source_dir=source_dir,
        )

        self._execute_plan(plan, old_files)

    def _install_rule(
        self,
        rule: RuleConfig,
        manifest: PluginManifest,
        source_dir: Path,
        adapter_vars: dict[str, Any],
        old_files: set[str] | None = None,
    ) -> None:
        """Install a single rule."""
        rules_dir = self.adapter.get_rules_directory(self.project.root)
        try:
            context_root = str(rules_dir.relative_to(self.project.root)) + "/"
        except ValueError:
            context_root = str(rules_dir) + "/"

        context = build_context(
            project_root=self.project.root,
            agent_name=self.project.agent,
            plugin=manifest,
            component=rule,
            component_type="rule",
            project_name=self.project.project_name,
            adapter_variables=adapter_vars,
            context_root=context_root,
        )

        rendered_content = self._render_context(rule.context, source_dir, context)

        plan = self.adapter.plan_rule_installation(
            rule=rule,
            plugin=manifest,
            rendered_content=rendered_content,
            project_root=self.project.root,
            source_dir=source_dir,
        )

        self._execute_plan(plan, old_files)

    def _install_instruction(
        self,
        instruction: InstructionConfig,
        manifest: PluginManifest,
        source_dir: Path,
        adapter_vars: dict[str, Any],
        old_files: set[str] | None = None,
    ) -> None:
        """Install a single instruction."""
        instructions_dir = self.adapter.get_instructions_directory(self.project.root)
        try:
            context_root = str(instructions_dir.relative_to(self.project.root)) + "/"
        except ValueError:
            context_root = str(instructions_dir) + "/"

        context = build_context(
            project_root=self.project.root,
            agent_name=self.project.agent,
            plugin=manifest,
            component=instruction,
            component_type="instruction",
            project_name=self.project.project_name,
            adapter_variables=adapter_vars,
            context_root=context_root,
        )

        rendered_content = self._render_context(instruction.context, source_dir, context)

        plan = self.adapter.plan_instruction_installation(
            instruction=instruction,
            plugin=manifest,
            rendered_content=rendered_content,
            project_root=self.project.root,
            source_dir=source_dir,
        )

        self._execute_plan(plan, old_files)

    def _install_prompt(
        self,
        prompt: PromptConfig,
        manifest: PluginManifest,
        source_dir: Path,
        adapter_vars: dict[str, Any],
        old_files: set[str] | None = None,
    ) -> None:
        """Install a single prompt."""
        prompts_dir = self.adapter.get_prompts_directory(self.project.root)
        try:
            context_root = str(prompts_dir.relative_to(self.project.root)) + "/"
        except ValueError:
            context_root = str(prompts_dir) + "/"

        context = build_context(
            project_root=self.project.root,
            agent_name=self.project.agent,
            plugin=manifest,
            component=prompt,
            component_type="prompt",
            project_name=self.project.project_name,
            adapter_variables=adapter_vars,
            context_root=context_root,
        )

        rendered_content = self._render_context(prompt.context, source_dir, context)

        plan = self.adapter.plan_prompt_installation(
            prompt=prompt,
            plugin=manifest,
            rendered_content=rendered_content,
            project_root=self.project.root,
            source_dir=source_dir,
        )

        self._execute_plan(plan, old_files)

    def _install_agent_file(
        self,
        agent_file_config: AgentFileConfig,
        manifest: PluginManifest,
        source_dir: Path,
        adapter_vars: dict[str, Any],
        old_files: set[str] | None = None,
    ) -> None:
        """Install content into the agent file (CLAUDE.md, AGENTS.md, etc.).

        This uses marker-based content management to safely inject plugin
        content into shared agent files without affecting other plugins
        or user content.
        """
        agent_file_path = self.adapter.get_agent_file_path(self.project.root)
        if not agent_file_path:
            # Adapter doesn't support agent file content injection
            return

        try:
            context_root = str(agent_file_path.parent.relative_to(self.project.root)) + "/"
        except ValueError:
            context_root = str(agent_file_path.parent) + "/"

        context = build_context(
            project_root=self.project.root,
            agent_name=self.project.agent,
            plugin=manifest,
            component=agent_file_config,
            component_type="agent_file",
            project_name=self.project.project_name,
            adapter_variables=adapter_vars,
            context_root=context_root,
        )

        rendered_content = self._render_context(agent_file_config.context, source_dir, context)

        plan = self.adapter.plan_agent_file_installation(
            agent_file_config=agent_file_config,
            plugin=manifest,
            rendered_content=rendered_content,
            project_root=self.project.root,
            source_dir=source_dir,
        )

        self._execute_plan(plan, old_files)

    def _render_context(
        self,
        context_spec: str | list[str | ConditionalContext],
        source_dir: Path,
        template_context: dict[str, Any],
    ) -> str:
        """Render context file(s) to a single string.

        Resolves platform-specific context overrides before rendering.
        """
        # Resolve platform-specific context files
        resolved_spec = resolve_context_spec(context_spec, source_dir, self.project.agent)

        if isinstance(resolved_spec, str):
            # Single file
            path = resolved_spec
            if path.startswith("./"):
                path = path[2:]
            full_path = source_dir / path

            try:
                return render_file(full_path, template_context)
            except FileNotFoundError:
                return f"<!-- Context file not found: {path} -->\n"
            except TemplateRenderError as e:
                raise InstallError(f"Template error in {path}: {e}") from e

        elif isinstance(resolved_spec, list):
            # Multiple files or conditional includes
            parts: list[str] = []
            for item in resolved_spec:
                if isinstance(item, str):
                    path = item
                    if path.startswith("./"):
                        path = path[2:]
                    full_path = source_dir / path

                    try:
                        parts.append(render_file(full_path, template_context))
                    except FileNotFoundError:
                        parts.append(f"<!-- Context file not found: {path} -->\n")
                    except TemplateRenderError as e:
                        raise InstallError(f"Template error in {path}: {e}") from e

                elif isinstance(item, dict):
                    # Conditional include
                    path = item.get("path", "")
                    condition = item.get("if", "")

                    if self._evaluate_condition(condition, template_context):
                        if path.startswith("./"):
                            path = path[2:]
                        full_path = source_dir / path

                        try:
                            parts.append(render_file(full_path, template_context))
                        except FileNotFoundError:
                            pass
                        except TemplateRenderError as e:
                            raise InstallError(f"Template error in {path}: {e}") from e

            return "\n".join(parts)

        return ""

    def _evaluate_condition(self, condition: str, context: dict[str, Any]) -> bool:
        """Evaluate a simple condition string against the context."""
        # Simple evaluation: "platform.os == 'windows'"
        # This is a basic implementation - a more robust one would use
        # the Jinja2 environment's evaluation
        try:
            # Create a simple expression evaluator
            from jinja2 import Environment

            env = Environment()
            template = env.from_string(f"{{{{ {condition} }}}}")
            result = template.render(context)
            return result.lower() in ("true", "1", "yes")
        except Exception:
            return False

    def _ensure_backup_dir(self) -> Path:
        """Ensure backup directory exists and has proper gitignore."""
        backup_dir = self.manifest_manager.manifest_dir / "cache"
        backup_dir.mkdir(parents=True, exist_ok=True)

        # Ensure .dex/.gitignore exists and excludes cache
        gitignore_path = self.manifest_manager.manifest_dir / ".gitignore"
        gitignore_content = "cache/\n"

        if gitignore_path.exists():
            existing = gitignore_path.read_text()
            if "cache/" not in existing:
                gitignore_path.write_text(existing.rstrip() + "\n" + gitignore_content)
        else:
            self.manifest_manager.manifest_dir.mkdir(parents=True, exist_ok=True)
            gitignore_path.write_text(gitignore_content)

        self._backup_dir = backup_dir
        return backup_dir

    def _check_file_conflicts(
        self, plan: InstallationPlan, exclude_files: set[str] | None = None
    ) -> list[str]:
        """Check if plan would overwrite unmanaged files.

        Args:
            plan: The installation plan to check
            exclude_files: Optional set of relative paths to exclude from conflict check
                          (e.g., files being reinstalled from the same plugin)

        Returns:
            List of relative paths that would conflict
        """
        conflicts: list[str] = []
        managed_files = self.manifest_manager.get_all_managed_files()
        exclude_files = exclude_files or set()

        # Check files to write
        for file_to_write in plan.files_to_write:
            if file_to_write.path.exists():
                try:
                    rel_path = str(file_to_write.path.relative_to(self.project.root))
                except ValueError:
                    rel_path = str(file_to_write.path)

                # File is a conflict if it exists, isn't managed, and isn't being excluded
                if rel_path not in managed_files and rel_path not in exclude_files:
                    conflicts.append(rel_path)

        # Check files to copy
        for _src, dest in plan.files_to_copy.items():
            if dest.exists():
                try:
                    rel_path = str(dest.relative_to(self.project.root))
                except ValueError:
                    rel_path = str(dest)

                if rel_path not in managed_files and rel_path not in exclude_files:
                    conflicts.append(rel_path)

        return conflicts

    def _backup_file(self, file_path: Path) -> None:
        """Backup a file before overwriting it."""
        if not file_path.exists():
            return

        backup_dir = self._ensure_backup_dir()
        try:
            rel_path = file_path.relative_to(self.project.root)
        except ValueError:
            rel_path = Path(file_path.name)

        backup_path = backup_dir / rel_path
        backup_path.parent.mkdir(parents=True, exist_ok=True)

        shutil.copy2(file_path, backup_path)
        self._backed_up_files.append((file_path, backup_path))
        logger.debug("Backed up %s to %s", file_path, backup_path)

    def _rollback_backups(self) -> None:
        """Restore all backed up files."""
        for original, backup in reversed(self._backed_up_files):
            if backup.exists():
                shutil.copy2(backup, original)
                logger.debug("Restored %s from backup", original)

        # Clean up backups
        self._cleanup_backups()

    def _cleanup_backups(self) -> None:
        """Clean up backup files after successful install."""
        if self._backup_dir and self._backup_dir.exists():
            shutil.rmtree(self._backup_dir, ignore_errors=True)
        self._backed_up_files.clear()

    def _execute_plan(
        self, plan: InstallationPlan, exclude_conflict_files: set[str] | None = None
    ) -> None:
        """Execute an installation plan.

        Checks for file conflicts before writing. Creates backups of existing
        files that will be overwritten (for rollback capability).

        Args:
            plan: The installation plan to execute
            exclude_conflict_files: Files to exclude from conflict check (e.g., old plugin files)
        """
        # Check for file conflicts (unless --force is used)
        if not self.force:
            conflicts = self._check_file_conflicts(plan, exclude_conflict_files)
            if conflicts:
                raise FileConflictError(conflicts, self._current_plugin)

        # Create directories
        for dir_path in plan.directories_to_create:
            ensure_directory(dir_path)
            # Track directory in manifest
            if self._current_plugin:
                self.manifest_manager.add_directory(self._current_plugin, dir_path)

        # Write files (with backup)
        for file_to_write in plan.files_to_write:
            # Backup existing file before overwriting
            self._backup_file(file_to_write.path)

            write_text_file(file_to_write.path, file_to_write.content)
            # Track file in manifest
            if self._current_plugin:
                self.manifest_manager.add_file(self._current_plugin, file_to_write.path)

        # Copy files (with backup)
        for src, dest in plan.files_to_copy.items():
            if src.exists():
                ensure_directory(dest.parent)

                # Backup existing file before overwriting
                self._backup_file(dest)

                shutil.copy2(src, dest)
                # Track copied file in manifest
                if self._current_plugin:
                    self.manifest_manager.add_file(self._current_plugin, dest)

    def _update_mcp_config(self, mcp_configs: dict[str, Any]) -> None:
        """Update the MCP configuration file.

        Supports both JSON and TOML formats based on file extension.
        - .json files use JSON format
        - .toml files use TOML format (for Codex)
        """
        config_path = self.adapter.get_mcp_config_path(self.project.root)

        # Skip if adapter doesn't support MCP configuration
        if config_path is None:
            return

        is_toml = config_path.suffix == ".toml"

        # Load existing config or create empty
        existing: dict[str, Any] = {}
        if config_path.exists():
            try:
                with open(
                    config_path, "rb" if is_toml else "r", encoding=None if is_toml else "utf-8"
                ) as f:
                    existing = tomllib.load(f) if is_toml else json.load(f)
            except (json.JSONDecodeError, tomllib.TOMLDecodeError, OSError):
                pass

        # Merge new configs
        merged = self.adapter.merge_mcp_config(existing, mcp_configs)

        # Save
        ensure_directory(config_path.parent)
        with open(
            config_path, "wb" if is_toml else "w", encoding=None if is_toml else "utf-8"
        ) as f:
            if is_toml:
                tomli_w.dump(merged, f)
            else:
                json.dump(merged, f, indent=2)
                f.write("\n")

    def _collect_env_warnings(self, manifests: list[PluginManifest]) -> list[str]:
        """Collect warnings about unset environment variables."""
        import os

        warnings: list[str] = []

        for manifest in manifests:
            for var_name, var_config in manifest.env_variables.items():
                if var_config.required and not os.environ.get(var_name):
                    warnings.append(
                        f"{var_name} (required by {manifest.name}): {var_config.description}"
                    )

        return warnings

    def _cleanup_temp_dirs(self) -> None:
        """Clean up temporary directories."""
        for temp_dir in self._temp_dirs:
            if temp_dir.exists():
                shutil.rmtree(temp_dir, ignore_errors=True)
        self._temp_dirs.clear()

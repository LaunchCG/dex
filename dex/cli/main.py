"""Main CLI application for Dex."""

import contextlib
import logging
from pathlib import Path
from typing import Annotated

import typer
from rich.console import Console
from rich.logging import RichHandler
from rich.table import Table

from dex import __version__
from dex.adapters import get_adapter, list_adapters
from dex.config.parser import ConfigError
from dex.config.schemas import AgentType, PluginSpec
from dex.core.installer import PluginInstaller
from dex.core.project import Project

# Create the main Typer app
app = typer.Typer(
    name="dex",
    help="AI Context Manager for AI-augmented development tools",
    add_completion=False,
    no_args_is_help=True,
)

console = Console()
error_console = Console(stderr=True)

# Set up logger for the dex package
logger = logging.getLogger("dex")


def setup_logging(verbosity: int) -> None:
    """Configure logging based on verbosity level.

    Args:
        verbosity: 0=WARNING, 1=INFO, 2=DEBUG, 3+=TRACE (DEBUG with extra detail)
    """
    if verbosity == 0:
        level = logging.WARNING
    elif verbosity == 1:
        level = logging.INFO
    elif verbosity >= 2:
        level = logging.DEBUG

    # Configure the root dex logger
    logger.setLevel(level)

    # Only add handler if not already configured
    if not logger.handlers:
        handler = RichHandler(
            console=error_console,
            show_time=verbosity >= 2,
            show_path=verbosity >= 3,
            rich_tracebacks=True,
        )
        handler.setLevel(level)
        logger.addHandler(handler)
    else:
        # Update existing handler level
        for h in logger.handlers:
            h.setLevel(level)


def print_error(message: str) -> None:
    """Print an error message to stderr."""
    error_console.print(f"[red]Error:[/red] {message}")


def print_success(message: str) -> None:
    """Print a success message."""
    console.print(f"[green]\u2713[/green] {message}")


def print_warning(message: str) -> None:
    """Print a warning message."""
    console.print(f"[yellow]\u26a0[/yellow] {message}")


def print_debug(message: str) -> None:
    """Print a debug message (only shown at -vv or higher)."""
    logger.debug(message)


def print_info(message: str) -> None:
    """Print an info message (only shown at -v or higher)."""
    logger.info(message)


def get_project(path: Path | None = None) -> Project:
    """Get the current project, raising an error if not found."""
    try:
        return Project.load(path)
    except FileNotFoundError as e:
        print_error(str(e))
        print_error("Run 'dex init' to create a new project")
        raise typer.Exit(1) from e
    except ConfigError as e:
        print_error(str(e))
        raise typer.Exit(1) from e


@app.callback()
def callback(
    verbose: Annotated[
        int,
        typer.Option(
            "--verbose",
            "-v",
            count=True,
            help="Increase verbosity (-v info, -vv debug, -vvv trace)",
        ),
    ] = 0,
) -> None:
    """Dex - AI Context Manager for AI-augmented development tools."""
    setup_logging(verbose)


@app.command()
def version() -> None:
    """Show the Dex version."""
    console.print(f"dex {__version__}")


@app.command()
def init(
    agent: Annotated[
        AgentType,
        typer.Option(
            "--agent",
            "-a",
            help="Target AI agent platform",
        ),
    ] = "claude-code",
    project_name: Annotated[
        str | None,
        typer.Option(
            "--name",
            "-n",
            help="Project name (defaults to directory name)",
        ),
    ] = None,
    path: Annotated[
        Path | None,
        typer.Option(
            "--path",
            "-p",
            help="Project directory (defaults to current directory)",
        ),
    ] = None,
) -> None:
    """Initialize a new Dex project.

    Creates a dex.yaml configuration file in the specified directory.
    """
    path = Path.cwd() if path is None else path.resolve()

    if not path.exists():
        print_error(f"Directory does not exist: {path}")
        raise typer.Exit(1)

    # Check if project already exists
    if (path / "dex.yaml").exists():
        print_error(f"Project already initialized in {path}")
        print_error("To reinitialize, delete dex.yaml first")
        raise typer.Exit(1)

    # Validate agent
    available_agents = list_adapters()
    if agent not in available_agents:
        print_error(f"Unknown agent: {agent}")
        print_error(f"Available agents: {', '.join(available_agents)}")
        raise typer.Exit(1)

    try:
        Project.init(path, agent, project_name)
        print_success(f"Initialized Dex project for {agent}")
        console.print(f"  Created: {path / 'dex.yaml'}")

        # Create platform-specific directory
        adapter = get_adapter(agent)
        base_dir = adapter.get_base_directory(path)
        base_dir.mkdir(parents=True, exist_ok=True)
        console.print(f"  Created: {base_dir}")

    except Exception as e:
        print_error(f"Failed to initialize project: {e}")
        raise typer.Exit(1) from e


@app.command()
def install(
    plugins: Annotated[
        list[str] | None,
        typer.Argument(
            help="Plugins to install (e.g., 'plugin-name', 'plugin-name@1.2.0', "
            "'plugin-name@^1.0.0', 'plugin-name@>=1.0.0')",
        ),
    ] = None,
    source: Annotated[
        str | None,
        typer.Option(
            "--source",
            "-s",
            help="Install from a direct source (file:// path)",
        ),
    ] = None,
    registry: Annotated[
        str | None,
        typer.Option(
            "--registry",
            "-r",
            help="Registry to use for this installation (overrides default)",
        ),
    ] = None,
    save: Annotated[
        bool,
        typer.Option(
            "--save",
            "-S",
            help="Save the installed plugins to dex.yaml",
        ),
    ] = False,
    no_lock: Annotated[
        bool,
        typer.Option(
            "--no-lock",
            help="Don't update the lock file",
        ),
    ] = False,
    force: Annotated[
        bool,
        typer.Option(
            "--force",
            "-f",
            help="Overwrite existing files even if not managed by dex",
        ),
    ] = False,
    path: Annotated[
        Path | None,
        typer.Option(
            "--path",
            "-p",
            help="Project directory",
        ),
    ] = None,
) -> None:
    """Install plugins.

    Without arguments, installs all plugins from dex.yaml using versions
    from the lock file.

    With plugin names, installs the specified plugins. Use --save to add
    them to dex.yaml. Supports version specifiers:
      - plugin-name         (latest version)
      - plugin-name@1.2.0   (exact version)
      - plugin-name@^1.0.0  (compatible with 1.x.x)
      - plugin-name@~1.2.0  (compatible with 1.2.x)
      - plugin-name@>=1.0.0 (1.0.0 or higher)

    By default, installation will fail if it would overwrite files that
    aren't managed by dex. Use --force to override this check.
    """
    from dex.core.installer import FileConflictError

    project = get_project(path)

    # Determine what to install
    plugin_specs: dict[str, str | PluginSpec] = {}

    if source:
        # Install from direct source
        # Try to determine the plugin name from the source
        if plugins and len(plugins) == 1:
            plugin_name = plugins[0].split("@")[0]
        else:
            # Load manifest to get name
            from dex.registry.factory import create_registry_client, normalize_source

            normalized = normalize_source(source)
            client = create_registry_client(normalized)
            packages = client.list_packages()
            if not packages:
                print_error(f"No plugin found at {source}")
                raise typer.Exit(1)
            plugin_name = packages[0]

        plugin_specs[plugin_name] = PluginSpec(source=source)

    elif plugins:
        # Parse plugin specifiers
        for plugin_arg in plugins:
            if "@" in plugin_arg:
                name, version = plugin_arg.rsplit("@", 1)
                spec = PluginSpec(version=version)
            else:
                spec = PluginSpec(version="latest")
                name = plugin_arg

            # Apply registry override if specified
            if registry:
                spec.registry = registry

            plugin_specs[name] = spec

    else:
        # Install from project config
        plugin_specs = dict(project.plugins)

    if not plugin_specs:
        console.print("No plugins to install")
        return

    # Run installation
    console.print(f"Installing {len(plugin_specs)} plugin(s)...")

    try:
        installer = PluginInstaller(project, force=force)
        summary = installer.install(
            plugin_specs=plugin_specs,
            use_lockfile=True,
            update_lockfile=not no_lock,
        )
    except FileConflictError as e:
        print_error(str(e))
        raise typer.Exit(1) from e

    # Print results
    for result in summary.results:
        if result.success:
            print_success(result.message)
            for warning in result.warnings:
                print_warning(f"  {warning}")
        else:
            print_error(f"Failed to install {result.plugin_name}: {result.message}")

    # Print environment variable warnings
    if summary.env_warnings:
        console.print()
        print_warning("Environment Variables Required:")
        for warning in summary.env_warnings:
            console.print(f"  {warning}")

    # Update project config if --save flag is used
    if save and (plugins or source):
        for plugin_name, plugin_spec in plugin_specs.items():
            if any(r.success and r.plugin_name == plugin_name for r in summary.results):
                project.add_plugin(plugin_name, plugin_spec)
        project.save()
        print_success("Saved to dex.yaml")

    if not summary.all_successful:
        raise typer.Exit(1)


@app.command("list")
def list_plugins(
    tree: Annotated[
        bool,
        typer.Option(
            "--tree",
            "-t",
            help="Show dependency tree",
        ),
    ] = False,
    path: Annotated[
        Path | None,
        typer.Option(
            "--path",
            "-p",
            help="Project directory",
        ),
    ] = None,
) -> None:
    """List installed plugins."""
    project = get_project(path)

    if not project.plugins:
        console.print("No plugins installed")
        return

    table = Table(title="Installed Plugins")
    table.add_column("Plugin", style="cyan")
    table.add_column("Version", style="green")
    table.add_column("Source", style="dim")

    for name, spec in project.plugins.items():
        if isinstance(spec, str):
            version = spec
            source = ""
        else:
            version = spec.version or "latest"
            source = spec.source or ""
            if spec.registry:
                source = f"registry:{spec.registry}"

        table.add_row(name, version, source)

    console.print(table)

    # Show lock file info
    from dex.core.lockfile import LockFileManager

    lock_manager = LockFileManager(project.root, project.agent)
    lock_manager.load()

    locked_count = len(lock_manager.list_locked())
    if locked_count > 0:
        console.print(f"\n{locked_count} plugin(s) locked in dex.lock")


@app.command()
def uninstall(
    plugins: Annotated[
        list[str],
        typer.Argument(help="Plugins to uninstall"),
    ],
    remove: Annotated[
        bool,
        typer.Option(
            "--remove",
            "-r",
            help="Also remove from dex.yaml (drop the dependency)",
        ),
    ] = False,
    path: Annotated[
        Path | None,
        typer.Option(
            "--path",
            "-p",
            help="Project directory",
        ),
    ] = None,
) -> None:
    """Uninstall plugins from the project.

    Deletes installed files tracked in the manifest and cleans up MCP servers
    that are no longer used.

    Use --remove to also remove the plugin from dex.yaml.
    """
    import json
    import shutil

    from dex.core.lockfile import LockFileManager
    from dex.core.manifest import ManifestManager

    project = get_project(path)
    adapter = get_adapter(project.agent)
    manifest_manager = ManifestManager(project.root)
    lock_manager = LockFileManager(project.root, project.agent)
    lock_manager.load()

    uninstalled = []
    removed_from_config = []

    for plugin_name in plugins:
        in_config = plugin_name in project.plugins
        plugin_files = manifest_manager.get_plugin_files(plugin_name)

        # Check if there's anything to uninstall
        if not in_config and not plugin_files:
            print_warning(f"Plugin not found: {plugin_name}")
            continue

        # Get MCP servers to remove (before removing from manifest)
        mcp_servers_to_remove = manifest_manager.get_mcp_servers_to_remove(plugin_name)

        # Remove tracked files
        if plugin_files:
            for rel_path in plugin_files.files:
                file_path = project.root / rel_path
                if file_path.exists():
                    file_path.unlink()

            # Remove tracked directories (in reverse order to handle nested dirs)
            for rel_path in sorted(plugin_files.directories, reverse=True):
                dir_path = project.root / rel_path
                if dir_path.exists() and dir_path.is_dir():
                    # Only remove if empty or only contains our managed files
                    with contextlib.suppress(OSError):
                        shutil.rmtree(dir_path)

            # Clean up empty parent directories
            base_dir = adapter.get_base_directory(project.root)
            for rel_path in sorted(plugin_files.directories, reverse=True):
                dir_path = project.root / rel_path
                parent = dir_path.parent
                # Walk up removing empty directories until we hit base_dir
                while parent != base_dir and parent != project.root:
                    if parent.exists() and parent.is_dir():
                        try:
                            if not any(parent.iterdir()):
                                parent.rmdir()
                        except OSError:
                            break
                    parent = parent.parent

            # Remove plugin from manifest
            manifest_manager.remove_plugin(plugin_name)
        else:
            # Fallback: remove skills directory if no manifest entry
            skills_dir = adapter.get_skills_directory(project.root) / plugin_name
            if skills_dir.exists():
                shutil.rmtree(skills_dir)

        # Remove MCP servers from config
        if mcp_servers_to_remove:
            mcp_config_path = adapter.get_mcp_config_path(project.root)
            if mcp_config_path is not None and mcp_config_path.exists():
                try:
                    with open(mcp_config_path, encoding="utf-8") as f:
                        mcp_config = json.load(f)

                    if "mcpServers" in mcp_config:
                        for server_name in mcp_servers_to_remove:
                            mcp_config["mcpServers"].pop(server_name, None)

                        # Write updated config
                        with open(mcp_config_path, "w", encoding="utf-8") as f:
                            json.dump(mcp_config, f, indent=2)
                            f.write("\n")
                except (json.JSONDecodeError, OSError):
                    pass

        # Remove from lock file
        lock_manager.unlock_plugin(plugin_name)

        uninstalled.append(plugin_name)
        print_success(f"Uninstalled {plugin_name}")

        # Only remove from dex.yaml if --remove flag is set
        if remove and in_config:
            project.remove_plugin(plugin_name)
            removed_from_config.append(plugin_name)
            print_success(f"Removed {plugin_name} from dex.yaml")

    if uninstalled:
        lock_manager.save()
        manifest_manager.save()

    if removed_from_config:
        project.save()


@app.command()
def info(
    plugin: Annotated[
        str,
        typer.Argument(help="Plugin name"),
    ],
    path: Annotated[
        Path | None,
        typer.Option(
            "--path",
            "-p",
            help="Project directory",
        ),
    ] = None,
) -> None:
    """Show information about an installed plugin."""
    project = get_project(path)

    if plugin not in project.plugins:
        print_error(f"Plugin not found: {plugin}")
        raise typer.Exit(1)

    spec = project.get_plugin_spec(plugin)
    if spec is None:
        print_error(f"Plugin spec not found: {plugin}")
        raise typer.Exit(1)

    console.print(f"[bold]{plugin}[/bold]")
    console.print()

    if spec.version:
        console.print(f"  Version: {spec.version}")
    if spec.source:
        console.print(f"  Source: {spec.source}")
    if spec.registry:
        console.print(f"  Registry: {spec.registry}")

    # Check lock file
    from dex.core.lockfile import LockFileManager

    lock_manager = LockFileManager(project.root, project.agent)
    lock_manager.load()
    locked = lock_manager.get_locked_plugin(plugin)

    if locked:
        console.print()
        console.print("  [dim]Locked:[/dim]")
        console.print(f"    Version: {locked.version}")
        console.print(f"    Resolved: {locked.resolved}")
        if locked.integrity:
            console.print(f"    Integrity: {locked.integrity[:32]}...")
        if locked.dependencies:
            console.print(f"    Dependencies: {', '.join(locked.dependencies.keys())}")


@app.command()
def manifest(
    plugin: Annotated[
        str | None,
        typer.Argument(help="Plugin name (shows all if not specified)"),
    ] = None,
    path: Annotated[
        Path | None,
        typer.Option(
            "--path",
            "-p",
            help="Project directory",
        ),
    ] = None,
) -> None:
    """Show files managed by Dex.

    Displays all files and directories being tracked by the manifest.
    Optionally filter by plugin name.
    """
    from dex.core.manifest import ManifestManager

    project = get_project(path)
    manifest_manager = ManifestManager(project.root)
    manifest_data = manifest_manager.load()

    if not manifest_data.plugins:
        console.print("No files managed by Dex")
        return

    # Filter by plugin if specified
    plugins_to_show = (
        {plugin: manifest_data.plugins[plugin]}
        if plugin and plugin in manifest_data.plugins
        else manifest_data.plugins
    )

    if plugin and plugin not in manifest_data.plugins:
        print_warning(f"Plugin not found in manifest: {plugin}")
        return

    # Display manifest
    for plugin_name, plugin_files in plugins_to_show.items():
        console.print(f"\n[bold cyan]{plugin_name}[/bold cyan]")

        if plugin_files.directories:
            console.print("  [dim]Directories:[/dim]")
            for dir_path in sorted(plugin_files.directories):
                console.print(f"    {dir_path}/")

        if plugin_files.files:
            console.print("  [dim]Files:[/dim]")
            for file_path in sorted(plugin_files.files):
                console.print(f"    {file_path}")

        if plugin_files.mcp_servers:
            console.print("  [dim]MCP Servers:[/dim]")
            for server_name in plugin_files.mcp_servers:
                console.print(f"    {server_name}")

    # Summary
    total_files = sum(len(p.files) for p in plugins_to_show.values())
    total_dirs = sum(len(p.directories) for p in plugins_to_show.values())
    total_mcp = sum(len(p.mcp_servers) for p in plugins_to_show.values())
    console.print()
    console.print(
        f"[dim]Total: {len(plugins_to_show)} plugin(s), "
        f"{total_files} file(s), {total_dirs} directory(ies), "
        f"{total_mcp} MCP server(s)[/dim]"
    )


# Gitignore management constants
GITIGNORE_HEADER = "# === Dex managed files (do not edit this section) ==="
GITIGNORE_FOOTER = "# === End Dex managed files ==="


@app.command("update-ignore")
def update_ignore(
    print_only: Annotated[
        bool,
        typer.Option(
            "--print",
            help="Print what would be written without modifying files",
        ),
    ] = False,
    path: Annotated[
        Path | None,
        typer.Option(
            "--path",
            "-p",
            help="Project directory",
        ),
    ] = None,
) -> None:
    """Update .gitignore with Dex-managed files.

    Adds or updates a managed section in .gitignore that excludes all
    directories and files tracked by Dex. The section is marked with
    header and footer comments so it can be parsed and updated.

    Use --print to see what would be written without modifying files.
    """
    from dex.core.manifest import ManifestManager

    project = get_project(path)
    manifest_manager = ManifestManager(project.root)
    manifest_data = manifest_manager.load()
    gitignore_path = project.root / ".gitignore"

    # Collect all top-level paths to ignore
    paths_to_ignore: set[str] = set()

    for plugin_files in manifest_data.plugins.values():
        # Add directories (only top-level)
        for dir_path in plugin_files.directories:
            # Get the top-level directory
            parts = dir_path.split("/")
            if parts:
                paths_to_ignore.add(parts[0] + "/")

        # Add files (only top-level or in top-level dirs)
        for file_path in plugin_files.files:
            parts = file_path.split("/")
            if parts:
                if len(parts) == 1:
                    # Top-level file
                    paths_to_ignore.add(parts[0])
                else:
                    # File in directory - add the directory
                    paths_to_ignore.add(parts[0] + "/")

    # Also add dex manifest directory
    paths_to_ignore.add(".dex/")

    # Generate the ignore section
    ignore_lines = [GITIGNORE_HEADER]
    for path_to_ignore in sorted(paths_to_ignore):
        ignore_lines.append(path_to_ignore)
    ignore_lines.append(GITIGNORE_FOOTER)
    ignore_section = "\n".join(ignore_lines)

    if print_only:
        console.print("[bold]Would add to .gitignore:[/bold]")
        console.print()
        for line in ignore_lines:
            console.print(line)
        return

    # Read existing gitignore content
    existing_content = ""
    if gitignore_path.exists():
        existing_content = gitignore_path.read_text()

    # Check if section already exists
    if GITIGNORE_HEADER in existing_content:
        # Replace existing section
        import re

        pattern = re.escape(GITIGNORE_HEADER) + r".*?" + re.escape(GITIGNORE_FOOTER)
        new_content = re.sub(pattern, ignore_section, existing_content, flags=re.DOTALL)
    else:
        # Append new section
        if existing_content and not existing_content.endswith("\n"):
            existing_content += "\n"
        if existing_content:
            existing_content += "\n"
        new_content = existing_content + ignore_section + "\n"

    # Write the file
    gitignore_path.write_text(new_content)
    print_success(f"Updated {gitignore_path}")

    # Show what was added
    console.print(f"\n[dim]Ignoring {len(paths_to_ignore)} path(s):[/dim]")
    for path_to_ignore in sorted(paths_to_ignore):
        console.print(f"  {path_to_ignore}")


if __name__ == "__main__":
    app()

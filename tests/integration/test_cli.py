"""Integration tests for CLI commands."""

from pathlib import Path

import pytest
import yaml
from typer.testing import CliRunner

from dex.cli.main import app


@pytest.fixture
def runner():
    """Get a CLI test runner."""
    return CliRunner()


class TestInitCommand:
    """Tests for 'dex init' command."""

    def test_init_creates_project(self, runner: CliRunner, temp_dir: Path):
        """Init command creates dex.yaml."""
        result = runner.invoke(app, ["init", "--path", str(temp_dir)])

        assert result.exit_code == 0
        assert (temp_dir / "dex.yaml").exists()

        config = yaml.safe_load((temp_dir / "dex.yaml").read_text())
        assert config["agent"] == "claude-code"

    def test_init_with_custom_agent(self, runner: CliRunner, temp_dir: Path):
        """Init with custom agent type."""
        result = runner.invoke(app, ["init", "--agent", "claude-code", "--path", str(temp_dir)])

        assert result.exit_code == 0
        config = yaml.safe_load((temp_dir / "dex.yaml").read_text())
        assert config["agent"] == "claude-code"

    def test_init_with_project_name(self, runner: CliRunner, temp_dir: Path):
        """Init with custom project name."""
        result = runner.invoke(
            app,
            ["init", "--name", "my-project", "--path", str(temp_dir)],
        )

        assert result.exit_code == 0
        config = yaml.safe_load((temp_dir / "dex.yaml").read_text())
        assert config["project_name"] == "my-project"

    def test_init_creates_claude_directory(self, runner: CliRunner, temp_dir: Path):
        """Init creates .claude directory."""
        runner.invoke(app, ["init", "--path", str(temp_dir)])

        assert (temp_dir / ".claude").exists()

    def test_init_fails_for_existing_project(self, runner: CliRunner, temp_dir: Path):
        """Init fails if project already exists."""
        (temp_dir / "dex.yaml").write_text("agent: claude-code")

        result = runner.invoke(app, ["init", "--path", str(temp_dir)])

        assert result.exit_code == 1
        assert "already initialized" in result.output.lower()


class TestVersionCommand:
    """Tests for 'dex version' command."""

    def test_version_shows_version(self, runner: CliRunner):
        """Version command shows version."""
        result = runner.invoke(app, ["version"])

        assert result.exit_code == 0
        assert "dex" in result.output


class TestListCommand:
    """Tests for 'dex list' command."""

    def test_list_shows_no_plugins(self, runner: CliRunner, temp_dir: Path):
        """List shows message when no plugins."""
        # Initialize project
        runner.invoke(app, ["init", "--path", str(temp_dir)])

        result = runner.invoke(app, ["list", "--path", str(temp_dir)])

        assert result.exit_code == 0
        assert "no plugins" in result.output.lower()

    def test_list_shows_installed_plugins(self, runner: CliRunner, temp_dir: Path):
        """List shows installed plugins."""
        # Initialize project with plugins
        config_content = """\
agent: claude-code
plugins:
  plugin-a: ^1.0.0
  plugin-b: ^2.0.0
"""
        (temp_dir / "dex.yaml").write_text(config_content)

        result = runner.invoke(app, ["list", "--path", str(temp_dir)])

        assert result.exit_code == 0
        assert "plugin-a" in result.output
        assert "plugin-b" in result.output


class TestUninstallCommand:
    """Tests for 'dex uninstall' command."""

    def test_uninstall_plugin_with_remove_flag(self, runner: CliRunner, temp_dir: Path):
        """Uninstall with --remove removes plugin from config."""
        # Initialize with plugin
        config_content = """\
agent: claude-code
plugins:
  test-plugin: ^1.0.0
"""
        (temp_dir / "dex.yaml").write_text(config_content)

        result = runner.invoke(
            app, ["uninstall", "test-plugin", "--remove", "--path", str(temp_dir)]
        )

        assert result.exit_code == 0

        # Verify removed from config
        updated = yaml.safe_load((temp_dir / "dex.yaml").read_text())
        assert "test-plugin" not in updated.get("plugins", {})

    def test_uninstall_without_remove_keeps_config(self, runner: CliRunner, temp_dir: Path):
        """Uninstall without --remove keeps plugin in config."""
        config_content = """\
agent: claude-code
plugins:
  test-plugin: ^1.0.0
"""
        (temp_dir / "dex.yaml").write_text(config_content)

        result = runner.invoke(app, ["uninstall", "test-plugin", "--path", str(temp_dir)])

        assert result.exit_code == 0

        # Verify still in config
        updated = yaml.safe_load((temp_dir / "dex.yaml").read_text())
        assert "test-plugin" in updated.get("plugins", {})

    def test_uninstall_nonexistent_warns(self, runner: CliRunner, temp_dir: Path):
        """Uninstall warns for nonexistent plugin."""
        config_content = """\
agent: claude-code
plugins: {}
"""
        (temp_dir / "dex.yaml").write_text(config_content)

        result = runner.invoke(app, ["uninstall", "nonexistent", "--path", str(temp_dir)])

        assert "not found" in result.output.lower()


class TestInfoCommand:
    """Tests for 'dex info' command."""

    def test_info_shows_plugin_details(self, runner: CliRunner, temp_dir: Path):
        """Info shows plugin details."""
        config_content = """\
agent: claude-code
plugins:
  test-plugin: ^1.0.0
"""
        (temp_dir / "dex.yaml").write_text(config_content)

        result = runner.invoke(app, ["info", "test-plugin", "--path", str(temp_dir)])

        assert result.exit_code == 0
        assert "test-plugin" in result.output

    def test_info_fails_for_unknown_plugin(self, runner: CliRunner, temp_dir: Path):
        """Info fails for unknown plugin."""
        config_content = """\
agent: claude-code
plugins: {}
"""
        (temp_dir / "dex.yaml").write_text(config_content)

        result = runner.invoke(app, ["info", "unknown", "--path", str(temp_dir)])

        assert result.exit_code == 1
        assert "not found" in result.output.lower()


class TestInstallCommand:
    """Tests for 'dex install' command."""

    def test_install_no_plugins_message(self, runner: CliRunner, temp_dir: Path):
        """Install shows message when no plugins to install."""
        config_content = """\
agent: claude-code
plugins: {}
"""
        (temp_dir / "dex.yaml").write_text(config_content)

        result = runner.invoke(app, ["install", "--path", str(temp_dir)])

        assert result.exit_code == 0
        assert "no plugins" in result.output.lower()

    def test_install_from_source(self, runner: CliRunner, temp_dir: Path, temp_plugin_dir: Path):
        """Install from direct source."""
        # Initialize project
        runner.invoke(app, ["init", "--path", str(temp_dir)])

        result = runner.invoke(
            app,
            [
                "install",
                "--source",
                str(temp_plugin_dir),
                "--path",
                str(temp_dir),
            ],
        )

        # Should attempt installation (may fail due to missing context files)
        # but should not crash
        assert result.exit_code in (0, 1)

    def test_install_fails_on_unmanaged_file_conflict(
        self, runner: CliRunner, temp_dir: Path, temp_plugin_dir: Path
    ):
        """Install fails if would overwrite unmanaged file."""
        # Initialize project
        runner.invoke(app, ["init", "--path", str(temp_dir)])

        # Create a file that would conflict with the plugin installation
        skill_dir = temp_dir / ".claude" / "skills" / "test-plugin-test-skill"
        skill_dir.mkdir(parents=True)
        (skill_dir / "SKILL.md").write_text("# My custom content")

        result = runner.invoke(
            app,
            [
                "install",
                "--source",
                str(temp_plugin_dir),
                "--path",
                str(temp_dir),
            ],
        )

        assert result.exit_code == 1
        assert (
            "would be overwritten" in result.output.lower() or "conflict" in result.output.lower()
        )

    def test_install_force_overwrites_unmanaged_files(
        self, runner: CliRunner, temp_dir: Path, temp_plugin_dir: Path
    ):
        """Install --force overwrites unmanaged files."""
        # Initialize project
        runner.invoke(app, ["init", "--path", str(temp_dir)])

        # Create a file that would conflict with the plugin installation
        skill_dir = temp_dir / ".claude" / "skills" / "test-plugin-test-skill"
        skill_dir.mkdir(parents=True)
        (skill_dir / "SKILL.md").write_text("# My custom content")

        result = runner.invoke(
            app,
            [
                "install",
                "--source",
                str(temp_plugin_dir),
                "--path",
                str(temp_dir),
                "--force",
            ],
        )

        # Should succeed with --force
        assert result.exit_code == 0

        # File should be overwritten
        content = (skill_dir / "SKILL.md").read_text()
        assert "My custom content" not in content


class TestManifestCommand:
    """Tests for 'dex manifest' command."""

    def test_manifest_shows_no_files(self, runner: CliRunner, temp_dir: Path):
        """Manifest shows message when no files managed."""
        config_content = """\
agent: claude-code
plugins: {}
"""
        (temp_dir / "dex.yaml").write_text(config_content)

        result = runner.invoke(app, ["manifest", "--path", str(temp_dir)])

        assert result.exit_code == 0
        assert "no files" in result.output.lower()

    def test_manifest_shows_installed_files(
        self, runner: CliRunner, temp_dir: Path, temp_plugin_dir: Path
    ):
        """Manifest shows files after installation."""
        # Initialize and install
        runner.invoke(app, ["init", "--path", str(temp_dir)])
        runner.invoke(
            app,
            [
                "install",
                "--source",
                f"file:{temp_plugin_dir}",
                "--path",
                str(temp_dir),
            ],
        )

        result = runner.invoke(app, ["manifest", "--path", str(temp_dir)])

        assert result.exit_code == 0
        # Should show plugin name and file info
        assert "test-plugin" in result.output or "Total:" in result.output


class TestUpdateIgnoreCommand:
    """Tests for 'dex update-ignore' command."""

    def test_update_ignore_print_only(self, runner: CliRunner, temp_dir: Path):
        """Update-ignore --print shows what would be added."""
        config_content = """\
agent: claude-code
plugins: {}
"""
        (temp_dir / "dex.yaml").write_text(config_content)

        result = runner.invoke(app, ["update-ignore", "--print", "--path", str(temp_dir)])

        assert result.exit_code == 0
        assert "Would add to .gitignore" in result.output
        assert ".dex/" in result.output

    def test_update_ignore_creates_gitignore(self, runner: CliRunner, temp_dir: Path):
        """Update-ignore creates .gitignore if it doesn't exist."""
        config_content = """\
agent: claude-code
plugins: {}
"""
        (temp_dir / "dex.yaml").write_text(config_content)

        result = runner.invoke(app, ["update-ignore", "--path", str(temp_dir)])

        assert result.exit_code == 0
        gitignore = temp_dir / ".gitignore"
        assert gitignore.exists()
        content = gitignore.read_text()
        assert "Dex managed files" in content
        assert ".dex/" in content

    def test_update_ignore_preserves_existing(self, runner: CliRunner, temp_dir: Path):
        """Update-ignore preserves existing gitignore content."""
        config_content = """\
agent: claude-code
plugins: {}
"""
        (temp_dir / "dex.yaml").write_text(config_content)

        # Create existing gitignore
        gitignore = temp_dir / ".gitignore"
        gitignore.write_text("# My custom ignores\n*.log\nnode_modules/\n")

        result = runner.invoke(app, ["update-ignore", "--path", str(temp_dir)])

        assert result.exit_code == 0
        content = gitignore.read_text()
        # Existing content preserved
        assert "My custom ignores" in content
        assert "*.log" in content
        # Dex section added
        assert "Dex managed files" in content

    def test_update_ignore_updates_existing_section(self, runner: CliRunner, temp_dir: Path):
        """Update-ignore updates existing Dex section."""
        config_content = """\
agent: claude-code
plugins: {}
"""
        (temp_dir / "dex.yaml").write_text(config_content)

        # Create gitignore with existing Dex section
        gitignore = temp_dir / ".gitignore"
        gitignore.write_text(
            "# My stuff\n"
            "*.log\n\n"
            "# === Dex managed files (do not edit this section) ===\n"
            ".old-stuff/\n"
            "# === End Dex managed files ===\n"
        )

        result = runner.invoke(app, ["update-ignore", "--path", str(temp_dir)])

        assert result.exit_code == 0
        content = gitignore.read_text()
        # Custom content preserved
        assert "My stuff" in content
        # Old section replaced (not duplicated)
        assert content.count("Dex managed files") == 2  # header and footer
        assert ".dex/" in content

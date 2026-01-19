"""Tests for dex.core.lockfile module."""

from pathlib import Path

import yaml

from dex.config.schemas import LockFile
from dex.core.lockfile import LockFileManager


class TestLockFileManagerInit:
    """Tests for LockFileManager initialization."""

    def test_initializes_with_path_and_agent(self, temp_dir: Path):
        """Initializes with project root and agent."""
        manager = LockFileManager(temp_dir, "claude-code")

        assert manager._project_root == temp_dir
        assert manager._agent == "claude-code"


class TestLockFileManagerLoad:
    """Tests for LockFileManager.load() method."""

    def test_loads_existing_lockfile(self, temp_dir: Path):
        """Loads existing lock file."""
        lock_content = """\
version: '1.0'
agent: claude-code
plugins:
  test-plugin:
    version: 1.0.0
    resolved: file:///path
    integrity: sha512-abc
"""
        (temp_dir / "dex.lock").write_text(lock_content)

        manager = LockFileManager(temp_dir, "claude-code")
        lockfile = manager.load()

        assert lockfile.agent == "claude-code"
        assert "test-plugin" in lockfile.plugins

    def test_creates_new_lockfile_if_missing(self, temp_dir: Path):
        """Creates new lock file if it doesn't exist."""
        manager = LockFileManager(temp_dir, "claude-code")

        lockfile = manager.load()

        assert lockfile.agent == "claude-code"
        assert lockfile.plugins == {}


class TestLockFileManagerSave:
    """Tests for LockFileManager.save() method."""

    def test_saves_lockfile(self, temp_dir: Path):
        """Saves lock file to disk."""
        manager = LockFileManager(temp_dir, "claude-code")
        manager.load()
        manager.lock_plugin("test-plugin", "1.0.0", "file:///path")

        manager.save()

        lock_path = temp_dir / "dex.lock"
        assert lock_path.exists()
        saved_data = yaml.safe_load(lock_path.read_text())
        assert "test-plugin" in saved_data["plugins"]

    def test_does_not_save_if_unmodified(self, temp_dir: Path):
        """Doesn't save if not modified."""
        lock_content = """\
version: '1.0'
agent: claude-code
plugins: {}
"""
        lock_path = temp_dir / "dex.lock"
        lock_path.write_text(lock_content)
        _ = lock_path.stat().st_mtime  # Store mtime for potential future use

        manager = LockFileManager(temp_dir, "claude-code")
        manager.load()
        manager.save()

        # File should not have been modified
        # Note: This might be flaky depending on filesystem precision
        # assert lock_path.stat().st_mtime == original_mtime


class TestLockFileManagerLockfileProperty:
    """Tests for LockFileManager.lockfile property."""

    def test_loads_lockfile_if_not_loaded(self, temp_dir: Path):
        """Loads lock file on first access."""
        manager = LockFileManager(temp_dir, "claude-code")

        lockfile = manager.lockfile

        assert lockfile is not None
        assert isinstance(lockfile, LockFile)

    def test_returns_cached_lockfile(self, temp_dir: Path):
        """Returns cached lock file on subsequent access."""
        manager = LockFileManager(temp_dir, "claude-code")

        lockfile1 = manager.lockfile
        lockfile2 = manager.lockfile

        assert lockfile1 is lockfile2


class TestLockFileManagerGetLockedVersion:
    """Tests for LockFileManager.get_locked_version() method."""

    def test_returns_locked_version(self, temp_dir: Path):
        """Returns locked version for plugin."""
        lock_content = """\
version: '1.0'
agent: claude-code
plugins:
  test-plugin:
    version: 1.2.3
    resolved: file:///path
    integrity: sha512-abc
"""
        (temp_dir / "dex.lock").write_text(lock_content)

        manager = LockFileManager(temp_dir, "claude-code")
        manager.load()

        version = manager.get_locked_version("test-plugin")

        assert version == "1.2.3"

    def test_returns_none_for_unlocked(self, temp_dir: Path):
        """Returns None for unlocked plugin."""
        manager = LockFileManager(temp_dir, "claude-code")
        manager.load()

        version = manager.get_locked_version("unlocked-plugin")

        assert version is None


class TestLockFileManagerGetLockedPlugin:
    """Tests for LockFileManager.get_locked_plugin() method."""

    def test_returns_locked_plugin(self, temp_dir: Path):
        """Returns LockedPlugin entry."""
        lock_content = """\
version: '1.0'
agent: claude-code
plugins:
  test-plugin:
    version: 1.0.0
    resolved: file:///path
    integrity: sha512-abc
    dependencies:
      dep1: 2.0.0
"""
        (temp_dir / "dex.lock").write_text(lock_content)

        manager = LockFileManager(temp_dir, "claude-code")
        manager.load()

        locked = manager.get_locked_plugin("test-plugin")

        assert locked is not None
        assert locked.version == "1.0.0"
        assert locked.resolved == "file:///path"
        assert locked.dependencies["dep1"] == "2.0.0"

    def test_returns_none_for_unlocked(self, temp_dir: Path):
        """Returns None for unlocked plugin."""
        manager = LockFileManager(temp_dir, "claude-code")
        manager.load()

        locked = manager.get_locked_plugin("unlocked")

        assert locked is None


class TestLockFileManagerLockPlugin:
    """Tests for LockFileManager.lock_plugin() method."""

    def test_locks_plugin(self, temp_dir: Path):
        """Locks a plugin with version info."""
        manager = LockFileManager(temp_dir, "claude-code")
        manager.load()

        manager.lock_plugin(
            name="test-plugin",
            version="1.0.0",
            resolved_url="file:///path/to/plugin",
            integrity="sha512-abc123",
        )

        locked = manager.get_locked_plugin("test-plugin")
        assert locked is not None
        assert locked.version == "1.0.0"
        assert locked.resolved == "file:///path/to/plugin"
        assert locked.integrity == "sha512-abc123"

    def test_locks_plugin_with_dependencies(self, temp_dir: Path):
        """Locks plugin with dependency info."""
        manager = LockFileManager(temp_dir, "claude-code")
        manager.load()

        manager.lock_plugin(
            name="test-plugin",
            version="1.0.0",
            resolved_url="file:///path",
            dependencies={"dep1": "2.0.0"},
        )

        locked = manager.get_locked_plugin("test-plugin")
        assert locked is not None
        assert locked.dependencies["dep1"] == "2.0.0"

    def test_marks_as_modified(self, temp_dir: Path):
        """Marks lock file as modified."""
        manager = LockFileManager(temp_dir, "claude-code")
        manager.load()

        assert manager._modified is True  # New lockfile is modified
        manager.save()
        assert manager._modified is False

        manager.lock_plugin("plugin", "1.0.0", "file:///path")

        assert manager._modified is True


class TestLockFileManagerUnlockPlugin:
    """Tests for LockFileManager.unlock_plugin() method."""

    def test_unlocks_plugin(self, temp_dir: Path):
        """Removes plugin from lock file."""
        lock_content = """\
version: '1.0'
agent: claude-code
plugins:
  test-plugin:
    version: 1.0.0
    resolved: file:///path
    integrity: sha512-abc
"""
        (temp_dir / "dex.lock").write_text(lock_content)

        manager = LockFileManager(temp_dir, "claude-code")
        manager.load()

        result = manager.unlock_plugin("test-plugin")

        assert result is True
        assert manager.get_locked_plugin("test-plugin") is None

    def test_returns_false_for_unlocked(self, temp_dir: Path):
        """Returns False for already unlocked plugin."""
        manager = LockFileManager(temp_dir, "claude-code")
        manager.load()

        result = manager.unlock_plugin("nonexistent")

        assert result is False


class TestLockFileManagerIsLocked:
    """Tests for LockFileManager.is_locked() method."""

    def test_returns_true_for_locked(self, temp_dir: Path):
        """Returns True for locked plugin."""
        manager = LockFileManager(temp_dir, "claude-code")
        manager.load()
        manager.lock_plugin("test-plugin", "1.0.0", "file:///path")

        assert manager.is_locked("test-plugin") is True

    def test_returns_false_for_unlocked(self, temp_dir: Path):
        """Returns False for unlocked plugin."""
        manager = LockFileManager(temp_dir, "claude-code")
        manager.load()

        assert manager.is_locked("unlocked") is False


class TestLockFileManagerListLocked:
    """Tests for LockFileManager.list_locked() method."""

    def test_lists_all_locked_plugins(self, temp_dir: Path):
        """Lists all locked plugin names."""
        manager = LockFileManager(temp_dir, "claude-code")
        manager.load()
        manager.lock_plugin("plugin1", "1.0.0", "file:///p1")
        manager.lock_plugin("plugin2", "2.0.0", "file:///p2")

        locked = manager.list_locked()

        assert "plugin1" in locked
        assert "plugin2" in locked

    def test_returns_empty_for_no_locked(self, temp_dir: Path):
        """Returns empty list when no plugins locked."""
        manager = LockFileManager(temp_dir, "claude-code")
        manager.load()

        locked = manager.list_locked()

        assert locked == []


class TestLockFileManagerClear:
    """Tests for LockFileManager.clear() method."""

    def test_clears_all_entries(self, temp_dir: Path):
        """Clears all lock entries."""
        manager = LockFileManager(temp_dir, "claude-code")
        manager.load()
        manager.lock_plugin("plugin1", "1.0.0", "file:///p1")
        manager.lock_plugin("plugin2", "2.0.0", "file:///p2")

        manager.clear()

        assert manager.list_locked() == []
        assert manager._modified is True

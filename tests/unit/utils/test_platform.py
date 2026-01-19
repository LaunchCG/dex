"""Tests for dex.utils.platform module."""

import os
from unittest.mock import patch

from dex.utils.platform import (
    get_arch,
    get_env,
    get_home_directory,
    get_os,
    get_platform_info,
    get_python_version,
    is_unix,
    is_windows,
    matches_platform,
)


class TestGetOS:
    """Tests for get_os function."""

    def test_returns_valid_os(self):
        """Returns one of the valid OS values."""
        result = get_os()
        assert result in ("windows", "linux", "macos")

    @patch("platform.system")
    def test_darwin_returns_macos(self, mock_system):
        """Darwin platform returns macos."""
        mock_system.return_value = "Darwin"
        assert get_os() == "macos"

    @patch("platform.system")
    def test_windows_returns_windows(self, mock_system):
        """Windows platform returns windows."""
        mock_system.return_value = "Windows"
        assert get_os() == "windows"

    @patch("platform.system")
    def test_linux_returns_linux(self, mock_system):
        """Linux platform returns linux."""
        mock_system.return_value = "Linux"
        assert get_os() == "linux"

    @patch("platform.system")
    def test_unknown_returns_linux(self, mock_system):
        """Unknown platform defaults to linux."""
        mock_system.return_value = "FreeBSD"
        assert get_os() == "linux"


class TestGetArch:
    """Tests for get_arch function."""

    def test_returns_valid_arch(self):
        """Returns one of the valid architecture values."""
        result = get_arch()
        assert result in ("x64", "arm64", "arm", "x86")

    @patch("platform.machine")
    def test_x86_64_returns_x64(self, mock_machine):
        """x86_64 returns x64."""
        mock_machine.return_value = "x86_64"
        assert get_arch() == "x64"

    @patch("platform.machine")
    def test_amd64_returns_x64(self, mock_machine):
        """AMD64 returns x64."""
        mock_machine.return_value = "AMD64"
        assert get_arch() == "x64"

    @patch("platform.machine")
    def test_aarch64_returns_arm64(self, mock_machine):
        """aarch64 returns arm64."""
        mock_machine.return_value = "aarch64"
        assert get_arch() == "arm64"

    @patch("platform.machine")
    def test_arm64_returns_arm64(self, mock_machine):
        """arm64 returns arm64."""
        mock_machine.return_value = "arm64"
        assert get_arch() == "arm64"

    @patch("platform.machine")
    def test_armv7_returns_arm(self, mock_machine):
        """ARMv7 returns arm."""
        mock_machine.return_value = "armv7l"
        assert get_arch() == "arm"

    @patch("platform.machine")
    def test_i686_returns_x86(self, mock_machine):
        """i686 returns x86."""
        mock_machine.return_value = "i686"
        assert get_arch() == "x86"

    @patch("platform.machine")
    def test_unknown_defaults_to_x64(self, mock_machine):
        """Unknown architecture defaults to x64."""
        mock_machine.return_value = "unknown_arch"
        assert get_arch() == "x64"


class TestIsUnix:
    """Tests for is_unix function."""

    @patch("dex.utils.platform.get_os")
    def test_linux_is_unix(self, mock_get_os):
        """Linux is Unix-like."""
        mock_get_os.return_value = "linux"
        assert is_unix() is True

    @patch("dex.utils.platform.get_os")
    def test_macos_is_unix(self, mock_get_os):
        """macOS is Unix-like."""
        mock_get_os.return_value = "macos"
        assert is_unix() is True

    @patch("dex.utils.platform.get_os")
    def test_windows_is_not_unix(self, mock_get_os):
        """Windows is not Unix-like."""
        mock_get_os.return_value = "windows"
        assert is_unix() is False


class TestIsWindows:
    """Tests for is_windows function."""

    @patch("dex.utils.platform.get_os")
    def test_windows_is_windows(self, mock_get_os):
        """Windows returns True."""
        mock_get_os.return_value = "windows"
        assert is_windows() is True

    @patch("dex.utils.platform.get_os")
    def test_linux_is_not_windows(self, mock_get_os):
        """Linux returns False."""
        mock_get_os.return_value = "linux"
        assert is_windows() is False

    @patch("dex.utils.platform.get_os")
    def test_macos_is_not_windows(self, mock_get_os):
        """macOS returns False."""
        mock_get_os.return_value = "macos"
        assert is_windows() is False


class TestGetHomeDirectory:
    """Tests for get_home_directory function."""

    def test_returns_string(self):
        """Returns a string path."""
        result = get_home_directory()
        assert isinstance(result, str)
        assert len(result) > 0

    def test_returns_existing_path(self):
        """Returns a path that exists."""
        from pathlib import Path

        result = get_home_directory()
        assert Path(result).exists()


class TestGetEnv:
    """Tests for get_env function."""

    def test_returns_existing_env_var(self):
        """Returns value of existing environment variable."""
        with patch.dict(os.environ, {"TEST_VAR": "test_value"}):
            assert get_env("TEST_VAR") == "test_value"

    def test_returns_default_for_missing(self):
        """Returns default for missing environment variable."""
        result = get_env("NONEXISTENT_VAR_12345", "default_value")
        assert result == "default_value"

    def test_returns_none_for_missing_without_default(self):
        """Returns None for missing env var without default."""
        result = get_env("NONEXISTENT_VAR_12345")
        assert result is None


class TestGetPythonVersion:
    """Tests for get_python_version function."""

    def test_returns_version_string(self):
        """Returns a version string."""
        result = get_python_version()
        assert isinstance(result, str)
        # Should be in format X.Y.Z
        parts = result.split(".")
        assert len(parts) >= 2
        assert all(part.isdigit() for part in parts[:2])


class TestGetPlatformInfo:
    """Tests for get_platform_info function."""

    def test_returns_dict(self):
        """Returns a dictionary."""
        result = get_platform_info()
        assert isinstance(result, dict)

    def test_contains_expected_keys(self):
        """Contains all expected keys."""
        result = get_platform_info()
        expected_keys = ["os", "arch", "python_version", "platform", "machine", "system"]
        for key in expected_keys:
            assert key in result, f"Missing key: {key}"

    def test_os_is_valid(self):
        """OS value is valid."""
        result = get_platform_info()
        assert result["os"] in ("windows", "linux", "macos")

    def test_arch_is_valid(self):
        """Architecture value is valid."""
        result = get_platform_info()
        assert result["arch"] in ("x64", "arm64", "arm", "x86")


class TestMatchesPlatform:
    """Tests for matches_platform function."""

    @patch("dex.utils.platform.get_os")
    def test_matches_current_os(self, mock_get_os):
        """Matches current OS."""
        mock_get_os.return_value = "linux"
        assert matches_platform("linux") is True

    @patch("dex.utils.platform.get_os")
    def test_does_not_match_different_os(self, mock_get_os):
        """Does not match different OS."""
        mock_get_os.return_value = "linux"
        assert matches_platform("windows") is False

    @patch("dex.utils.platform.get_os")
    def test_unix_matches_linux(self, mock_get_os):
        """Unix target matches Linux."""
        mock_get_os.return_value = "linux"
        assert matches_platform("unix") is True

    @patch("dex.utils.platform.get_os")
    def test_unix_matches_macos(self, mock_get_os):
        """Unix target matches macOS."""
        mock_get_os.return_value = "macos"
        assert matches_platform("unix") is True

    @patch("dex.utils.platform.get_os")
    def test_unix_does_not_match_windows(self, mock_get_os):
        """Unix target does not match Windows."""
        mock_get_os.return_value = "windows"
        assert matches_platform("unix") is False

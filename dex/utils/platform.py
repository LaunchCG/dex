"""Platform and OS detection utilities."""

import os
import platform
from typing import Literal

PlatformOS = Literal["windows", "linux", "macos"]
PlatformArch = Literal["x64", "arm64", "arm", "x86"]


def get_os() -> PlatformOS:
    """Get the current operating system.

    Returns:
        One of: "windows", "linux", "macos"
    """
    system = platform.system().lower()
    if system == "darwin":
        return "macos"
    elif system == "windows":
        return "windows"
    else:
        return "linux"


def get_arch() -> PlatformArch:
    """Get the current CPU architecture.

    Returns:
        One of: "x64", "arm64", "arm", "x86"
    """
    machine = platform.machine().lower()

    if machine in ("x86_64", "amd64"):
        return "x64"
    elif machine in ("aarch64", "arm64"):
        return "arm64"
    elif machine.startswith("arm"):
        return "arm"
    elif machine in ("i386", "i686", "x86"):
        return "x86"
    else:
        # Default to x64 for unknown architectures
        return "x64"


def is_unix() -> bool:
    """Check if the current OS is Unix-like (Linux or macOS).

    Returns:
        True if running on Linux or macOS
    """
    return get_os() in ("linux", "macos")


def is_windows() -> bool:
    """Check if the current OS is Windows.

    Returns:
        True if running on Windows
    """
    return get_os() == "windows"


def get_home_directory() -> str:
    """Get the user's home directory.

    Returns:
        Path to the home directory
    """
    return os.path.expanduser("~")


def get_env(name: str, default: str | None = None) -> str | None:
    """Get an environment variable.

    Args:
        name: Environment variable name
        default: Default value if not set

    Returns:
        Environment variable value or default
    """
    return os.environ.get(name, default)


def get_python_version() -> str:
    """Get the Python version string.

    Returns:
        Python version (e.g., "3.11.4")
    """
    return platform.python_version()


def get_platform_info() -> dict[str, str]:
    """Get comprehensive platform information.

    Returns:
        Dictionary with platform details
    """
    return {
        "os": get_os(),
        "arch": get_arch(),
        "python_version": get_python_version(),
        "platform": platform.platform(),
        "machine": platform.machine(),
        "system": platform.system(),
    }


def matches_platform(target: str) -> bool:
    """Check if the current platform matches a target specification.

    Args:
        target: Platform target ("windows", "linux", "macos", "unix")

    Returns:
        True if the current platform matches
    """
    if target == "unix":
        return is_unix()
    return target == get_os()

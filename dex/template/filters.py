"""Custom Jinja2 filters for Dex templates."""

import os


def basename(path: str) -> str:
    """Get the basename (filename) from a path.

    Example: "/path/to/file.txt" -> "file.txt"
    """
    return os.path.basename(path)


def dirname(path: str) -> str:
    """Get the directory name from a path.

    Example: "/path/to/file.txt" -> "/path/to"
    """
    return os.path.dirname(path)


def abspath(path: str) -> str:
    """Convert a path to absolute path.

    Example: "./file.txt" -> "/current/dir/file.txt"
    """
    return os.path.abspath(path)


def normpath(path: str) -> str:
    """Normalize a path (resolve .., ., and normalize separators).

    Example: "/path/to/../file.txt" -> "/path/file.txt"
    """
    return os.path.normpath(path)


def joinpath(*parts: str) -> str:
    """Join path components.

    Example: joinpath("/path", "to", "file.txt") -> "/path/to/file.txt"
    """
    return os.path.join(*parts)


def splitext(path: str) -> tuple[str, str]:
    """Split a path into root and extension.

    Example: "/path/to/file.txt" -> ("/path/to/file", ".txt")
    """
    return os.path.splitext(path)


def extension(path: str) -> str:
    """Get the file extension from a path.

    Example: "/path/to/file.txt" -> ".txt"
    """
    return os.path.splitext(path)[1]


def to_posix(path: str) -> str:
    """Convert a path to POSIX format (forward slashes).

    Example: "path\\to\\file" -> "path/to/file"
    """
    # Use replace() instead of Path.as_posix() because as_posix()
    # doesn't convert backslashes on Unix (backslash is a valid
    # filename character there).
    return path.replace("\\", "/")


def default_value(value: str | None, default: str) -> str:
    """Return value if truthy, otherwise return default.

    This is an enhanced version of Jinja2's built-in default filter
    that also handles empty strings.
    """
    return value if value else default


# Dictionary of all custom filters to register
CUSTOM_FILTERS = {
    "basename": basename,
    "dirname": dirname,
    "abspath": abspath,
    "normpath": normpath,
    "joinpath": joinpath,
    "splitext": splitext,
    "extension": extension,
    "to_posix": to_posix,
    "default_value": default_value,
}

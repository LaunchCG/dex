"""Registry client factory."""

from urllib.parse import urlparse

from dex.registry.base import RegistryClient
from dex.registry.local import LocalRegistryClient


class UnsupportedProtocolError(Exception):
    """Error when a registry URL uses an unsupported protocol."""

    def __init__(self, protocol: str, url: str):
        self.protocol = protocol
        self.url = url
        super().__init__(f"Unsupported registry protocol: {protocol} (in {url})")


def create_registry_client(url: str) -> RegistryClient:
    """Create a registry client for the given URL.

    Args:
        url: Registry URL (file://, https://, s3://, etc.)

    Returns:
        Appropriate RegistryClient instance

    Raises:
        UnsupportedProtocolError: If the protocol is not supported
    """
    # Handle file: URLs (both file:// and file:)
    if url.startswith("file:"):
        return LocalRegistryClient(url)

    # Parse URL to determine protocol
    parsed = urlparse(url)
    protocol = parsed.scheme.lower()

    if protocol in ("file", ""):
        # Treat as local path
        return LocalRegistryClient(url)
    elif protocol == "https":
        # HTTPS registry (not implemented yet)
        raise UnsupportedProtocolError(protocol, url)
    elif protocol in ("http", "s3", "az", "git"):
        raise UnsupportedProtocolError(protocol, url)
    else:
        raise UnsupportedProtocolError(protocol, url)


def is_local_source(source: str) -> bool:
    """Check if a source string is a local file source.

    Args:
        source: Source URL or path

    Returns:
        True if the source is local (file:// or relative path)
    """
    if source.startswith("file:"):
        return True
    if source.startswith(("./", "../", "/")):
        return True
    # Check if it's a Windows absolute path
    return len(source) > 2 and source[1] == ":" and source[2] in ("/", "\\")


def normalize_source(source: str) -> str:
    """Normalize a source string to a standard URL format.

    Args:
        source: Source URL or path

    Returns:
        Normalized URL string
    """
    if source.startswith("file:"):
        return source
    if source.startswith(("./", "../")):
        return f"file:{source}"
    if source.startswith("/"):
        return f"file://{source}"
    # Windows absolute path
    if len(source) > 2 and source[1] == ":" and source[2] in ("/", "\\"):
        return f"file:///{source}"
    return source

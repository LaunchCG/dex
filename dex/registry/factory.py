"""Registry client factory."""

import logging
from pathlib import Path
from urllib.parse import urlparse

from dex.registry.base import RegistryClient
from dex.registry.common import SourceMode
from dex.registry.local import LocalRegistryClient

logger = logging.getLogger(__name__)


class UnsupportedProtocolError(Exception):
    """Error when a registry URL uses an unsupported protocol."""

    def __init__(self, protocol: str, url: str):
        self.protocol = protocol
        self.url = url
        super().__init__(f"Unsupported registry protocol: {protocol} (in {url})")


def create_registry_client(
    url: str,
    mode: SourceMode = "registry",
    cache_dir: Path | None = None,
) -> RegistryClient:
    """Create a registry client for the given URL.

    Args:
        url: Registry URL (file://, https://, s3://, git+https://, etc.)
        mode: Source mode - "registry" expects registry.json,
              "package" expects package.json
        cache_dir: Optional directory for caching remote downloads

    Returns:
        Appropriate RegistryClient instance

    Raises:
        UnsupportedProtocolError: If the protocol is not supported
    """
    logger.debug("Creating registry client for URL: %s (mode=%s)", url, mode)

    # Handle git+ URLs (git+https://, git+ssh://)
    if is_git_source(url):
        from dex.registry.git import GitRegistryClient

        logger.info("Creating Git registry client for %s", url)
        return GitRegistryClient(url, mode=mode, cache_dir=cache_dir)

    # Handle file: URLs (both file:// and file:)
    if url.startswith("file:"):
        logger.info("Creating local registry client for %s", url)
        return LocalRegistryClient(url, mode=mode)

    # Parse URL to determine protocol
    parsed = urlparse(url)
    protocol = parsed.scheme.lower()

    if protocol in ("file", ""):
        # Treat as local path
        logger.info("Creating local registry client for %s", url)
        return LocalRegistryClient(url, mode=mode)
    elif protocol == "s3":
        # S3 registry
        from dex.registry.s3 import S3RegistryClient

        logger.info("Creating S3 registry client for %s", url)
        return S3RegistryClient(url, mode=mode, cache_dir=cache_dir)
    elif protocol == "https":
        # HTTPS registry
        from dex.registry.https import HttpsRegistryClient

        logger.info("Creating HTTPS registry client for %s", url)
        return HttpsRegistryClient(url, mode=mode, cache_dir=cache_dir)
    elif protocol == "az":
        # Azure Blob Storage registry
        from dex.registry.azure import AzureRegistryClient

        logger.info("Creating Azure registry client for %s", url)
        return AzureRegistryClient(url, mode=mode, cache_dir=cache_dir)
    elif protocol in ("http", "git"):
        logger.error("Unsupported protocol: %s in URL %s", protocol, url)
        raise UnsupportedProtocolError(protocol, url)
    else:
        logger.error("Unsupported protocol: %s in URL %s", protocol, url)
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


def is_git_source(source: str) -> bool:
    """Check if a source string is a Git source.

    Args:
        source: Source URL or path

    Returns:
        True if the source is a Git URL (git+https://, git+ssh://)
    """
    return source.startswith("git+")


def is_s3_source(source: str) -> bool:
    """Check if a source string is an S3 source.

    Args:
        source: Source URL or path

    Returns:
        True if the source is an S3 URL (s3://)
    """
    return source.startswith("s3://")


def is_https_source(source: str) -> bool:
    """Check if a source string is an HTTPS source.

    Args:
        source: Source URL or path

    Returns:
        True if the source is an HTTPS URL (https://)
    """
    return source.startswith("https://")


def is_azure_source(source: str) -> bool:
    """Check if a source string is an Azure Blob Storage source.

    Args:
        source: Source URL or path

    Returns:
        True if the source is an Azure URL (az://)
    """
    return source.startswith("az://")


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

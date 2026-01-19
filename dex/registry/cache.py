"""Registry cache utilities for remote downloads."""

from __future__ import annotations

import hashlib
import json
import logging
import shutil
import time
from dataclasses import dataclass, field
from pathlib import Path
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from typing import Any

logger = logging.getLogger(__name__)


@dataclass
class CacheEntry:
    """A cached item with metadata."""

    url: str
    path: Path
    timestamp: float
    metadata: dict[str, Any] = field(default_factory=dict)


class RegistryCache:
    """File-based cache for remote registry downloads.

    Provides caching with TTL (time-to-live) for downloaded files
    from remote registries (S3, Git, HTTPS, etc.).

    Cache structure:
        cache_dir/
            metadata.json     # Tracks all cached items
            <sha256-hash>/    # Content directories keyed by URL hash
    """

    DEFAULT_TTL_SECONDS = 86400  # 24 hours

    def __init__(self, cache_dir: Path, ttl_seconds: int | None = None):
        """Initialize the registry cache.

        Args:
            cache_dir: Directory to store cached files
            ttl_seconds: Time-to-live in seconds (default: 24 hours)
        """
        self._cache_dir = cache_dir
        self._ttl_seconds = ttl_seconds or self.DEFAULT_TTL_SECONDS
        self._metadata_file = cache_dir / "metadata.json"
        self._entries: dict[str, CacheEntry] = {}
        self._load_metadata()
        logger.debug("Initialized cache at %s with TTL %d seconds", cache_dir, self._ttl_seconds)

    @property
    def cache_dir(self) -> Path:
        """Get the cache directory."""
        return self._cache_dir

    def _load_metadata(self) -> None:
        """Load cache metadata from disk."""
        if not self._metadata_file.exists():
            self._entries = {}
            return

        try:
            with open(self._metadata_file, encoding="utf-8") as f:
                data = json.load(f)

            self._entries = {}
            for url, entry_data in data.get("entries", {}).items():
                self._entries[url] = CacheEntry(
                    url=url,
                    path=Path(entry_data["path"]),
                    timestamp=entry_data["timestamp"],
                    metadata=entry_data.get("metadata", {}),
                )
        except (json.JSONDecodeError, KeyError):
            self._entries = {}

    def _save_metadata(self) -> None:
        """Save cache metadata to disk."""
        self._cache_dir.mkdir(parents=True, exist_ok=True)

        data = {
            "entries": {
                url: {
                    "path": str(entry.path),
                    "timestamp": entry.timestamp,
                    "metadata": entry.metadata,
                }
                for url, entry in self._entries.items()
            }
        }

        with open(self._metadata_file, "w", encoding="utf-8") as f:
            json.dump(data, f, indent=2)

    @staticmethod
    def _hash_url(url: str) -> str:
        """Generate a SHA256 hash of a URL for use as cache key."""
        return hashlib.sha256(url.encode("utf-8")).hexdigest()[:16]

    def _is_expired(self, entry: CacheEntry) -> bool:
        """Check if a cache entry has expired."""
        age = time.time() - entry.timestamp
        return age >= self._ttl_seconds

    def get(self, url: str) -> Path | None:
        """Get a cached path for a URL.

        Args:
            url: URL to look up in cache

        Returns:
            Path to cached content if valid cache exists, None otherwise
        """
        entry = self._entries.get(url)
        if entry is None:
            logger.debug("Cache miss for %s", url)
            return None

        if self._is_expired(entry):
            # Remove expired entry
            logger.debug("Cache entry expired for %s", url)
            self._remove_entry(url)
            return None

        if not entry.path.exists():
            # Remove orphaned entry
            logger.debug("Cache entry orphaned (file missing) for %s", url)
            del self._entries[url]
            self._save_metadata()
            return None

        logger.debug("Cache hit for %s at %s", url, entry.path)
        return entry.path

    def put(self, url: str, source_path: Path, metadata: dict[str, Any] | None = None) -> Path:
        """Add or update a cached item.

        Args:
            url: URL being cached
            source_path: Path to the content to cache (file or directory)
            metadata: Optional metadata to store with the entry

        Returns:
            Path to the cached content
        """
        cache_key = self._hash_url(url)
        cache_path = self._cache_dir / cache_key
        logger.debug("Caching %s to %s", url, cache_path)

        # Remove old entry if exists
        if url in self._entries:
            logger.debug("Replacing existing cache entry for %s", url)
            self._remove_entry(url)

        # Create cache directory
        self._cache_dir.mkdir(parents=True, exist_ok=True)

        # Copy content to cache
        if source_path.is_dir():
            if cache_path.exists():
                shutil.rmtree(cache_path)
            shutil.copytree(source_path, cache_path)
            logger.debug("Cached directory (%d files)", len(list(cache_path.rglob("*"))))
        else:
            cache_path.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source_path, cache_path)
            logger.debug("Cached file (%d bytes)", cache_path.stat().st_size)

        # Update metadata
        self._entries[url] = CacheEntry(
            url=url,
            path=cache_path,
            timestamp=time.time(),
            metadata=metadata or {},
        )
        self._save_metadata()

        return cache_path

    def _remove_entry(self, url: str) -> None:
        """Remove a cache entry and its files."""
        entry = self._entries.get(url)
        if entry is None:
            return

        if entry.path.exists():
            if entry.path.is_dir():
                shutil.rmtree(entry.path)
            else:
                entry.path.unlink()

        del self._entries[url]
        self._save_metadata()

    def clear(self) -> None:
        """Clear all cached items."""
        entry_count = len(self._entries)
        logger.info("Clearing cache (%d entries)", entry_count)
        for url in list(self._entries.keys()):
            self._remove_entry(url)

        # Also remove any orphaned directories
        if self._cache_dir.exists():
            for item in self._cache_dir.iterdir():
                if item != self._metadata_file:
                    if item.is_dir():
                        shutil.rmtree(item)
                    else:
                        item.unlink()

        self._entries = {}
        self._save_metadata()
        logger.debug("Cache cleared")

    def cleanup_expired(self) -> int:
        """Remove all expired cache entries.

        Returns:
            Number of entries removed
        """
        expired = [url for url, entry in self._entries.items() if self._is_expired(entry)]
        logger.debug("Found %d expired cache entries", len(expired))

        for url in expired:
            logger.debug("Removing expired entry: %s", url)
            self._remove_entry(url)

        if expired:
            logger.info("Cleaned up %d expired cache entries", len(expired))
        return len(expired)

    def get_entry(self, url: str) -> CacheEntry | None:
        """Get the cache entry for a URL (including metadata).

        Args:
            url: URL to look up

        Returns:
            CacheEntry if exists and valid, None otherwise
        """
        entry = self._entries.get(url)
        if entry is None:
            return None

        if self._is_expired(entry) or not entry.path.exists():
            return None

        return entry

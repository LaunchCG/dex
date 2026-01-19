"""Tests for dex.registry.cache module."""

import time
from pathlib import Path

from dex.registry.cache import RegistryCache


class TestRegistryCacheInit:
    """Tests for RegistryCache initialization."""

    def test_creates_cache_directory_on_first_put(self, temp_dir: Path):
        """Cache directory is created when first item is cached."""
        cache_dir = temp_dir / "cache"
        cache = RegistryCache(cache_dir)

        # Directory not created until first put
        assert not cache_dir.exists()

        # Create a test file
        test_file = temp_dir / "test.txt"
        test_file.write_text("test content")

        cache.put("https://example.com/test", test_file)

        assert cache_dir.exists()

    def test_loads_existing_metadata(self, temp_dir: Path):
        """Loads existing cache metadata on initialization."""
        cache_dir = temp_dir / "cache"
        cache1 = RegistryCache(cache_dir)

        test_file = temp_dir / "test.txt"
        test_file.write_text("test content")
        cache1.put("https://example.com/test", test_file)

        # Create new cache instance pointing to same directory
        cache2 = RegistryCache(cache_dir)

        assert cache2.get("https://example.com/test") is not None


class TestRegistryCacheGet:
    """Tests for RegistryCache.get()."""

    def test_returns_none_for_missing_url(self, temp_dir: Path):
        """Returns None for URLs not in cache."""
        cache = RegistryCache(temp_dir / "cache")

        result = cache.get("https://example.com/nonexistent")

        assert result is None

    def test_returns_path_for_cached_file(self, temp_dir: Path):
        """Returns path for cached file."""
        cache = RegistryCache(temp_dir / "cache")

        test_file = temp_dir / "test.txt"
        test_file.write_text("test content")
        cache.put("https://example.com/test", test_file)

        result = cache.get("https://example.com/test")

        assert result is not None
        assert result.exists()

    def test_returns_none_for_expired_entry(self, temp_dir: Path):
        """Returns None for expired cache entries."""
        cache = RegistryCache(temp_dir / "cache", ttl_seconds=1)

        test_file = temp_dir / "test.txt"
        test_file.write_text("test content")
        cache.put("https://example.com/test", test_file)

        # Wait for expiration
        time.sleep(1.5)

        result = cache.get("https://example.com/test")

        assert result is None

    def test_returns_none_for_orphaned_entry(self, temp_dir: Path):
        """Returns None when cached file was deleted externally."""
        cache = RegistryCache(temp_dir / "cache")

        test_file = temp_dir / "test.txt"
        test_file.write_text("test content")
        cached_path = cache.put("https://example.com/test", test_file)

        # Delete the cached file externally
        cached_path.unlink()

        result = cache.get("https://example.com/test")

        assert result is None


class TestRegistryCachePut:
    """Tests for RegistryCache.put()."""

    def test_caches_file(self, temp_dir: Path):
        """Caches a file and returns path to cached copy."""
        cache = RegistryCache(temp_dir / "cache")

        test_file = temp_dir / "test.txt"
        test_file.write_text("test content")

        result = cache.put("https://example.com/test", test_file)

        assert result.exists()
        assert result.read_text() == "test content"
        # Cached copy should be different from original
        assert result != test_file

    def test_caches_directory(self, temp_dir: Path):
        """Caches a directory and returns path to cached copy."""
        cache = RegistryCache(temp_dir / "cache")

        test_dir = temp_dir / "test-dir"
        test_dir.mkdir()
        (test_dir / "file1.txt").write_text("content 1")
        (test_dir / "subdir").mkdir()
        (test_dir / "subdir" / "file2.txt").write_text("content 2")

        result = cache.put("https://example.com/test-dir", test_dir)

        assert result.exists()
        assert result.is_dir()
        assert (result / "file1.txt").read_text() == "content 1"
        assert (result / "subdir" / "file2.txt").read_text() == "content 2"

    def test_replaces_existing_cache_entry(self, temp_dir: Path):
        """Replaces existing cache entry with new content."""
        cache = RegistryCache(temp_dir / "cache")

        test_file1 = temp_dir / "test1.txt"
        test_file1.write_text("original content")
        cache.put("https://example.com/test", test_file1)

        test_file2 = temp_dir / "test2.txt"
        test_file2.write_text("new content")
        result = cache.put("https://example.com/test", test_file2)

        assert result.read_text() == "new content"

    def test_stores_metadata(self, temp_dir: Path):
        """Stores metadata with cache entry."""
        cache = RegistryCache(temp_dir / "cache")

        test_file = temp_dir / "test.txt"
        test_file.write_text("test content")
        cache.put("https://example.com/test", test_file, metadata={"version": "1.0.0"})

        entry = cache.get_entry("https://example.com/test")

        assert entry is not None
        assert entry.metadata == {"version": "1.0.0"}


class TestRegistryCacheClear:
    """Tests for RegistryCache.clear()."""

    def test_removes_all_entries(self, temp_dir: Path):
        """Removes all cached entries."""
        cache = RegistryCache(temp_dir / "cache")

        test_file = temp_dir / "test.txt"
        test_file.write_text("test content")
        cache.put("https://example.com/test1", test_file)
        cache.put("https://example.com/test2", test_file)

        cache.clear()

        assert cache.get("https://example.com/test1") is None
        assert cache.get("https://example.com/test2") is None


class TestRegistryCacheCleanupExpired:
    """Tests for RegistryCache.cleanup_expired()."""

    def test_removes_expired_entries(self, temp_dir: Path):
        """Removes only expired entries."""
        cache = RegistryCache(temp_dir / "cache", ttl_seconds=1)

        test_file = temp_dir / "test.txt"
        test_file.write_text("test content")
        cache.put("https://example.com/old", test_file)

        # Wait for first entry to expire
        time.sleep(1.5)

        # Add new entry
        cache.put("https://example.com/new", test_file)

        removed = cache.cleanup_expired()

        assert removed == 1
        assert cache.get("https://example.com/old") is None
        assert cache.get("https://example.com/new") is not None

    def test_returns_count_of_removed_entries(self, temp_dir: Path):
        """Returns the number of entries removed."""
        cache = RegistryCache(temp_dir / "cache", ttl_seconds=1)

        test_file = temp_dir / "test.txt"
        test_file.write_text("test content")
        cache.put("https://example.com/1", test_file)
        cache.put("https://example.com/2", test_file)
        cache.put("https://example.com/3", test_file)

        time.sleep(1.5)

        removed = cache.cleanup_expired()

        assert removed == 3


class TestRegistryCacheGetEntry:
    """Tests for RegistryCache.get_entry()."""

    def test_returns_entry_with_metadata(self, temp_dir: Path):
        """Returns full entry including metadata."""
        cache = RegistryCache(temp_dir / "cache")

        test_file = temp_dir / "test.txt"
        test_file.write_text("test content")
        cache.put("https://example.com/test", test_file, metadata={"foo": "bar"})

        entry = cache.get_entry("https://example.com/test")

        assert entry is not None
        assert entry.url == "https://example.com/test"
        assert entry.path.exists()
        assert entry.metadata == {"foo": "bar"}
        assert entry.timestamp > 0

    def test_returns_none_for_missing(self, temp_dir: Path):
        """Returns None for missing entries."""
        cache = RegistryCache(temp_dir / "cache")

        entry = cache.get_entry("https://example.com/nonexistent")

        assert entry is None

    def test_returns_none_for_expired(self, temp_dir: Path):
        """Returns None for expired entries."""
        cache = RegistryCache(temp_dir / "cache", ttl_seconds=1)

        test_file = temp_dir / "test.txt"
        test_file.write_text("test content")
        cache.put("https://example.com/test", test_file)

        time.sleep(1.5)

        entry = cache.get_entry("https://example.com/test")

        assert entry is None


class TestRegistryCacheHashUrl:
    """Tests for URL hashing."""

    def test_consistent_hashing(self, temp_dir: Path):
        """Same URL produces same hash."""
        hash1 = RegistryCache._hash_url("https://example.com/test")
        hash2 = RegistryCache._hash_url("https://example.com/test")

        assert hash1 == hash2

    def test_different_urls_different_hashes(self, temp_dir: Path):
        """Different URLs produce different hashes."""
        hash1 = RegistryCache._hash_url("https://example.com/test1")
        hash2 = RegistryCache._hash_url("https://example.com/test2")

        assert hash1 != hash2

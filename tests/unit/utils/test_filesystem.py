"""Tests for dex.utils.filesystem module."""

import tarfile
from pathlib import Path

import pytest

from dex.utils.filesystem import (
    compute_file_hash,
    compute_integrity,
    copy_directory,
    copy_file,
    create_tarball,
    ensure_directory,
    extract_tarball,
    read_text_file,
    remove_directory,
    remove_file,
    write_text_file,
)


class TestEnsureDirectory:
    """Tests for ensure_directory function."""

    def test_creates_directory(self, temp_dir: Path):
        """Creates a new directory."""
        new_dir = temp_dir / "new_dir"
        assert not new_dir.exists()

        result = ensure_directory(new_dir)

        assert new_dir.exists()
        assert new_dir.is_dir()
        assert result == new_dir

    def test_creates_nested_directories(self, temp_dir: Path):
        """Creates nested directories."""
        nested_dir = temp_dir / "a" / "b" / "c"
        assert not nested_dir.exists()

        result = ensure_directory(nested_dir)

        assert nested_dir.exists()
        assert result == nested_dir

    def test_handles_existing_directory(self, temp_dir: Path):
        """Handles existing directory without error."""
        existing_dir = temp_dir / "existing"
        existing_dir.mkdir()

        result = ensure_directory(existing_dir)

        assert existing_dir.exists()
        assert result == existing_dir


class TestCopyFile:
    """Tests for copy_file function."""

    def test_copies_file_to_directory(self, temp_dir: Path):
        """Copies a file to a directory."""
        src_file = temp_dir / "source.txt"
        src_file.write_text("content")
        dest_dir = temp_dir / "dest"
        dest_dir.mkdir()

        result = copy_file(src_file, dest_dir)

        assert result == dest_dir / "source.txt"
        assert result.exists()
        assert result.read_text() == "content"

    def test_copies_file_to_file(self, temp_dir: Path):
        """Copies a file to a specific file path."""
        src_file = temp_dir / "source.txt"
        src_file.write_text("content")
        dest_file = temp_dir / "dest.txt"

        result = copy_file(src_file, dest_file)

        assert result == dest_file
        assert dest_file.read_text() == "content"

    def test_creates_parent_directories(self, temp_dir: Path):
        """Creates parent directories if needed."""
        src_file = temp_dir / "source.txt"
        src_file.write_text("content")
        dest_file = temp_dir / "new" / "nested" / "dest.txt"

        result = copy_file(src_file, dest_file)

        assert result.exists()
        assert result.read_text() == "content"


class TestCopyDirectory:
    """Tests for copy_directory function."""

    def test_copies_directory(self, temp_dir: Path):
        """Copies a directory recursively."""
        src_dir = temp_dir / "src"
        src_dir.mkdir()
        (src_dir / "file.txt").write_text("content")
        (src_dir / "subdir").mkdir()
        (src_dir / "subdir" / "nested.txt").write_text("nested")

        dest_dir = temp_dir / "dest"
        result = copy_directory(src_dir, dest_dir)

        assert result == dest_dir
        assert (dest_dir / "file.txt").read_text() == "content"
        assert (dest_dir / "subdir" / "nested.txt").read_text() == "nested"

    def test_replaces_existing_directory(self, temp_dir: Path):
        """Replaces existing directory."""
        src_dir = temp_dir / "src"
        src_dir.mkdir()
        (src_dir / "new.txt").write_text("new")

        dest_dir = temp_dir / "dest"
        dest_dir.mkdir()
        (dest_dir / "old.txt").write_text("old")

        copy_directory(src_dir, dest_dir)

        assert (dest_dir / "new.txt").exists()
        assert not (dest_dir / "old.txt").exists()


class TestRemoveDirectory:
    """Tests for remove_directory function."""

    def test_removes_directory(self, temp_dir: Path):
        """Removes an existing directory."""
        dir_to_remove = temp_dir / "to_remove"
        dir_to_remove.mkdir()
        (dir_to_remove / "file.txt").write_text("content")

        result = remove_directory(dir_to_remove)

        assert result is True
        assert not dir_to_remove.exists()

    def test_returns_false_for_nonexistent(self, temp_dir: Path):
        """Returns False for nonexistent directory."""
        nonexistent = temp_dir / "nonexistent"

        result = remove_directory(nonexistent)

        assert result is False


class TestRemoveFile:
    """Tests for remove_file function."""

    def test_removes_file(self, temp_dir: Path):
        """Removes an existing file."""
        file_to_remove = temp_dir / "to_remove.txt"
        file_to_remove.write_text("content")

        result = remove_file(file_to_remove)

        assert result is True
        assert not file_to_remove.exists()

    def test_returns_false_for_nonexistent(self, temp_dir: Path):
        """Returns False for nonexistent file."""
        nonexistent = temp_dir / "nonexistent.txt"

        result = remove_file(nonexistent)

        assert result is False


class TestExtractTarball:
    """Tests for extract_tarball function."""

    def test_extracts_tarball(self, temp_dir: Path):
        """Extracts a tarball to destination."""
        # Create a tarball
        src_dir = temp_dir / "source"
        src_dir.mkdir()
        (src_dir / "file.txt").write_text("content")

        tarball_path = temp_dir / "archive.tar.gz"
        with tarfile.open(tarball_path, "w:gz") as tar:
            tar.add(src_dir, arcname="source")

        dest_dir = temp_dir / "extract"
        result = extract_tarball(tarball_path, dest_dir)

        # Should return the extracted directory
        assert result.exists()
        assert (result / "file.txt").read_text() == "content"

    def test_returns_single_directory(self, temp_dir: Path):
        """Returns the single top-level directory if present."""
        # Create a tarball with single top-level dir
        src_dir = temp_dir / "plugin"
        src_dir.mkdir()
        (src_dir / "file.txt").write_text("content")

        tarball_path = temp_dir / "archive.tar.gz"
        with tarfile.open(tarball_path, "w:gz") as tar:
            tar.add(src_dir, arcname="plugin")

        dest_dir = temp_dir / "extract"
        result = extract_tarball(tarball_path, dest_dir)

        assert result.name == "plugin"
        assert (result / "file.txt").exists()

    def test_rejects_path_traversal(self, temp_dir: Path):
        """Rejects tarballs with path traversal."""
        tarball_path = temp_dir / "malicious.tar.gz"

        # Create a tarball with path traversal
        with tarfile.open(tarball_path, "w:gz") as tar:
            # Add a file with path traversal
            info = tarfile.TarInfo(name="../../../etc/passwd")
            info.size = 0
            tar.addfile(info)

        dest_dir = temp_dir / "extract"
        with pytest.raises(ValueError, match="Unsafe path"):
            extract_tarball(tarball_path, dest_dir)


class TestCreateTarball:
    """Tests for create_tarball function."""

    def test_creates_tarball(self, temp_dir: Path):
        """Creates a valid tarball."""
        src_dir = temp_dir / "source"
        src_dir.mkdir()
        (src_dir / "file.txt").write_text("content")

        tarball_path = temp_dir / "archive.tar.gz"
        result = create_tarball(src_dir, tarball_path)

        assert result == tarball_path
        assert tarball_path.exists()

        # Verify contents
        with tarfile.open(tarball_path, "r:gz") as tar:
            names = tar.getnames()
            assert any("file.txt" in name for name in names)

    def test_creates_parent_directories(self, temp_dir: Path):
        """Creates parent directories for tarball."""
        src_dir = temp_dir / "source"
        src_dir.mkdir()

        tarball_path = temp_dir / "nested" / "dir" / "archive.tar.gz"
        result = create_tarball(src_dir, tarball_path)

        assert result.exists()


class TestComputeFileHash:
    """Tests for compute_file_hash function."""

    def test_computes_sha256_hash(self, temp_dir: Path):
        """Computes SHA256 hash by default."""
        file_path = temp_dir / "file.txt"
        file_path.write_text("test content")

        result = compute_file_hash(file_path)

        # Should be a hex string
        assert isinstance(result, str)
        assert len(result) == 64  # SHA256 produces 64 hex chars
        assert all(c in "0123456789abcdef" for c in result)

    def test_same_content_same_hash(self, temp_dir: Path):
        """Same content produces same hash."""
        file1 = temp_dir / "file1.txt"
        file2 = temp_dir / "file2.txt"
        file1.write_text("same content")
        file2.write_text("same content")

        assert compute_file_hash(file1) == compute_file_hash(file2)

    def test_different_content_different_hash(self, temp_dir: Path):
        """Different content produces different hash."""
        file1 = temp_dir / "file1.txt"
        file2 = temp_dir / "file2.txt"
        file1.write_text("content 1")
        file2.write_text("content 2")

        assert compute_file_hash(file1) != compute_file_hash(file2)


class TestComputeIntegrity:
    """Tests for compute_integrity function."""

    def test_returns_sri_format(self, temp_dir: Path):
        """Returns SRI format integrity string."""
        file_path = temp_dir / "file.txt"
        file_path.write_text("test content")

        result = compute_integrity(file_path)

        assert result.startswith("sha512-")
        # Base64 encoded SHA512 should be 88 chars
        assert len(result.split("-")[1]) == 88


class TestReadTextFile:
    """Tests for read_text_file function."""

    def test_reads_file_content(self, temp_dir: Path):
        """Reads file content as string."""
        file_path = temp_dir / "file.txt"
        file_path.write_text("test content", encoding="utf-8")

        result = read_text_file(file_path)

        assert result == "test content"

    def test_reads_utf8_content(self, temp_dir: Path):
        """Reads UTF-8 encoded content."""
        file_path = temp_dir / "file.txt"
        file_path.write_text("日本語テスト", encoding="utf-8")

        result = read_text_file(file_path)

        assert result == "日本語テスト"


class TestWriteTextFile:
    """Tests for write_text_file function."""

    def test_writes_file_content(self, temp_dir: Path):
        """Writes content to file."""
        file_path = temp_dir / "file.txt"

        write_text_file(file_path, "test content")

        assert file_path.read_text() == "test content"

    def test_creates_parent_directories(self, temp_dir: Path):
        """Creates parent directories if needed."""
        file_path = temp_dir / "nested" / "dir" / "file.txt"

        write_text_file(file_path, "content")

        assert file_path.exists()
        assert file_path.read_text() == "content"

    def test_writes_utf8_content(self, temp_dir: Path):
        """Writes UTF-8 encoded content."""
        file_path = temp_dir / "file.txt"

        write_text_file(file_path, "日本語テスト")

        assert file_path.read_text(encoding="utf-8") == "日本語テスト"

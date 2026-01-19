"""Filesystem utilities for Dex."""

import hashlib
import shutil
import tarfile
from pathlib import Path


def ensure_directory(path: Path) -> Path:
    """Ensure a directory exists, creating it if necessary.

    Args:
        path: Directory path to ensure exists

    Returns:
        The directory path
    """
    path.mkdir(parents=True, exist_ok=True)
    return path


def copy_file(src: Path, dest: Path) -> Path:
    """Copy a file to a destination.

    Args:
        src: Source file path
        dest: Destination path (file or directory)

    Returns:
        Path to the copied file
    """
    if dest.is_dir():
        dest = dest / src.name
    dest.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(src, dest)
    return dest


def copy_directory(src: Path, dest: Path) -> Path:
    """Copy a directory recursively.

    Args:
        src: Source directory path
        dest: Destination directory path

    Returns:
        Path to the copied directory
    """
    if dest.exists():
        shutil.rmtree(dest)
    shutil.copytree(src, dest)
    return dest


def remove_directory(path: Path) -> bool:
    """Remove a directory and its contents.

    Args:
        path: Directory path to remove

    Returns:
        True if the directory was removed, False if it didn't exist
    """
    if not path.exists():
        return False
    shutil.rmtree(path)
    return True


def remove_file(path: Path) -> bool:
    """Remove a file.

    Args:
        path: File path to remove

    Returns:
        True if the file was removed, False if it didn't exist
    """
    if not path.exists():
        return False
    path.unlink()
    return True


def extract_tarball(tarball_path: Path, dest_dir: Path) -> Path:
    """Extract a tarball to a destination directory.

    Args:
        tarball_path: Path to the .tar.gz file
        dest_dir: Destination directory

    Returns:
        Path to the extracted content directory
    """
    dest_dir.mkdir(parents=True, exist_ok=True)

    with tarfile.open(tarball_path, "r:gz") as tar:
        # Security: prevent path traversal
        for member in tar.getmembers():
            member_path = Path(member.name)
            if member_path.is_absolute() or ".." in member_path.parts:
                raise ValueError(f"Unsafe path in tarball: {member.name}")
        tar.extractall(dest_dir)

    # If there's a single top-level directory, return its path
    # Ignore macOS metadata files (._*) and other hidden files
    contents = [
        p for p in dest_dir.iterdir() if not p.name.startswith("._") and not p.name.startswith(".")
    ]
    if len(contents) == 1 and contents[0].is_dir():
        return contents[0]
    return dest_dir


def create_tarball(source_dir: Path, tarball_path: Path) -> Path:
    """Create a tarball from a directory.

    Args:
        source_dir: Directory to archive
        tarball_path: Path for the output .tar.gz file

    Returns:
        Path to the created tarball
    """
    tarball_path.parent.mkdir(parents=True, exist_ok=True)

    with tarfile.open(tarball_path, "w:gz") as tar:
        tar.add(source_dir, arcname=source_dir.name)

    return tarball_path


def compute_file_hash(path: Path, algorithm: str = "sha256") -> str:
    """Compute the hash of a file.

    Args:
        path: Path to the file
        algorithm: Hash algorithm (default: sha256)

    Returns:
        Hex-encoded hash string
    """
    hasher = hashlib.new(algorithm)
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(8192), b""):
            hasher.update(chunk)
    return hasher.hexdigest()


def compute_integrity(path: Path) -> str:
    """Compute an integrity string for a file (SRI format).

    Args:
        path: Path to the file

    Returns:
        SRI-format integrity string (e.g., "sha512-abc123...")
    """
    import base64

    hasher = hashlib.sha512()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(8192), b""):
            hasher.update(chunk)

    hash_bytes = hasher.digest()
    hash_b64 = base64.b64encode(hash_bytes).decode("ascii")
    return f"sha512-{hash_b64}"


def read_text_file(path: Path) -> str:
    """Read a text file.

    Args:
        path: Path to the file

    Returns:
        File contents as a string
    """
    return path.read_text(encoding="utf-8")


def write_text_file(path: Path, content: str) -> None:
    """Write content to a text file.

    Args:
        path: Path to the file
        content: Content to write
    """
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")

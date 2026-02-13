#!/usr/bin/env bash
set -euo pipefail

###############################################################################
# install.sh - Install dex binary
#
# Usage:
#   curl -fsSL <url>/install.sh | bash
#   curl -fsSL <url>/install.sh | bash -s -- --version 0.1.0
#
# Options:
#   --version <ver>  Install a specific version (default: latest)
###############################################################################

ARTIFACTS_URL="https://dexartifactsproduction.z13.web.core.windows.net"
INSTALL_DIR="${HOME}/.bin"
BINARY_NAME="dex"

# Parse arguments
VERSION=""
while [ $# -gt 0 ]; do
  case "$1" in
    --version)
      VERSION="$2"
      shift 2
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

# Detect OS
case "$(uname -s)" in
  Linux*)  OS="linux" ;;
  Darwin*) OS="darwin" ;;
  *)
    echo "Error: Unsupported operating system: $(uname -s)"
    exit 1
    ;;
esac

# Detect architecture
case "$(uname -m)" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)
    echo "Error: Unsupported architecture: $(uname -m)"
    exit 1
    ;;
esac

# Fetch latest version if not specified
if [ -z "$VERSION" ]; then
  echo "Fetching latest version..."
  VERSION=$(curl -fsSL "${ARTIFACTS_URL}/latest") || {
    echo "Error: Could not fetch latest version from ${ARTIFACTS_URL}/latest"
    exit 1
  }
fi

ARCHIVE="dex-${VERSION}-${OS}-${ARCH}.tar.gz"
DOWNLOAD_URL="${ARTIFACTS_URL}/${ARCHIVE}"

echo "Installing dex ${VERSION} (${OS}/${ARCH})..."

# Create temp directory
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

# Download
echo "Downloading ${DOWNLOAD_URL}..."
curl -fsSL "$DOWNLOAD_URL" -o "${TMP_DIR}/${ARCHIVE}" || {
  echo "Error: Download failed. Check that version '${VERSION}' exists for ${OS}/${ARCH}."
  exit 1
}

# Extract
echo "Extracting..."
tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "${TMP_DIR}"

# Install
mkdir -p "$INSTALL_DIR"
mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

echo "Installed to ${INSTALL_DIR}/${BINARY_NAME}"

# Verify
if "${INSTALL_DIR}/${BINARY_NAME}" version &>/dev/null; then
  echo ""
  "$INSTALL_DIR/${BINARY_NAME}" version
else
  echo "Warning: Could not verify installation."
fi

# PATH reminder (always shown)
echo ""
echo "============================================"
echo "  Installation complete!"
echo "============================================"
echo ""
echo "  Binary:  ${INSTALL_DIR}/${BINARY_NAME}"
echo ""
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*)
    echo "  PATH:    ${INSTALL_DIR} is already in your PATH. You're all set!"
    ;;
  *)
    echo "  WARNING: ${INSTALL_DIR} is NOT in your PATH."
    echo ""
    echo "  Add it by running one of the following:"
    echo ""
    echo "    # bash"
    echo "    echo 'export PATH=\"\$HOME/.bin:\$PATH\"' >> ~/.bashrc && source ~/.bashrc"
    echo ""
    echo "    # zsh"
    echo "    echo 'export PATH=\"\$HOME/.bin:\$PATH\"' >> ~/.zshrc && source ~/.zshrc"
    echo ""
    echo "    # fish"
    echo "    fish_add_path ~/.bin"
    ;;
esac
echo ""

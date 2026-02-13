#!/usr/bin/env bash
set -euo pipefail

###############################################################################
# build-all.sh - Cross-compile dex for all supported platforms
#
# Usage: ./scripts/build-all.sh
#
# Environment variables:
#   VERSION - Version string (default: git describe or "dev")
#
# Outputs to build/:
#   dex-{version}-{os}-{arch}.tar.gz  (linux, darwin)
#   dex-{version}-windows-amd64.zip   (windows)
#   latest                            (version string)
#   checksums.txt                     (SHA-256 checksums)
###############################################################################

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
BUILD_DIR="${PROJECT_DIR}/build"

# Version sourcing: $VERSION env var > git describe > "dev"
VERSION="${VERSION:-$(git describe --tags --always 2>/dev/null || echo "dev")}"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")"
DATE="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

LDFLAGS="-s -w \
  -X github.com/launchcg/dex/internal/cli.Version=${VERSION} \
  -X github.com/launchcg/dex/internal/cli.Commit=${COMMIT} \
  -X github.com/launchcg/dex/internal/cli.Date=${DATE}"

# Target platforms: os/arch
TARGETS=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
)

echo "============================================"
echo "  Dex - Cross-Platform Build"
echo "============================================"
echo ""
echo "Version : ${VERSION}"
echo "Commit  : ${COMMIT}"
echo "Date    : ${DATE}"
echo "Output  : ${BUILD_DIR}"
echo ""

# Clean and create build directory
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

for target in "${TARGETS[@]}"; do
  os="${target%/*}"
  arch="${target#*/}"

  binary_name="dex"
  if [ "${os}" = "windows" ]; then
    binary_name="dex.exe"
  fi

  archive_name="dex-${VERSION}-${os}-${arch}"
  echo "Building ${os}/${arch}..."

  # Build
  CGO_ENABLED=0 GOOS="${os}" GOARCH="${arch}" go build \
    -ldflags "${LDFLAGS}" \
    -o "${BUILD_DIR}/${binary_name}" \
    ./cmd/dex

  # Package
  staging_dir="${BUILD_DIR}/staging-${os}-${arch}"
  mkdir -p "${staging_dir}"
  mv "${BUILD_DIR}/${binary_name}" "${staging_dir}/"

  if [ "${os}" = "windows" ]; then
    (cd "${staging_dir}" && zip -q "${BUILD_DIR}/${archive_name}.zip" "${binary_name}")
  else
    tar -czf "${BUILD_DIR}/${archive_name}.tar.gz" -C "${staging_dir}" "${binary_name}"
  fi

  rm -rf "${staging_dir}"
  echo "  -> ${archive_name}$([ "${os}" = "windows" ] && echo ".zip" || echo ".tar.gz")"
done

# Generate latest file
echo "${VERSION}" > "${BUILD_DIR}/latest"
echo ""
echo "Generated latest file: ${VERSION}"

# Generate checksums
echo ""
echo "Generating checksums..."
(cd "${BUILD_DIR}" && shasum -a 256 dex-*.tar.gz dex-*.zip 2>/dev/null > checksums.txt)
echo "  -> checksums.txt"

echo ""
echo "============================================"
echo "  Build complete!"
echo "============================================"
echo ""
echo "Archives:"
ls -lh "${BUILD_DIR}"/dex-* 2>/dev/null

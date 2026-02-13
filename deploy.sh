#!/usr/bin/env bash
set -euo pipefail

###############################################################################
# deploy.sh - Build and deploy dex artifacts to Azure Blob Storage
#
# Usage: ./deploy.sh [flags]
#
# Flags:
#   --infra   Deploy Azure infrastructure (runs infrastructure/deploy.sh)
#   --clean   Remove build/ directory before building
#
# This script:
#   1. Optionally deploys infrastructure
#   2. Sources infrastructure/config.sh for Azure settings
#   3. Runs scripts/build-all.sh to cross-compile
#   4. Copies install scripts into build/
#   5. Uploads build/ to Azure $web container
###############################################################################

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

DEPLOY_INFRA=false
CLEAN_BUILD=false

for arg in "$@"; do
  case "$arg" in
    --infra) DEPLOY_INFRA=true ;;
    --clean) CLEAN_BUILD=true ;;
    *) echo "Unknown flag: $arg"; exit 1 ;;
  esac
done

echo "============================================"
echo "  Dex - Build & Deploy"
echo "============================================"
echo ""

# ---- Optionally deploy infrastructure ---------------------------------------
if [ "$DEPLOY_INFRA" = true ]; then
  echo "Deploying infrastructure..."
  "${SCRIPT_DIR}/infrastructure/deploy.sh"
  echo ""
fi

# ---- Source config -----------------------------------------------------------
CONFIG_FILE="${SCRIPT_DIR}/infrastructure/config.sh"
if [ ! -f "$CONFIG_FILE" ]; then
  echo "ERROR: ${CONFIG_FILE} not found."
  echo "Run ./infrastructure/deploy.sh first, or use --infra flag."
  exit 1
fi

source "$CONFIG_FILE"
echo "Using storage account: ${STORAGE_ACCOUNT}"
echo "Artifacts URL: ${ARTIFACTS_URL}"
echo ""

# ---- Optionally clean -------------------------------------------------------
if [ "$CLEAN_BUILD" = true ]; then
  echo "Cleaning build directory..."
  rm -rf "${SCRIPT_DIR}/build"
  echo ""
fi

# ---- Build all platforms -----------------------------------------------------
echo "Building all platforms..."
"${SCRIPT_DIR}/scripts/build-all.sh"
echo ""

# ---- Copy install scripts into build/ ---------------------------------------
echo "Copying install scripts to build/..."
cp "${SCRIPT_DIR}/scripts/install.sh" "${SCRIPT_DIR}/build/"
cp "${SCRIPT_DIR}/scripts/install.ps1" "${SCRIPT_DIR}/build/"
echo "  -> install.sh"
echo "  -> install.ps1"
echo ""

# ---- Upload to Azure ---------------------------------------------------------
echo "Uploading to Azure Blob Storage..."
echo "  Container: ${CONTAINER_NAME}"

az storage blob upload-batch \
  --account-name "${STORAGE_ACCOUNT}" \
  --destination "${CONTAINER_NAME}" \
  --source "${SCRIPT_DIR}/build" \
  --overwrite \
  --auth-mode key \
  --output none

echo ""
echo "============================================"
echo "  Deploy complete!"
echo "============================================"
echo ""
echo "Artifacts available at: ${ARTIFACTS_URL}"
echo ""
echo "Install (unix):      curl -fsSL ${ARTIFACTS_URL}install.sh | bash"
echo "Install (powershell): irm ${ARTIFACTS_URL}install.ps1 | iex"

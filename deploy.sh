#!/usr/bin/env bash
set -euo pipefail

###############################################################################
# deploy.sh - Build and deploy dex to Azure
#
# Usage: ./deploy.sh [flags]
#
# Flags:
#   --infra   Provision/update Azure infrastructure first
#   --clean   Remove build/ directory before building
#   --nuke    Destroy all Azure resources
#
# This script:
#   1. Checks for a clean semver tag on the current commit
#   2. Optionally provisions infrastructure
#   3. Sources infrastructure/config.sh for Azure settings
#   4. Runs scripts/build-all.sh to cross-compile
#   5. Copies install scripts into build/
#   6. Uploads build/ to Azure $web container (binaries + install scripts)
#   7. Deploys website/ to Azure Static Web App (index.html, docs.html)
###############################################################################

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

DEPLOY_INFRA=false
CLEAN_BUILD=false
NUKE=false

for arg in "$@"; do
  case "$arg" in
    --infra) DEPLOY_INFRA=true ;;
    --clean) CLEAN_BUILD=true ;;
    --nuke)  NUKE=true ;;
    *) echo "Unknown flag: $arg"; exit 1 ;;
  esac
done

# ---- Nuke -------------------------------------------------------------------
if [ "$NUKE" = true ]; then
  RESOURCE_GROUP="dex-artifacts-rg"
  echo "============================================"
  echo "  DESTROYING all resources in ${RESOURCE_GROUP}"
  echo "============================================"
  echo ""
  read -r -p "Are you sure? This deletes EVERYTHING. [y/N] " confirm
  if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    echo "Aborted."
    exit 0
  fi
  echo ""
  echo "==> Deleting resource group '${RESOURCE_GROUP}'..."
  az group delete --name "${RESOURCE_GROUP}" --yes --no-wait
  echo "==> Deletion initiated (runs in background)."
  rm -f infrastructure/config.sh
  echo "Done."
  exit 0
fi

# ---- Require a clean semver tag on the current commit -----------------------
VERSION="$(git describe --exact-match --tags HEAD 2>/dev/null || true)"
if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Error: current commit must be tagged with a release version (e.g. v1.2.3)."
  echo "Current: ${VERSION:-<no tag>}"
  exit 1
fi

echo "============================================"
echo "  Dex - Build & Deploy ${VERSION}"
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

# ---- Deploy website to Static Web App ---------------------------------------
if [ -z "${DEPLOYMENT_TOKEN:-}" ]; then
  echo "WARNING: DEPLOYMENT_TOKEN not set in config.sh — skipping website deploy."
  echo "         Run with --infra to provision the Static Web App."
else
  echo "Deploying website to Azure Static Web App..."
  SWA_CLI_DEPLOYMENT_TOKEN="${DEPLOYMENT_TOKEN}" \
    npx --yes @azure/static-web-apps-cli@latest deploy \
    --output-location "${SCRIPT_DIR}/website" \
    --env production
  echo ""
fi

echo "============================================"
echo "  Deploy complete!"
echo "============================================"
echo ""
echo "Artifacts : ${ARTIFACTS_URL}"
echo "Website   : ${WEBSITE_URL:-<run --infra to provision SWA>}"
echo ""
echo "Install (unix):       curl -fsSL ${ARTIFACTS_URL}install.sh | bash"
echo "Install (powershell): irm ${ARTIFACTS_URL}install.ps1 | iex"

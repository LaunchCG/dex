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
#   --site    Deploy only the website (no build, no version tag required)
#   --nuke    Destroy all Azure resources
#
# This script:
#   1. Optionally provisions infrastructure
#   2. Looks up Azure storage account from the resource group
#   3. Checks for a clean semver tag on the current commit (skipped with --site)
#   4. Runs scripts/build-all.sh to cross-compile (skipped with --site)
#   5. Copies install scripts into build/ (skipped with --site)
#   6. Uploads build/ to Azure $web container (skipped with --site)
#   7. Uploads website/ to Azure $web container
###############################################################################

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_GROUP="dex-artifacts-rg"

DEPLOY_INFRA=false
CLEAN_BUILD=false
SITE_ONLY=false
NUKE=false

for arg in "$@"; do
  case "$arg" in
    --infra) DEPLOY_INFRA=true ;;
    --clean) CLEAN_BUILD=true ;;
    --site)  SITE_ONLY=true ;;
    --nuke)  NUKE=true ;;
    *) echo "Unknown flag: $arg"; exit 1 ;;
  esac
done

# ---- Nuke -------------------------------------------------------------------
if [ "$NUKE" = true ]; then
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
  echo "Done."
  exit 0
fi

# ---- Optionally deploy infrastructure ---------------------------------------
if [ "$DEPLOY_INFRA" = true ]; then
  echo "Deploying infrastructure..."
  "${SCRIPT_DIR}/infrastructure/deploy.sh"
  echo ""
fi

# ---- Resolve Azure config from resource group --------------------------------
echo "Looking up storage account in '${RESOURCE_GROUP}'..."

STORAGE_ACCOUNT=$(az storage account list \
  --resource-group "${RESOURCE_GROUP}" \
  --query "[0].name" \
  --output tsv 2>/dev/null || true)

if [ -z "${STORAGE_ACCOUNT}" ]; then
  echo "ERROR: No storage account found in resource group '${RESOURCE_GROUP}'."
  echo "Run with --infra to provision infrastructure first."
  exit 1
fi

CONTAINER_NAME='$web'
ARTIFACTS_URL=$(az storage account show \
  --name "${STORAGE_ACCOUNT}" \
  --resource-group "${RESOURCE_GROUP}" \
  --query "primaryEndpoints.web" \
  --output tsv)

echo "  Storage account : ${STORAGE_ACCOUNT}"
echo "  Artifacts URL   : ${ARTIFACTS_URL}"
echo ""

# ---- Build & upload artifacts (skipped with --site) --------------------------
if [ "$SITE_ONLY" = false ]; then

  # Require a clean semver tag on the current commit
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

  # Optionally clean
  if [ "$CLEAN_BUILD" = true ]; then
    echo "Cleaning build directory..."
    rm -rf "${SCRIPT_DIR}/build"
    echo ""
  fi

  # Build all platforms
  echo "Building all platforms..."
  "${SCRIPT_DIR}/scripts/build-all.sh"
  echo ""

  # Copy install scripts into build/
  echo "Copying install scripts to build/..."
  cp "${SCRIPT_DIR}/scripts/install.sh" "${SCRIPT_DIR}/build/"
  cp "${SCRIPT_DIR}/scripts/install.ps1" "${SCRIPT_DIR}/build/"
  echo "  -> install.sh"
  echo "  -> install.ps1"
  echo ""

  # Upload artifacts to Azure
  echo "Uploading artifacts to Azure Blob Storage..."
  echo "  Container: ${CONTAINER_NAME}"

  az storage blob upload-batch \
    --account-name "${STORAGE_ACCOUNT}" \
    --destination "${CONTAINER_NAME}" \
    --source "${SCRIPT_DIR}/build" \
    --overwrite \
    --auth-mode key \
    --output none

  echo ""
fi

# ---- Upload website to Azure (always runs) -----------------------------------
echo "Uploading website to Azure Blob Storage..."

az storage blob upload-batch \
  --account-name "${STORAGE_ACCOUNT}" \
  --destination "${CONTAINER_NAME}" \
  --source "${SCRIPT_DIR}/website" \
  --overwrite \
  --auth-mode key \
  --output none

echo ""

echo "============================================"
echo "  Deploy complete!"
echo "============================================"
echo ""
echo "Website & Artifacts : ${ARTIFACTS_URL}"
echo ""
echo "Install (unix):       curl -fsSL ${ARTIFACTS_URL}install.sh | bash"
echo "Install (powershell): irm ${ARTIFACTS_URL}install.ps1 | iex"

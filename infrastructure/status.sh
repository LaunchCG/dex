#!/usr/bin/env bash
set -euo pipefail

###############################################################################
# status.sh - Check if dex artifact infrastructure is already provisioned
#
# Usage: ./infrastructure/status.sh [RESOURCE_GROUP]
#
# This script checks the current Azure account for:
#   1. Azure CLI authentication
#   2. Resource group existence
#   3. Storage account provisioned by the Bicep template
#   4. Static website hosting configuration
#   5. Latest version available
###############################################################################

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

RESOURCE_GROUP="${1:-dex-artifacts-rg}"

echo "============================================"
echo "  Dex Artifacts - Infrastructure Status"
echo "============================================"
echo ""

# ---- Check Azure CLI auth ------------------------------------------------
echo "[1/5] Checking Azure CLI authentication..."
if ! az account show &>/dev/null; then
  echo "       Not authenticated. Run: az login"
  exit 1
fi

ACCOUNT_NAME=$(az account show --query name -o tsv)
SUBSCRIPTION_ID=$(az account show --query id -o tsv)
echo "       Account      : ${ACCOUNT_NAME}"
echo "       Subscription : ${SUBSCRIPTION_ID}"

# ---- Check resource group -------------------------------------------------
echo ""
echo "[2/5] Checking resource group '${RESOURCE_GROUP}'..."
if ! az group show --name "${RESOURCE_GROUP}" &>/dev/null; then
  echo "       Resource group does not exist. No infrastructure provisioned."
  echo ""
  echo "  To deploy: ./infrastructure/deploy.sh"
  exit 0
fi

LOCATION=$(az group show --name "${RESOURCE_GROUP}" --query location -o tsv)
echo "       Exists in: ${LOCATION}"

# ---- Check storage account ------------------------------------------------
echo ""
echo "[3/5] Checking storage accounts in '${RESOURCE_GROUP}'..."
STORAGE_ACCOUNTS=$(az storage account list \
  --resource-group "${RESOURCE_GROUP}" \
  --query "[].{name:name, sku:sku.name, created:creationTime}" \
  -o json)

ACCOUNT_COUNT=$(echo "$STORAGE_ACCOUNTS" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))")

if [ "$ACCOUNT_COUNT" = "0" ]; then
  echo "       No storage accounts found. Infrastructure not provisioned."
  echo ""
  echo "  To deploy: ./infrastructure/deploy.sh"
  exit 0
fi

STORAGE_ACCOUNT=$(echo "$STORAGE_ACCOUNTS" | python3 -c "import sys,json; print(json.load(sys.stdin)[0]['name'])")
STORAGE_SKU=$(echo "$STORAGE_ACCOUNTS" | python3 -c "import sys,json; print(json.load(sys.stdin)[0]['sku'])")
STORAGE_CREATED=$(echo "$STORAGE_ACCOUNTS" | python3 -c "import sys,json; print(json.load(sys.stdin)[0]['created'])")

echo "       Storage Account : ${STORAGE_ACCOUNT}"
echo "       SKU             : ${STORAGE_SKU}"
echo "       Created         : ${STORAGE_CREATED}"

# ---- Check static website hosting -----------------------------------------
echo ""
echo "[4/5] Checking static website hosting..."
STATIC_WEBSITE=$(az storage blob service-properties show \
  --account-name "${STORAGE_ACCOUNT}" \
  --auth-mode key \
  --query "staticWebsite" \
  -o json 2>/dev/null) || true

if [ -z "$STATIC_WEBSITE" ]; then
  echo "       Could not query static website properties."
  echo "       (May need storage account key access)"
else
  ENABLED=$(echo "$STATIC_WEBSITE" | python3 -c "import sys,json; print(json.load(sys.stdin).get('enabled', False))")
  INDEX_DOC=$(echo "$STATIC_WEBSITE" | python3 -c "import sys,json; print(json.load(sys.stdin).get('indexDocument', 'N/A'))")

  if [ "$ENABLED" = "True" ]; then
    echo "       Static website : ENABLED"
    echo "       Index document : ${INDEX_DOC}"

    WEB_ENDPOINT=$(az storage account show \
      --name "${STORAGE_ACCOUNT}" \
      --query "primaryEndpoints.web" -o tsv)
    BLOB_ENDPOINT=$(az storage account show \
      --name "${STORAGE_ACCOUNT}" \
      --query "primaryEndpoints.blob" -o tsv)
    echo "       Artifacts URL  : ${WEB_ENDPOINT}"
    echo "       Blob Endpoint  : ${BLOB_ENDPOINT}"
  else
    echo "       Static website : DISABLED"
    echo "       Run ./infrastructure/deploy.sh to enable."
  fi
fi

# ---- Check latest version -------------------------------------------------
echo ""
echo "[5/5] Checking latest version available..."

ARTIFACTS_URL=$(az storage account show \
  --name "${STORAGE_ACCOUNT}" \
  --query "primaryEndpoints.web" -o tsv 2>/dev/null) || true

if [ -n "$ARTIFACTS_URL" ]; then
  LATEST=$(curl -sf "${ARTIFACTS_URL}latest" 2>/dev/null) || true

  if [ -n "$LATEST" ]; then
    echo "       Latest version: ${LATEST}"

    # Check available archives
    echo "       Checking available archives..."
    for platform in linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64; do
      ext="tar.gz"
      if [[ "$platform" == windows-* ]]; then
        ext="zip"
      fi
      url="${ARTIFACTS_URL}dex-${LATEST}-${platform}.${ext}"
      if curl -sf --head "$url" &>/dev/null; then
        echo "         [OK] dex-${LATEST}-${platform}.${ext}"
      else
        echo "         [--] dex-${LATEST}-${platform}.${ext} (not found)"
      fi
    done
  else
    echo "       No 'latest' file found. Artifacts may not have been deployed yet."
    echo ""
    echo "  To deploy: ./deploy.sh"
  fi
else
  echo "       Could not determine artifacts URL."
fi

# ---- Check local config sync ----------------------------------------------
echo ""
echo "--------------------------------------------"
CONFIG_FILE="${SCRIPT_DIR}/config.sh"
if [ -f "$CONFIG_FILE" ]; then
  LOCAL_ACCOUNT=$(grep "^export STORAGE_ACCOUNT=" "$CONFIG_FILE" | cut -d'"' -f2)
  if [ "$LOCAL_ACCOUNT" = "$STORAGE_ACCOUNT" ]; then
    echo "  Local config.sh is in sync."
  else
    echo "  WARNING: Local config.sh references '${LOCAL_ACCOUNT}'"
    echo "           but Azure has '${STORAGE_ACCOUNT}'."
    echo "           Re-run ./infrastructure/deploy.sh to update."
  fi
else
  echo "  No local config.sh found."
  echo "  Run ./infrastructure/deploy.sh to generate it."
fi

echo ""
echo "============================================"

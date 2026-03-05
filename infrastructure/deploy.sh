#!/usr/bin/env bash
set -euo pipefail

###############################################################################
# deploy.sh - Deploy Azure infrastructure for dex
#
# Usage: ./infrastructure/deploy.sh [RESOURCE_GROUP]
#
# This script:
#   1. Creates the resource group if it doesn't exist
#   2. Deploys the Bicep template (storage account + Static Web App)
#   3. Retrieves deployment outputs
#   4. Enables static website hosting on the storage account
#   5. Retrieves the SWA deployment token
#   6. Patches staticwebapp.config.json with the tenant ID
#   7. Writes config.sh with exported variables for downstream scripts
###############################################################################

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

RESOURCE_GROUP="${1:-dex-artifacts-rg}"
LOCATION="eastus"
DEPLOYMENT_NAME="dex-artifacts-$(date +%Y%m%d%H%M%S)"

TEMPLATE_FILE="${SCRIPT_DIR}/main.bicep"
PARAMETERS_FILE="${SCRIPT_DIR}/parameters.json"
CONFIG_FILE="${SCRIPT_DIR}/config.sh"
SWA_CONFIG_FILE="${PROJECT_ROOT}/website/staticwebapp.config.json"

echo "============================================"
echo "  Dex - Azure Infrastructure Deploy"
echo "============================================"
echo ""
echo "Resource Group : ${RESOURCE_GROUP}"
echo "Location       : ${LOCATION}"
echo "Template       : ${TEMPLATE_FILE}"
echo "Parameters     : ${PARAMETERS_FILE}"
echo ""

# ---- Step 1: Ensure resource group exists ------------------------------------
echo "[1/5] Checking resource group '${RESOURCE_GROUP}'..."
if az group show --name "${RESOURCE_GROUP}" &>/dev/null; then
  echo "       Resource group already exists."
else
  echo "       Creating resource group '${RESOURCE_GROUP}' in '${LOCATION}'..."
  az group create \
    --name "${RESOURCE_GROUP}" \
    --location "${LOCATION}" \
    --output none
  echo "       Resource group created."
fi

# ---- Step 2: Deploy Bicep template ------------------------------------------
echo ""
echo "[2/5] Deploying Bicep template..."
echo "       Deployment name: ${DEPLOYMENT_NAME}"

az deployment group create \
  --resource-group "${RESOURCE_GROUP}" \
  --name "${DEPLOYMENT_NAME}" \
  --template-file "${TEMPLATE_FILE}" \
  --parameters "@${PARAMETERS_FILE}" \
  --output none

echo "       Deployment succeeded."

# ---- Step 3: Retrieve outputs ------------------------------------------------
echo ""
echo "[3/5] Retrieving deployment outputs..."

STORAGE_ACCOUNT=$(az deployment group show \
  --resource-group "${RESOURCE_GROUP}" \
  --name "${DEPLOYMENT_NAME}" \
  --query "properties.outputs.storageAccountName.value" \
  --output tsv)

ARTIFACTS_URL=$(az deployment group show \
  --resource-group "${RESOURCE_GROUP}" \
  --name "${DEPLOYMENT_NAME}" \
  --query "properties.outputs.artifactsUrl.value" \
  --output tsv)

BLOB_ENDPOINT=$(az deployment group show \
  --resource-group "${RESOURCE_GROUP}" \
  --name "${DEPLOYMENT_NAME}" \
  --query "properties.outputs.blobEndpoint.value" \
  --output tsv)

SWA_NAME=$(az deployment group show \
  --resource-group "${RESOURCE_GROUP}" \
  --name "${DEPLOYMENT_NAME}" \
  --query "properties.outputs.staticWebAppName.value" \
  --output tsv)

WEBSITE_URL=$(az deployment group show \
  --resource-group "${RESOURCE_GROUP}" \
  --name "${DEPLOYMENT_NAME}" \
  --query "properties.outputs.websiteUrl.value" \
  --output tsv)

TENANT_ID=$(az account show --query tenantId -o tsv)

echo "       Storage Account : ${STORAGE_ACCOUNT}"
echo "       Artifacts URL   : ${ARTIFACTS_URL}"
echo "       Blob Endpoint   : ${BLOB_ENDPOINT}"
echo "       Static Web App  : ${SWA_NAME}"
echo "       Website URL     : ${WEBSITE_URL}"
echo "       Tenant ID       : ${TENANT_ID}"

# ---- Step 4: Enable static website hosting on storage account ----------------
echo ""
echo "[4/7] Enabling static website hosting on storage account..."

az storage blob service-properties update \
  --account-name "${STORAGE_ACCOUNT}" \
  --static-website \
  --index-document "index.html" \
  --auth-mode key \
  --output none

echo "       Static website hosting enabled."

# ---- Step 5: Retrieve SWA deployment token -----------------------------------
echo ""
echo "[5/7] Retrieving Static Web App deployment token..."

DEPLOYMENT_TOKEN=$(az staticwebapp secrets list \
  --name "${SWA_NAME}" \
  --resource-group "${RESOURCE_GROUP}" \
  --query "properties.apiKey" \
  --output tsv)

echo "       Deployment token retrieved."

# ---- Step 6: Patch staticwebapp.config.json with tenant ID ------------------
echo ""
echo "[6/7] Patching staticwebapp.config.json with tenant ID..."

python3 - <<PYEOF
import json, re

path = "${SWA_CONFIG_FILE}"
tenant = "${TENANT_ID}"

with open(path, "r") as f:
    config = json.load(f)

def add_tenant(url):
    base = url.split("?")[0]
    params = url.split("?")[1] if "?" in url else ""
    params = re.sub(r"tenantId=[^&]*&?", "", params).strip("&")
    new_params = f"tenantId={tenant}"
    if params:
        new_params += f"&{params}"
    return f"{base}?{new_params}"

for route in config.get("routes", []):
    if "redirect" in route and "/.auth/login/aad" in route["redirect"]:
        route["redirect"] = add_tenant(route["redirect"])

overrides = config.get("responseOverrides", {})
if "401" in overrides and "redirect" in overrides["401"]:
    if "/.auth/login/aad" in overrides["401"]["redirect"]:
        overrides["401"]["redirect"] = add_tenant(overrides["401"]["redirect"])

with open(path, "w") as f:
    json.dump(config, f, indent=2)
    f.write("\n")

print("       staticwebapp.config.json updated with tenantId.")
PYEOF

# ---- Step 7: Write config.sh -------------------------------------------------
echo ""
echo "[7/7] Writing config to '${CONFIG_FILE}'..."

cat > "${CONFIG_FILE}" <<EOF
# Auto-generated by deploy.sh on $(date -u +"%Y-%m-%dT%H:%M:%SZ")
# Source this file to set artifact environment variables:
#   source infrastructure/config.sh

export STORAGE_ACCOUNT="${STORAGE_ACCOUNT}"
export CONTAINER_NAME='\$web'
export ARTIFACTS_URL="${ARTIFACTS_URL}"
export BLOB_ENDPOINT="${BLOB_ENDPOINT}"
export SWA_NAME="${SWA_NAME}"
export WEBSITE_URL="${WEBSITE_URL}"
export DEPLOYMENT_TOKEN="${DEPLOYMENT_TOKEN}"
export TENANT_ID="${TENANT_ID}"
EOF

echo "       Config written. Source it with:"
echo "         source ${CONFIG_FILE}"

echo ""
echo "============================================"
echo "  Deployment complete!"
echo "============================================"
echo ""
echo "  Artifacts URL : ${ARTIFACTS_URL}"
echo "  Website URL   : ${WEBSITE_URL}"
echo ""
echo "  NOTE: staticwebapp.config.json has been updated with your tenant ID."
echo "  Commit it and run website/deploy.sh to push the site."

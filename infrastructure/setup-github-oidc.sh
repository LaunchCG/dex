#!/usr/bin/env bash
set -euo pipefail

###############################################################################
# setup-github-oidc.sh - Configure Azure OIDC for GitHub Actions
#
# Prerequisites:
#   - Azure CLI authenticated with Owner or User Access Administrator role
#   - GitHub CLI authenticated with repo admin access
#
# This script:
#   1. Creates an Azure AD app registration
#   2. Creates a service principal
#   3. Adds federated identity credentials for GitHub Actions
#   4. Assigns Contributor role on the artifact resource group
#   5. Sets GitHub repository secrets
###############################################################################

GITHUB_ORG="launchcg"
GITHUB_REPO="dex"
APP_NAME="dex-artifacts-github-actions"
RESOURCE_GROUP="dex-artifacts-rg"

SUBSCRIPTION_ID=$(az account show --query id -o tsv)
TENANT_ID=$(az account show --query tenantId -o tsv)

echo "============================================"
echo "  OIDC Setup for GitHub Actions"
echo "============================================"
echo ""
echo "Subscription : ${SUBSCRIPTION_ID}"
echo "Tenant       : ${TENANT_ID}"
echo "GitHub Repo  : ${GITHUB_ORG}/${GITHUB_REPO}"
echo "Resource Grp : ${RESOURCE_GROUP}"
echo ""

# ---- Step 1: Create app registration ----------------------------------------
echo "[1/5] Creating Azure AD app registration '${APP_NAME}'..."

CLIENT_ID=$(az ad app list --display-name "${APP_NAME}" --query "[0].appId" -o tsv)

if [ -z "$CLIENT_ID" ] || [ "$CLIENT_ID" = "None" ]; then
  CLIENT_ID=$(az ad app create --display-name "${APP_NAME}" --query appId -o tsv)
  echo "       Created app: ${CLIENT_ID}"
else
  echo "       App already exists: ${CLIENT_ID}"
fi

# ---- Step 2: Create service principal ---------------------------------------
echo ""
echo "[2/5] Creating service principal..."

SP_ID=$(az ad sp list --filter "appId eq '${CLIENT_ID}'" --query "[0].id" -o tsv)

if [ -z "$SP_ID" ] || [ "$SP_ID" = "None" ]; then
  SP_ID=$(az ad sp create --id "${CLIENT_ID}" --query id -o tsv)
  echo "       Created service principal: ${SP_ID}"
else
  echo "       Service principal already exists: ${SP_ID}"
fi

# ---- Step 3: Add federated credentials --------------------------------------
echo ""
echo "[3/5] Adding federated identity credentials..."

# For pushes to main
EXISTING=$(az ad app federated-credential list --id "${CLIENT_ID}" --query "[?name=='github-main'].name" -o tsv)
if [ -z "$EXISTING" ]; then
  az ad app federated-credential create --id "${CLIENT_ID}" --parameters '{
    "name": "github-main",
    "issuer": "https://token.actions.githubusercontent.com",
    "subject": "repo:'"${GITHUB_ORG}/${GITHUB_REPO}"':ref:refs/heads/main",
    "audiences": ["api://AzureADTokenExchange"]
  }' --output none
  echo "       Added credential for main branch"
else
  echo "       Credential for main branch already exists"
fi

# For production environment
EXISTING=$(az ad app federated-credential list --id "${CLIENT_ID}" --query "[?name=='github-environment-production'].name" -o tsv)
if [ -z "$EXISTING" ]; then
  az ad app federated-credential create --id "${CLIENT_ID}" --parameters '{
    "name": "github-environment-production",
    "issuer": "https://token.actions.githubusercontent.com",
    "subject": "repo:'"${GITHUB_ORG}/${GITHUB_REPO}"':environment:production",
    "audiences": ["api://AzureADTokenExchange"]
  }' --output none
  echo "       Added credential for production environment"
else
  echo "       Credential for production environment already exists"
fi

# ---- Step 4: Assign Contributor role ----------------------------------------
echo ""
echo "[4/5] Assigning Contributor role on '${RESOURCE_GROUP}'..."

RG_ID=$(az group show --name "${RESOURCE_GROUP}" --query id -o tsv)

EXISTING_ROLE=$(az role assignment list \
  --assignee "${SP_ID}" \
  --scope "${RG_ID}" \
  --role "Contributor" \
  --query "[0].id" -o tsv 2>/dev/null)

if [ -z "$EXISTING_ROLE" ]; then
  az role assignment create \
    --assignee "${SP_ID}" \
    --role "Contributor" \
    --scope "${RG_ID}" \
    --output none
  echo "       Role assigned."
else
  echo "       Role already assigned."
fi

# ---- Step 5: Set GitHub secrets ---------------------------------------------
echo ""
echo "[5/5] Setting GitHub repository secrets..."

gh secret set AZURE_CLIENT_ID --body "${CLIENT_ID}" --repo "${GITHUB_ORG}/${GITHUB_REPO}"
gh secret set AZURE_TENANT_ID --body "${TENANT_ID}" --repo "${GITHUB_ORG}/${GITHUB_REPO}"
gh secret set AZURE_SUBSCRIPTION_ID --body "${SUBSCRIPTION_ID}" --repo "${GITHUB_ORG}/${GITHUB_REPO}"

echo "       Secrets set."

echo ""
echo "============================================"
echo "  Setup complete!"
echo "============================================"
echo ""
echo "GitHub Actions will now authenticate via OIDC for:"
echo "  - Pushes to main branch"
echo "  - Production environment deployments"
echo ""
echo "No secrets to rotate - federated credentials are token-based."

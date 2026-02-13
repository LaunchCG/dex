# Azure OIDC Setup for GitHub Actions

This document describes how to configure Azure OIDC authentication so that GitHub Actions can deploy dex artifacts to Azure Blob Storage without storing long-lived credentials.

## Prerequisites

- **Azure CLI** authenticated with **Owner** or **User Access Administrator** role on the subscription
- **GitHub CLI** (`gh`) authenticated with admin access to `launchcg/dex`
- The Azure infrastructure must already be deployed (resource group `dex-artifacts-rg` must exist)

## Who Can Do This

You need one of the following Azure roles:
- **Owner** on the subscription or resource group
- **User Access Administrator** on the subscription or resource group

This is required to:
- Create Azure AD app registrations
- Create service principals
- Assign roles (Contributor) on the resource group

## Setup

### 1. Deploy Infrastructure First

If not already done:

```bash
./infrastructure/deploy.sh
```

### 2. Run the OIDC Setup Script

```bash
./infrastructure/setup-github-oidc.sh
```

The script will:
1. Create an Azure AD app registration (`dex-artifacts-github-actions`)
2. Create a service principal for the app
3. Add federated identity credentials for:
   - `github-main`: Pushes to the `main` branch
   - `github-environment-production`: The `production` GitHub environment
4. Assign the **Contributor** role on `dex-artifacts-rg`
5. Set GitHub repository secrets:
   - `AZURE_CLIENT_ID`
   - `AZURE_TENANT_ID`
   - `AZURE_SUBSCRIPTION_ID`

### 3. Create the GitHub Environment

In the GitHub repository settings:
1. Go to **Settings > Environments**
2. Create a new environment named **production**
3. Optionally add protection rules (required reviewers, etc.)

## Verification

After setup, verify the configuration:

```bash
# Check the app registration exists
az ad app list --display-name "dex-artifacts-github-actions" --query "[].appId" -o tsv

# Check the service principal
az ad sp list --filter "displayName eq 'dex-artifacts-github-actions'" --query "[].id" -o tsv

# Check federated credentials
CLIENT_ID=$(az ad app list --display-name "dex-artifacts-github-actions" --query "[0].appId" -o tsv)
az ad app federated-credential list --id "$CLIENT_ID" --query "[].{name:name, subject:subject}" -o table

# Check role assignment
az role assignment list --resource-group dex-artifacts-rg --query "[?principalType=='ServicePrincipal'].{role:roleDefinitionName, principal:principalId}" -o table
```

## Manual Steps (If Script Fails)

If the automated script encounters issues, you can set up OIDC manually:

### Create App Registration

```bash
az ad app create --display-name "dex-artifacts-github-actions"
```

### Create Service Principal

```bash
CLIENT_ID=$(az ad app list --display-name "dex-artifacts-github-actions" --query "[0].appId" -o tsv)
az ad sp create --id "$CLIENT_ID"
```

### Add Federated Credentials

```bash
# For main branch
az ad app federated-credential create --id "$CLIENT_ID" --parameters '{
  "name": "github-main",
  "issuer": "https://token.actions.githubusercontent.com",
  "subject": "repo:launchcg/dex:ref:refs/heads/main",
  "audiences": ["api://AzureADTokenExchange"]
}'

# For production environment
az ad app federated-credential create --id "$CLIENT_ID" --parameters '{
  "name": "github-environment-production",
  "issuer": "https://token.actions.githubusercontent.com",
  "subject": "repo:launchcg/dex:environment:production",
  "audiences": ["api://AzureADTokenExchange"]
}'
```

### Assign Role

```bash
SP_ID=$(az ad sp list --filter "appId eq '$CLIENT_ID'" --query "[0].id" -o tsv)
RG_ID=$(az group show --name "dex-artifacts-rg" --query id -o tsv)
az role assignment create --assignee "$SP_ID" --role "Contributor" --scope "$RG_ID"
```

### Set GitHub Secrets

```bash
gh secret set AZURE_CLIENT_ID --body "$CLIENT_ID" --repo launchcg/dex
gh secret set AZURE_TENANT_ID --body "$(az account show --query tenantId -o tsv)" --repo launchcg/dex
gh secret set AZURE_SUBSCRIPTION_ID --body "$(az account show --query id -o tsv)" --repo launchcg/dex
```

## How It Works

When a `v*` tag is pushed:
1. The `release.yml` workflow triggers
2. GitHub Actions requests an OIDC token from GitHub's token service
3. The token includes claims about the repo, branch, and environment
4. Azure AD validates the token against the federated credentials
5. If valid, Azure issues a short-lived access token
6. The workflow uses this token to upload artifacts to Blob Storage

No long-lived secrets are stored — the trust relationship is based on the OIDC federation.

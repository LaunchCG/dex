// Azure Bicep template for dex artifact distribution
// Deploys:
//   - Storage Account with static website hosting for binary artifacts + install scripts
//   - Azure Static Web App for the public-facing website (index.html, docs.html) with AAD auth

@description('Project name used to derive resource names')
param projectName string

@description('Deployment environment')
param environment string = 'production'

@description('Azure region for the storage account')
param location string = resourceGroup().location

@description('Azure region for the Static Web App (SWA has limited region support)')
@allowed(['eastus2', 'centralus', 'westus2', 'westeurope', 'eastasia', 'eastus', 'northeurope', 'southeastasia'])
param swaLocation string = 'eastus2'

// Derive a storage account name: remove hyphens, lowercase, max 24 chars
var cleanName = replace(replace('${projectName}${environment}', '-', ''), '_', '')
var storageAccountName = toLower(take(cleanName, 24))

resource storageAccount 'Microsoft.Storage/storageAccounts@2023-05-01' = {
  name: storageAccountName
  location: location
  kind: 'StorageV2'
  sku: {
    name: 'Standard_LRS'
  }
  properties: {
    accessTier: 'Hot'
    supportsHttpsTrafficOnly: true
    minimumTlsVersion: 'TLS1_2'
    allowBlobPublicAccess: true
  }
}

resource blobServices 'Microsoft.Storage/storageAccounts/blobServices@2023-05-01' = {
  parent: storageAccount
  name: 'default'
  properties: {
    isVersioningEnabled: true
  }
}

// The $web container for static website hosting with public read access
resource webContainer 'Microsoft.Storage/storageAccounts/blobServices/containers@2023-05-01' = {
  parent: blobServices
  name: '$web'
  properties: {
    publicAccess: 'Blob'
  }
}

// Note: Static website hosting (index document = index.html) is enabled
// via 'az storage blob service-properties update' in deploy.sh because
// Bicep does not natively expose the staticWebsite property on storage accounts.

// ── Static Web App (website with AAD auth) ──────────────────────────────────
// Serves index.html and docs.html behind Azure AD authentication.
// Auth is temporary; remove allowedRoles from staticwebapp.config.json to go public.
// Install scripts and binaries remain on blob storage (always public).

var websiteAppName = 'dex-website-${environment}'

resource staticWebApp 'Microsoft.Web/staticSites@2023-01-01' = {
  name: websiteAppName
  location: swaLocation
  sku: {
    name: 'Free'
    tier: 'Free'
  }
  properties: {
    buildProperties: {
      // GitHub Actions handles deployment; suppress auto-generated workflow
      skipGithubActionWorkflowGeneration: true
    }
  }
}

@description('Name of the deployed storage account')
output storageAccountName string = storageAccount.name

@description('Static website primary endpoint URL for artifacts')
output artifactsUrl string = storageAccount.properties.primaryEndpoints.web

@description('Blob service endpoint')
output blobEndpoint string = storageAccount.properties.primaryEndpoints.blob

@description('Name of the Static Web App')
output staticWebAppName string = staticWebApp.name

@description('Default hostname of the Static Web App (no protocol)')
output defaultHostname string = staticWebApp.properties.defaultHostname

@description('Full HTTPS URL of the website')
output websiteUrl string = 'https://${staticWebApp.properties.defaultHostname}'

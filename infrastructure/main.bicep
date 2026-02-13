// Azure Bicep template for dex artifact distribution
// Deploys a Storage Account with static website hosting for binary artifacts

@description('Project name used to derive resource names')
param projectName string

@description('Deployment environment')
param environment string = 'production'

@description('Azure region for resources')
param location string = resourceGroup().location

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

@description('Name of the deployed storage account')
output storageAccountName string = storageAccount.name

@description('Static website primary endpoint URL for artifacts')
output artifactsUrl string = storageAccount.properties.primaryEndpoints.web

@description('Blob service endpoint')
output blobEndpoint string = storageAccount.properties.primaryEndpoints.blob

# install.ps1 - Install dex binary on Windows
#
# Usage:
#   irm <url>/install.ps1 | iex
#   .\install.ps1 -Version 0.1.0

param(
    [string]$Version = ""
)

$ErrorActionPreference = "Stop"

# Ensure TLS 1.2 is enabled (required by Azure, not default on older PowerShell)
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

$ArtifactsUrl = "https://dexartifactsproduction.z13.web.core.windows.net"
$InstallDir = "$HOME\.bin"
$BinaryName = "dex.exe"

Write-Host "============================================"
Write-Host "  Dex Installer for Windows"
Write-Host "============================================"
Write-Host ""

# Fetch latest version if not specified
if (-not $Version) {
    Write-Host "Fetching latest version..."
    try {
        $response = Invoke-WebRequest -Uri "$ArtifactsUrl/latest" -UseBasicParsing
        $Version = [System.Text.Encoding]::UTF8.GetString($response.Content).Trim()
    } catch {
        Write-Host "ERROR: Could not fetch latest version from $ArtifactsUrl/latest" -ForegroundColor Red
        Write-Host "Details: $_" -ForegroundColor Red
        exit 1
    }
}

$Archive = "dex-$Version-windows-amd64.zip"
$DownloadUrl = "$ArtifactsUrl/$Archive"

Write-Host "Installing dex $Version (windows/amd64)..."

# Create temp directory
$TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) "dex-install-$(Get-Random)"
New-Item -ItemType Directory -Path $TmpDir -Force | Out-Null

try {
    # Download
    Write-Host "Downloading $DownloadUrl..."
    try {
        Invoke-WebRequest -Uri $DownloadUrl -OutFile "$TmpDir\$Archive" -UseBasicParsing
    } catch {
        Write-Error "Download failed. Check that version '$Version' exists for windows/amd64."
        exit 1
    }

    # Extract
    Write-Host "Extracting..."
    Expand-Archive -Path "$TmpDir\$Archive" -DestinationPath $TmpDir -Force

    # Install
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    Copy-Item "$TmpDir\$BinaryName" "$InstallDir\$BinaryName" -Force
    Write-Host "Installed to $InstallDir\$BinaryName"

    # Verify
    try {
        $versionOutput = & "$InstallDir\$BinaryName" version 2>&1
        Write-Host ""
        Write-Host $versionOutput
    } catch {
        Write-Host "Warning: Could not verify installation."
    }

    # Add to PATH if needed
    $UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    $PathUpdated = $false
    if ($UserPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable(
            "PATH",
            "$InstallDir;$UserPath",
            "User"
        )
        $PathUpdated = $true
    }

    # Always show summary
    Write-Host ""
    Write-Host "============================================"
    Write-Host "  Installation complete!"
    Write-Host "============================================"
    Write-Host ""
    Write-Host "  Binary:  $InstallDir\$BinaryName"
    Write-Host ""
    if ($PathUpdated) {
        Write-Host "  PATH:    $InstallDir has been added to your user PATH."
        Write-Host ""
        Write-Host "  IMPORTANT: Restart your terminal for the PATH change to take effect."
        Write-Host "  Then run 'dex version' to verify."
    } elseif ($env:PATH -like "*$InstallDir*") {
        Write-Host "  PATH:    $InstallDir is already in your PATH. You're all set!"
    } else {
        Write-Host "  PATH:    $InstallDir is in your user PATH but not in this session."
        Write-Host ""
        Write-Host "  Restart your terminal, or run:"
        Write-Host "    `$env:PATH = `"$InstallDir;`$env:PATH`""
    }
    Write-Host ""
} finally {
    # Cleanup
    Remove-Item -Path $TmpDir -Recurse -Force -ErrorAction SilentlyContinue
}

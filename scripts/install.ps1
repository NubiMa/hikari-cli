# Hikari Windows Installer (PowerShell)
# Usage (run in an elevated or user PowerShell):
#   iwr -useb https://raw.githubusercontent.com/NubiMa/hikari-cli/main/scripts/install.ps1 | iex
#
# Or download and run manually:
#   Invoke-WebRequest -Uri "..." -OutFile install.ps1; .\install.ps1

[CmdletBinding()]
param(
    [string]$InstallDir = "$env:LOCALAPPDATA\Programs\hikari",
    [string]$Repo      = "NubiMa/hikari-cli",
    [string]$Binary    = "hikari.exe"
)

$ErrorActionPreference = "Stop"

Write-Host "=== Hikari Installer (Windows) ===" -ForegroundColor Cyan

# ── Platform detection ─────────────────────────────────────────────────────────
$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64"   { "amd64"  }
    "ARM64"   { "arm64"  }
    "x86"     { "386"    }
    default   { throw "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
}
Write-Host "Detected platform: windows_$arch"

# ── Fetch latest release ───────────────────────────────────────────────────────
try {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" `
        -Headers @{ Accept = "application/vnd.github+json" } `
        -TimeoutSec 15
    $tag     = $release.tag_name        # e.g. "v1.2.3"
    $version = $tag.TrimStart("v")      # e.g. "1.2.3"
} catch {
    Write-Warning "Could not fetch latest release: $_"
    Write-Host ""
    Write-Host "No releases published yet or no network access." -ForegroundColor Yellow
    Write-Host "Please build from source:"
    Write-Host "  git clone https://github.com/$Repo.git hikari-cli"
    Write-Host "  cd hikari-cli && go build -o hikari.exe ./cmd/hikari"
    exit 1
}

Write-Host "Latest release: $tag"

# ── Find matching asset ────────────────────────────────────────────────────────
$assetName = "hikari_${version}_windows_${arch}.zip"
$asset = $release.assets | Where-Object { $_.name -eq $assetName } | Select-Object -First 1

if (-not $asset) {
    Write-Warning "No prebuilt binary found for $assetName"
    Write-Host ""
    Write-Host "Please download manually from:"
    Write-Host "  https://github.com/$Repo/releases/tag/$tag"
    exit 1
}

# ── Download ───────────────────────────────────────────────────────────────────
$tmp = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path "$($_.FullName)_dir" }
try {
    $zipPath = Join-Path $tmp.FullName "$assetName"
    Write-Host "Downloading $($asset.browser_download_url)..."
    Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $zipPath -TimeoutSec 120

    # ── Extract ────────────────────────────────────────────────────────────────
    $extractDir = Join-Path $tmp.FullName "extracted"
    Expand-Archive -Path $zipPath -DestinationPath $extractDir -Force

    # Find the exe (it may be in a subdirectory)
    $exeFile = Get-ChildItem -Recurse -Path $extractDir -Filter "hikari.exe" | Select-Object -First 1
    if (-not $exeFile) {
        throw "hikari.exe not found in archive"
    }

    # ── Install ────────────────────────────────────────────────────────────────
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }
    Copy-Item -Path $exeFile.FullName -Destination (Join-Path $InstallDir $Binary) -Force

    Write-Host ""
    Write-Host "✓ Hikari $tag installed to $InstallDir\$Binary" -ForegroundColor Green

} finally {
    Remove-Item -Recurse -Force $tmp.FullName -ErrorAction SilentlyContinue
}

# ── Add to PATH ────────────────────────────────────────────────────────────────
$userPath = [System.Environment]::GetEnvironmentVariable("PATH", "User")
if ($userPath -notlike "*$InstallDir*") {
    [System.Environment]::SetEnvironmentVariable(
        "PATH",
        "$userPath;$InstallDir",
        "User"
    )
    Write-Host "  Added $InstallDir to your user PATH." -ForegroundColor Cyan
    Write-Host "  Restart your terminal for PATH changes to take effect."
} else {
    Write-Host "  $InstallDir is already in your PATH."
}

Write-Host ""
Write-Host "Run 'hikari --init' to create your config file, then start with 'hikari'."

<#
.SYNOPSIS
    Installs aws-tui on Windows from GitHub Releases.

.DESCRIPTION
    Detects the architecture, downloads the release .zip archive, verifies the
    SHA-256 checksum, extracts the binary into the install directory and adds it to
    the user PATH if needed.

.PARAMETER Version
    Version to install (e.g. v1.2.3). Default: latest release.

.PARAMETER InstallDir
    Install directory. Default: %LOCALAPPDATA%\aws-tui\bin.

.EXAMPLE
    irm https://raw.githubusercontent.com/VeugDamien/aws-tui/main/scripts/install.ps1 | iex

.EXAMPLE
    .\install.ps1 -Version v1.2.3 -InstallDir "C:\Tools\aws-tui"
#>

[CmdletBinding()]
param(
    [string]$Version = "",
    [string]$InstallDir = "$env:LOCALAPPDATA\aws-tui\bin"
)

$ErrorActionPreference = "Stop"
$Owner  = "VeugDamien"
$Repo   = "aws-tui"
$Binary = "aws-tui"

function Write-Info  { param($m) Write-Host "==> $m" -ForegroundColor Cyan }
function Write-Warn  { param($m) Write-Host "warning: $m" -ForegroundColor Yellow }

# --- Architecture -----------------------------------------------------------

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { "amd64" }
    "ARM64" { "arm64" }
    default { throw "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
}
if ($arch -eq "arm64") {
    Write-Warn "No Windows arm64 binary published; trying amd64 (emulation)."
    $arch = "amd64"
}

# --- Version ----------------------------------------------------------------

if ([string]::IsNullOrEmpty($Version)) {
    Write-Info "Looking up the latest version..."
    $api = "https://api.github.com/repos/$Owner/$Repo/releases/latest"
    $rel = Invoke-RestMethod -Uri $api -Headers @{ "User-Agent" = "aws-tui-installer" }
    $Version = $rel.tag_name
    if ([string]::IsNullOrEmpty($Version)) {
        throw "Unable to determine the latest version (no release published?)."
    }
}
$num = $Version.TrimStart("v")

# --- Download ---------------------------------------------------------------

$archive = "$($Binary)_$($num)_windows_$($arch).zip"
$base    = "https://github.com/$Owner/$Repo/releases/download/$Version"
$url     = "$base/$archive"

$tmp = Join-Path $env:TEMP ("aws-tui-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tmp -Force | Out-Null
try {
    $zipPath = Join-Path $tmp $archive
    Write-Info "Downloading $archive ($Version)..."
    Invoke-WebRequest -Uri $url -OutFile $zipPath -UseBasicParsing

    # Verify the checksum if available.
    $sumPath = Join-Path $tmp "checksums.txt"
    try {
        Invoke-WebRequest -Uri "$base/checksums.txt" -OutFile $sumPath -UseBasicParsing
        Write-Info "Verifying SHA-256 checksum..."
        $expected = (Select-String -Path $sumPath -Pattern ([regex]::Escape($archive)) |
                     Select-Object -First 1).Line -split '\s+' | Select-Object -First 1
        if ($expected) {
            $actual = (Get-FileHash -Path $zipPath -Algorithm SHA256).Hash.ToLower()
            if ($actual -ne $expected.ToLower()) {
                throw "Invalid checksum (expected $expected, got $actual)."
            }
        }
    } catch {
        Write-Warn "checksums.txt unavailable or not verified: $($_.Exception.Message)"
    }

    # --- Extraction + installation ------------------------------------------

    Write-Info "Extracting..."
    Expand-Archive -Path $zipPath -DestinationPath $tmp -Force
    $src = Get-ChildItem -Path $tmp -Recurse -Filter "$Binary.exe" | Select-Object -First 1
    if (-not $src) { throw "Binary $Binary.exe not found in the archive." }

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Copy-Item -Path $src.FullName -Destination (Join-Path $InstallDir "$Binary.exe") -Force
    Write-Info "Installed: $(Join-Path $InstallDir "$Binary.exe")"

    # --- User PATH ----------------------------------------------------------

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable("Path", "$userPath;$InstallDir", "User")
        Write-Info "Added to the user PATH. Reopen your terminal to use it."
    }
}
finally {
    Remove-Item -Path $tmp -Recurse -Force -ErrorAction SilentlyContinue
}

# --- Post-install checks ----------------------------------------------------

if (-not (Get-Command aws -ErrorAction SilentlyContinue)) {
    Write-Warn "AWS CLI v2 ('aws') not found: required for logins and SSM sessions."
}
if (-not (Get-Command session-manager-plugin -ErrorAction SilentlyContinue)) {
    Write-Warn "session-manager-plugin not found: required for SSM sessions/port-forwards."
}

& (Join-Path $InstallDir "$Binary.exe") --version
Write-Info "Done. Run: $Binary"

<#
.SYNOPSIS
    Installe aws-tui sur Windows depuis les Releases GitHub.

.DESCRIPTION
    Détecte l'architecture, télécharge l'archive .zip de la release, vérifie le
    checksum SHA-256, extrait le binaire dans le répertoire d'installation et
    l'ajoute au PATH utilisateur si nécessaire.

.PARAMETER Version
    Version à installer (ex. v1.2.3). Par défaut : dernière release.

.PARAMETER InstallDir
    Répertoire d'installation. Par défaut : %LOCALAPPDATA%\aws-tui\bin.

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
function Write-Warn  { param($m) Write-Host "attention: $m" -ForegroundColor Yellow }

# --- Architecture -----------------------------------------------------------

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { "amd64" }
    "ARM64" { "arm64" }
    default { throw "Architecture non supportée : $env:PROCESSOR_ARCHITECTURE" }
}
if ($arch -eq "arm64") {
    Write-Warn "Aucun binaire Windows arm64 publié ; tentative avec amd64 (émulation)."
    $arch = "amd64"
}

# --- Version ----------------------------------------------------------------

if ([string]::IsNullOrEmpty($Version)) {
    Write-Info "Recherche de la dernière version..."
    $api = "https://api.github.com/repos/$Owner/$Repo/releases/latest"
    $rel = Invoke-RestMethod -Uri $api -Headers @{ "User-Agent" = "aws-tui-installer" }
    $Version = $rel.tag_name
    if ([string]::IsNullOrEmpty($Version)) {
        throw "Impossible de déterminer la dernière version (aucune release publiée ?)."
    }
}
$num = $Version.TrimStart("v")

# --- Téléchargement ---------------------------------------------------------

$archive = "$($Binary)_$($num)_windows_$($arch).zip"
$base    = "https://github.com/$Owner/$Repo/releases/download/$Version"
$url     = "$base/$archive"

$tmp = Join-Path $env:TEMP ("aws-tui-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tmp -Force | Out-Null
try {
    $zipPath = Join-Path $tmp $archive
    Write-Info "Téléchargement de $archive ($Version)..."
    Invoke-WebRequest -Uri $url -OutFile $zipPath -UseBasicParsing

    # Vérification du checksum si disponible.
    $sumPath = Join-Path $tmp "checksums.txt"
    try {
        Invoke-WebRequest -Uri "$base/checksums.txt" -OutFile $sumPath -UseBasicParsing
        Write-Info "Vérification du checksum SHA-256..."
        $expected = (Select-String -Path $sumPath -Pattern ([regex]::Escape($archive)) |
                     Select-Object -First 1).Line -split '\s+' | Select-Object -First 1
        if ($expected) {
            $actual = (Get-FileHash -Path $zipPath -Algorithm SHA256).Hash.ToLower()
            if ($actual -ne $expected.ToLower()) {
                throw "Checksum invalide (attendu $expected, obtenu $actual)."
            }
        }
    } catch {
        Write-Warn "checksums.txt indisponible ou non vérifié : $($_.Exception.Message)"
    }

    # --- Extraction + installation ------------------------------------------

    Write-Info "Extraction..."
    Expand-Archive -Path $zipPath -DestinationPath $tmp -Force
    $src = Get-ChildItem -Path $tmp -Recurse -Filter "$Binary.exe" | Select-Object -First 1
    if (-not $src) { throw "Binaire $Binary.exe introuvable dans l'archive." }

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Copy-Item -Path $src.FullName -Destination (Join-Path $InstallDir "$Binary.exe") -Force
    Write-Info "Installé : $(Join-Path $InstallDir "$Binary.exe")"

    # --- PATH utilisateur ---------------------------------------------------

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable("Path", "$userPath;$InstallDir", "User")
        Write-Info "Ajouté au PATH utilisateur. Rouvrez votre terminal pour en profiter."
    }
}
finally {
    Remove-Item -Path $tmp -Recurse -Force -ErrorAction SilentlyContinue
}

# --- Vérifications post-installation ----------------------------------------

if (-not (Get-Command aws -ErrorAction SilentlyContinue)) {
    Write-Warn "AWS CLI v2 ('aws') introuvable : requis pour les connexions et sessions SSM."
}
if (-not (Get-Command session-manager-plugin -ErrorAction SilentlyContinue)) {
    Write-Warn "session-manager-plugin introuvable : requis pour les sessions/port-forwards SSM."
}

& (Join-Path $InstallDir "$Binary.exe") --version
Write-Info "Terminé. Lancez : $Binary"

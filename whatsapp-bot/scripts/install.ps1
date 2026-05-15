#Requires -RunAsAdministrator

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# ── Configuration ────────────────────────────────────────────────────────────
$REPO_URL    = "https://github.com/zaelix/whatsapp-bot.git"
$INSTALL_DIR = "$env:ProgramData\zaelix"
$SRC_DIR     = "$INSTALL_DIR\src"
$BIN_PATH    = "$INSTALL_DIR\zaelix.exe"
# ─────────────────────────────────────────────────────────────────────────────

function Write-Step($msg) { Write-Host "`n==> $msg" -ForegroundColor Cyan }
function Write-Ok($msg)   { Write-Host "    $msg"   -ForegroundColor Green }
function Write-Err($msg)  { Write-Host "`n    ERROR: $msg" -ForegroundColor Red; exit 1 }

# ── Require admin ─────────────────────────────────────────────────────────────
Write-Step "Checking prerequisites"

if (-not ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole(
    [Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Err "This script must be run as Administrator."
}

# git
if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Write-Err "git is not installed. Please install Git for Windows: https://git-scm.com"
}
Write-Ok "Git: $(git --version)"

# go
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Err "Go is not installed. Please install it from: https://go.dev/dl"
}
Write-Ok "Go: $(go version)"

# ── Setup install dir ─────────────────────────────────────────────────────────
Write-Step "Preparing installation directory"
New-Item -ItemType Directory -Force -Path $INSTALL_DIR | Out-Null
Write-Ok "Install dir: $INSTALL_DIR"

# ── Clone or update repo ──────────────────────────────────────────────────────
Write-Step "Setting up source"
if (Test-Path "$SRC_DIR\.git") {
    Write-Ok "Updating existing clone"
    git -C $SRC_DIR pull --ff-only
} else {
    Write-Ok "Cloning $REPO_URL"
    git clone $REPO_URL $SRC_DIR
}

# ── Build ─────────────────────────────────────────────────────────────────────
Write-Step "Building zaelix"
$env:CGO_ENABLED = "0"

Push-Location $SRC_DIR
go build "-ldflags=-s -w -X main.sourceDir=$SRC_DIR" -trimpath -o $BIN_PATH .
Pop-Location

if (-not (Test-Path $BIN_PATH)) {
    Write-Err "Build failed: binary not found at $BIN_PATH"
}
Write-Ok "Binary: $BIN_PATH"

# ── Update PATH ───────────────────────────────────────────────────────────────
Write-Step "Updating system PATH"
$syspath = [System.Environment]::GetEnvironmentVariable("PATH", "Machine")
$normDir = $INSTALL_DIR.TrimEnd('\')

if (($syspath -split ';') -notcontains $normDir) {
    [System.Environment]::SetEnvironmentVariable("PATH", "$syspath;$normDir", "Machine")
    $env:PATH += ";$normDir"
    Write-Ok "Added $normDir to system PATH"
} else {
    Write-Ok "Already in PATH"
}

# ── Done ──────────────────────────────────────────────────────────────────────
Write-Host ""
Write-Host "  Ortsax is installed!" -ForegroundColor Green
Write-Host ""
Write-Host "  Run with      zaelix --phone-number <number>"
Write-Host "  Update with   zaelix --update"
Write-Host "  Sessions      zaelix --list-sessions"
Write-Host "                zaelix --delete-session <phone>"
Write-Host "                zaelix --reset-session  <phone>"
Write-Host ""
Write-Host "  Restart your terminal for PATH changes to take effect." -ForegroundColor Yellow

#!/usr/bin/env bash
# Ortsax installer for macOS.
# Must be run as root (sudo bash install-mac.sh).
set -euo pipefail

# ── Configuration ────────────────────────────────────────────────────────────
REPO_URL="https://github.com/zaelix/whatsapp-bot.git"
INSTALL_DIR="/opt/zaelix"
SRC_DIR="$INSTALL_DIR/src"
BIN_PATH="/usr/local/bin/zaelix"
GO_FALLBACK="1.25.0"
GOROOT="/usr/local/go"
# ─────────────────────────────────────────────────────────────────────────────

CYAN='\033[0;36m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; RESET='\033[0m'

step() { echo; echo -e "${CYAN}==> $*${RESET}"; }
ok()   { echo -e "    ${GREEN}$*${RESET}"; }
warn() { echo -e "    ${YELLOW}$*${RESET}"; }
err()  { echo -e "\n    ${RED}ERROR: $*${RESET}" >&2; exit 1; }

# ── Require root ──────────────────────────────────────────────────────────────
if [ "$(id -u)" -ne 0 ]; then
    err "This script must be run as root. Try: sudo bash $0"
fi

# ── Detect architecture ───────────────────────────────────────────────────────
step "Detecting system architecture"
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)        GOARCH="amd64" ;;
    arm64|aarch64) GOARCH="arm64" ;;
    *) err "Unsupported architecture: $ARCH" ;;
esac
ok "Architecture: $GOARCH"

# ── Check / install Git ───────────────────────────────────────────────────────
step "Checking Git"
if ! command -v git &>/dev/null; then
    ok "Installing Xcode Command Line Tools (includes git)"
    xcode-select --install 2>/dev/null || true
    for i in $(seq 1 30); do
        command -v git &>/dev/null && break
        sleep 2
    done
    command -v git &>/dev/null || err "git installation failed. Please install Xcode Command Line Tools manually."
fi
ok "Git: $(git --version)"

# ── Check / install Go ────────────────────────────────────────────────────────
step "Checking Go"
if command -v brew &>/dev/null; then
    if brew list go &>/dev/null 2>&1; then
        ok "Go already installed via Homebrew: $(go version)"
    else
        ok "Installing Go via Homebrew"
        brew install go
    fi
elif command -v go &>/dev/null; then
    ok "Go already installed: $(go version)"
else
    step "Detecting latest Go version"
    GO_VERSION=$(curl -fsSL "https://go.dev/dl/?mode=json" 2>/dev/null \
        | grep -o '"version":"go[^"]*"' | head -1 \
        | grep -o '[0-9][0-9.]*' | head -1) || true
    GO_VERSION="${GO_VERSION:-$GO_FALLBACK}"

    step "Installing Go $GO_VERSION from go.dev"
    TARBALL="/tmp/go${GO_VERSION}.darwin-${GOARCH}.tar.gz"
    DL_URL="https://go.dev/dl/go${GO_VERSION}.darwin-${GOARCH}.tar.gz"
    ok "Downloading $DL_URL"
    curl -fsSL "$DL_URL" -o "$TARBALL"
    rm -rf "$GOROOT"
    tar -C /usr/local -xzf "$TARBALL"
    rm "$TARBALL"
    echo '/usr/local/go/bin' > /etc/paths.d/go
    export PATH="$PATH:/usr/local/go/bin"
    ok "Go $GO_VERSION installed"
fi

export PATH="$PATH:$GOROOT/bin"

# ── Clone or update repo ──────────────────────────────────────────────────────
step "Setting up source"
mkdir -p "$INSTALL_DIR"
mkdir -p /usr/local/bin
if [ -d "$SRC_DIR/.git" ]; then
    ok "Updating existing clone"
    git -C "$SRC_DIR" pull --ff-only
else
    ok "Cloning $REPO_URL"
    git clone "$REPO_URL" "$SRC_DIR"
fi

# ── Build ─────────────────────────────────────────────────────────────────────
step "Building zaelix"
CGO_ENABLED=0 go build \
    -ldflags="-s -w -X main.sourceDir=${SRC_DIR}" \
    -trimpath \
    -o "$BIN_PATH" \
    "$SRC_DIR"
chmod +x "$BIN_PATH"
ok "Binary: $BIN_PATH"

# ── Done ──────────────────────────────────────────────────────────────────────
echo
echo -e "${GREEN}  Ortsax is installed!${RESET}"
echo
echo "  Run with      zaelix --phone-number <number>"
echo "  Update with   zaelix --update"
echo "  Sessions      zaelix --list-sessions"
echo "                zaelix --delete-session <phone>"
echo "                zaelix --reset-session  <phone>"
echo
echo -e "${YELLOW}  Note: open a new terminal for PATH changes to take effect.${RESET}"


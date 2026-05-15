#!/usr/bin/env bash
# Ortsax installer for Linux.
# Must be run as root (or with sudo).
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
    x86_64)        GOARCH="amd64"   ;;
    aarch64|arm64) GOARCH="arm64"   ;;
    armv6l)        GOARCH="armv6l"  ;;
    i686)          GOARCH="386"     ;;
    *) err "Unsupported architecture: $ARCH" ;;
esac
ok "Architecture: $GOARCH"

# ── Check / install dependencies ─────────────────────────────────────────────
step "Checking dependencies"

if ! command -v git &>/dev/null; then
    ok "Installing git"
    if   command -v apt-get &>/dev/null; then apt-get update -qq && apt-get install -y -qq git
    elif command -v yum     &>/dev/null; then yum install -y -q git
    elif command -v dnf     &>/dev/null; then dnf install -y -q git
    elif command -v pacman  &>/dev/null; then pacman -Sy --noconfirm git
    else err "Cannot install git automatically. Please install it manually."
    fi
fi
ok "Git: $(git --version)"

if ! command -v curl &>/dev/null && ! command -v wget &>/dev/null; then
    if   command -v apt-get &>/dev/null; then apt-get install -y -qq curl
    elif command -v yum     &>/dev/null; then yum install -y -q curl
    fi
fi

# ── Check / install Go ────────────────────────────────────────────────────────
step "Checking Go"
if command -v go &>/dev/null; then
    ok "Go already installed: $(go version)"
else
    step "Detecting latest Go version"
    if command -v curl &>/dev/null; then
        GO_VERSION=$(curl -fsSL "https://go.dev/dl/?mode=json" 2>/dev/null \
            | grep -o '"version":"go[^"]*"' | head -1 \
            | grep -o '[0-9][0-9.]*' | head -1) || true
    elif command -v wget &>/dev/null; then
        GO_VERSION=$(wget -qO- "https://go.dev/dl/?mode=json" 2>/dev/null \
            | grep -o '"version":"go[^"]*"' | head -1 \
            | grep -o '[0-9][0-9.]*' | head -1) || true
    fi
    GO_VERSION="${GO_VERSION:-$GO_FALLBACK}"

    step "Installing Go $GO_VERSION"
    TARBALL="/tmp/go${GO_VERSION}.linux-${GOARCH}.tar.gz"
    DL_URL="https://go.dev/dl/go${GO_VERSION}.linux-${GOARCH}.tar.gz"
    ok "Downloading $DL_URL"
    if command -v curl &>/dev/null; then
        curl -fsSL "$DL_URL" -o "$TARBALL"
    else
        wget -q "$DL_URL" -O "$TARBALL"
    fi
    rm -rf "$GOROOT"
    tar -C /usr/local -xzf "$TARBALL"
    rm "$TARBALL"
    echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/go.sh
    chmod +x /etc/profile.d/go.sh
    export PATH="$PATH:/usr/local/go/bin"
    ok "Go $GO_VERSION installed"
fi

export PATH="$PATH:$GOROOT/bin"

# ── Clone or update repo ──────────────────────────────────────────────────────
step "Setting up source"
mkdir -p "$INSTALL_DIR"
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
echo -e "${YELLOW}  Note: open a new shell or run 'source /etc/profile.d/go.sh' if Go was just installed.${RESET}"


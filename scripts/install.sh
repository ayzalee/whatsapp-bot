#!/usr/bin/env bash
set -euo pipefail

BOT_NAME="whatsapp-bot"
REPO_URL="https://github.com/ayzalee/whatsapp-bot.git"
LOG_FILE="install.log"
INSTALL_SUCCESS=0

# Colors
if [ -t 1 ]; then
    CYAN='\033[36m'; GREEN='\033[32m'; YELLOW='\033[33m'; RED='\033[31m'
    BOLD='\033[1m'; NC='\033[0m'
else
    CYAN=''; GREEN=''; YELLOW=''; RED=''; BOLD=''; NC=''
fi

info()    { printf "${CYAN}[INFO]${NC} %s\n" "$1"; echo "[INFO] $1" >> "$LOG_FILE"; }
success() { printf "${GREEN}[OK]${NC} %s\n" "$1"; echo "[OK] $1" >> "$LOG_FILE"; }
warn()    { printf "${YELLOW}[WARN]${NC} %s\n" "$1"; echo "[WARN] $1" >> "$LOG_FILE"; }
error()   { printf "${RED}[ERROR]${NC} %s\n" "$1"; echo "[ERROR] $1" >> "$LOG_FILE"; }

spin() {
    local msg="$1"; shift
    printf "  - %s..." "$msg"
    if "$@" >> "$LOG_FILE" 2>&1; then
        printf "\r${GREEN}  + %s${NC}\n" "$msg"
        return 0
    else
        printf "\r${RED}  x %s${NC}\n" "$msg"
        return 1
    fi
}

cleanup() {
    if [ "${INSTALL_SUCCESS:-0}" -ne 1 ]; then
        error "Installation failed. See $LOG_FILE for details."
        tail -20 "$LOG_FILE" 2>/dev/null || true
    fi
}
trap cleanup EXIT

run_privileged() {
    if [ "$(id -u)" -ne 0 ] && command -v sudo > /dev/null 2>&1; then
        sudo "$@"
    else
        "$@"
    fi
}

detect_os() {
    local kernel; kernel="$(uname -s)"
    case "$kernel" in
        Linux)
            if [ -n "${TERMUX_VERSION:-}" ] || [ -d "/data/data/com.termux" ]; then
                OS_TYPE="termux"; PKG_MANAGER="pkg"
            elif [ -f /etc/os-release ]; then
                . /etc/os-release
                case "${ID:-}" in
                    debian|ubuntu|pop|linuxmint) PKG_MANAGER="apt" ;;
                    fedora|rhel|centos|rocky)
                        command -v dnf > /dev/null 2>&1 && PKG_MANAGER="dnf" || PKG_MANAGER="yum" ;;
                    arch|manjaro) PKG_MANAGER="pacman" ;;
                    alpine) PKG_MANAGER="apk" ;;
                    *) PKG_MANAGER="apt" ;;
                esac
                OS_TYPE="linux"
            fi ;;
        Darwin) OS_TYPE="darwin"; PKG_MANAGER="brew" ;;
        *) error "Unsupported OS: $kernel"; exit 1 ;;
    esac
    info "Detected: $OS_TYPE ($PKG_MANAGER)"
}

install_deps() {
    info "Installing dependencies..."
    case "$PKG_MANAGER" in
        pkg)
            spin "Updating packages" pkg update -y
            spin "Installing Go, git, ffmpeg, imagemagick" pkg install -y golang git ffmpeg imagemagick
            spin "Installing yt-dlp" pkg install -y python-yt-dlp
            ;;
        apt)
            spin "Updating packages" run_privileged apt-get update -y
            spin "Installing Go, git, ffmpeg, imagemagick" run_privileged apt-get install -y golang-go git ffmpeg imagemagick python3-pip
            spin "Installing yt-dlp" run_privileged pip3 install yt-dlp --break-system-packages 2>/dev/null || \
                run_privileged apt-get install -y yt-dlp 2>/dev/null || true
            ;;
        dnf|yum)
            spin "Installing dependencies" run_privileged "$PKG_MANAGER" install -y golang git ffmpeg ImageMagick python3-pip
            spin "Installing yt-dlp" run_privileged pip3 install yt-dlp || true
            ;;
        pacman)
            spin "Installing dependencies" run_privileged pacman -Sy --noconfirm go git ffmpeg imagemagick yt-dlp
            ;;
        apk)
            spin "Installing dependencies" run_privileged apk add --no-cache go git ffmpeg imagemagick py3-pip
            spin "Installing yt-dlp" run_privileged pip3 install yt-dlp || true
            ;;
        brew)
            spin "Installing dependencies" brew install go git ffmpeg imagemagick yt-dlp
            ;;
    esac
    success "Dependencies installed"
}

clone_repo() {
    if [ -d "$BOT_NAME/.git" ]; then
        info "Repo exists — pulling updates..."
        spin "Pulling latest" git -C "$BOT_NAME" pull --ff-only
    else
        spin "Cloning repository" git clone --depth 1 "$REPO_URL" "$BOT_NAME"
    fi
    success "Repository ready"
}

setup_env() {
    local env_file="$BOT_NAME/.env"
    if [ -f "$env_file" ]; then
        info ".env already exists — skipping"
        return 0
    fi
    cp "$BOT_NAME/.env.example" "$env_file"

    echo ""
    printf "${BOLD}Configure your bot:${NC}\n"

    printf "  Database URL (leave blank for SQLite): "
    read -r db_url
    [ -n "$db_url" ] && sed -i "s|DATABASE_URL=.*|DATABASE_URL=$db_url|" "$env_file"

    printf "  Bot prefix (default: .): "
    read -r prefix
    [ -n "$prefix" ] && sed -i "s|BOT_PREFIX=.*|BOT_PREFIX=$prefix|" "$env_file"

    printf "  Sudo phone number: "
    read -r sudo_num
    [ -n "$sudo_num" ] && sed -i "s|SUDO=.*|SUDO=$sudo_num|" "$env_file"

    success ".env configured"
}

build_bot() {
    cd "$BOT_NAME"
    info "Building bot..."
    spin "Downloading Go modules" go mod tidy
    spin "Building binary" go build -ldflags "-X main.sourceDir=$(pwd)" -o zaelix .
    cd ..
    success "Bot built successfully"
}

print_done() {
    echo ""
    printf "${GREEN}${BOLD}Installation complete!${NC}\n"
    echo ""
    printf "  Start the bot:\n"
    printf "    ${CYAN}cd $BOT_NAME && ./zaelix${NC}\n"
    echo ""
    printf "  Pair your WhatsApp:\n"
    printf "    ${CYAN}cd $BOT_NAME && ./zaelix --phone-number <your_number>${NC}\n"
    echo ""
}

# Main
> "$LOG_FILE"
detect_os
install_deps
clone_repo
setup_env
build_bot
INSTALL_SUCCESS=1
print_done

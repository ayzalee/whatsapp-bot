#!/usr/bin/env bash
set -euo pipefail

REPO_URL="https://github.com/ayzalee/whatsapp-bot.git"
LOG_FILE="install.log"
INSTALL_SUCCESS=0
OS_TYPE=""
PKG_MANAGER=""
BOT_DIR=""

if [ -t 1 ]; then
    CYAN='\033[36m'; GREEN='\033[32m'; YELLOW='\033[33m'; RED='\033[31m'
    BOLD='\033[1m'; NC='\033[0m'
else
    CYAN=''; GREEN=''; YELLOW=''; RED=''; BOLD=''; NC=''
fi

info()    { printf "${CYAN}[INFO]${NC} %s\n" "$1"; echo "[INFO] $1" >> "$LOG_FILE"; }
success() { printf "${GREEN}[OK]${NC} %s\n" "$1"; echo "[OK] $1" >> "$LOG_FILE"; }
warn()    { printf "${YELLOW}[WARN]${NC} %s\n" "$1"; echo "[WARN] $1" >> "$LOG_FILE"; }
error()   { printf "${RED}[ERROR]${NC} %s\n" "$1" >&2; echo "[ERROR] $1" >> "$LOG_FILE"; }

spin() {
    local msg="$1"; shift
    printf "  - %s..." "$msg"
    if "$@" >> "$LOG_FILE" 2>&1; then
        printf "\r${GREEN}  + %s${NC}\n" "$msg"
    else
        local rc=$?
        printf "\r${RED}  x %s${NC}\n" "$msg"
        return $rc
    fi
}

cleanup() {
    if [ "${INSTALL_SUCCESS:-0}" -ne 1 ]; then
        error "Installation failed. Last 20 lines of $LOG_FILE:"
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
                    debian|ubuntu|pop|linuxmint|kali) PKG_MANAGER="apt" ;;
                    fedora|rhel|centos|rocky|alma)
                        command -v dnf > /dev/null 2>&1 && PKG_MANAGER="dnf" || PKG_MANAGER="yum" ;;
                    arch|manjaro|endeavouros) PKG_MANAGER="pacman" ;;
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

add_swap() {
    [ "$OS_TYPE" = "termux" ] || [ "$OS_TYPE" = "darwin" ] && return 0
    local total_mem swap_exists
    total_mem=$(free -m 2>/dev/null | awk '/^Mem:/{print $2}' || echo 2000)
    swap_exists=$(free -m 2>/dev/null | awk '/^Swap:/{print $2}' || echo 0)
    if [ "$total_mem" -lt 1500 ] && [ "$swap_exists" -eq 0 ]; then
        warn "Low RAM (${total_mem}MB). Adding 2GB swap..."
        run_privileged fallocate -l 2G /swapfile 2>/dev/null || \
            run_privileged dd if=/dev/zero of=/swapfile bs=1M count=2048 >> "$LOG_FILE" 2>&1
        run_privileged chmod 600 /swapfile
        run_privileged mkswap /swapfile >> "$LOG_FILE" 2>&1
        run_privileged swapon /swapfile >> "$LOG_FILE" 2>&1
        echo '/swapfile none swap sw 0 0' | run_privileged tee -a /etc/fstab >> "$LOG_FILE" 2>&1
        success "Swap added"
    fi
}

install_deps() {
    info "Installing dependencies..."
    case "$PKG_MANAGER" in
        pkg)
            spin "Updating packages" pkg update -y
            spin "Installing deps" pkg install -y golang git ffmpeg imagemagick
            spin "Installing yt-dlp" pkg install -y python-yt-dlp
            ;;
        apt)
            spin "Updating packages" run_privileged apt-get update -y
            spin "Installing deps" run_privileged apt-get install -y golang-go git ffmpeg imagemagick python3-pip curl
            spin "Installing yt-dlp" run_privileged pip3 install -U yt-dlp --break-system-packages 2>/dev/null || true
            ;;
        dnf|yum)
            spin "Installing deps" run_privileged "$PKG_MANAGER" install -y golang git ffmpeg ImageMagick python3-pip curl
            spin "Installing yt-dlp" run_privileged pip3 install -U yt-dlp || true
            ;;
        pacman)
            spin "Installing deps" run_privileged pacman -Sy --noconfirm go git ffmpeg imagemagick yt-dlp curl
            ;;
        apk)
            spin "Installing deps" run_privileged apk add --no-cache go git ffmpeg imagemagick py3-pip curl
            spin "Installing yt-dlp" pip3 install -U yt-dlp || true
            ;;
        brew)
            spin "Installing deps" brew install go git ffmpeg imagemagick yt-dlp
            ;;
    esac
    success "Dependencies installed"
}

clone_repo() {
    if [ -d "$BOT_DIR/.git" ]; then
        info "Repo exists — pulling updates..."
        spin "Pulling latest" git -C "$BOT_DIR" pull --ff-only
    else
        spin "Cloning repository" git clone --depth 1 "$REPO_URL" "$BOT_DIR"
    fi
    success "Repository ready"
}

setup_env() {
    local env_file="$BOT_DIR/.env"
    [ -f "$env_file" ] && info ".env exists — skipping" && return 0
    cp "$BOT_DIR/.env.example" "$env_file"

    echo ""
    printf "${BOLD}Configure your bot:${NC}\n"

    printf "  Database URL (leave blank for SQLite): "
    read -r db_url
    [ -n "$db_url" ] && sed -i "s|DATABASE_URL=.*|DATABASE_URL=$db_url|" "$env_file"

    printf "  Bot prefix (default: .): "
    read -r bot_prefix
    [ -n "$bot_prefix" ] && sed -i "s|BOT_PREFIX=.*|BOT_PREFIX=$bot_prefix|" "$env_file"

    printf "  Your phone number (sudo): "
    read -r sudo_num
    [ -n "$sudo_num" ] && sed -i "s|SUDO=.*|SUDO=$sudo_num|" "$env_file"

    success ".env configured"
}

build_bot() {
    cd "$BOT_DIR"
    info "Building bot..."
    spin "Downloading modules" go mod tidy
    spin "Building binary" go build -ldflags "-X main.sourceDir=$(pwd)" -o zaelix .
    cd ..
    success "Bot built"
}

setup_service() {
    [ "$OS_TYPE" = "termux" ] || [ "$OS_TYPE" = "darwin" ] && return 0
    ! command -v systemctl > /dev/null 2>&1 && return 0

    local bot_user bot_path
    bot_user="${SUDO_USER:-$(whoami)}"
    bot_path="$(cd "$BOT_DIR" && pwd)"

    run_privileged tee /etc/systemd/system/zaelix.service > /dev/null << SVCEOF
[Unit]
Description=Zaelix WhatsApp Bot
After=network.target
StartLimitIntervalSec=60
StartLimitBurst=5

[Service]
Type=simple
User=$bot_user
WorkingDirectory=$bot_path
ExecStart=$bot_path/zaelix
Restart=always
RestartSec=5
StandardOutput=append:$bot_path/bot.log
StandardError=append:$bot_path/bot.log

[Install]
WantedBy=multi-user.target
SVCEOF

    run_privileged systemctl daemon-reload
    run_privileged systemctl enable zaelix >> "$LOG_FILE" 2>&1
    success "Service installed (auto-starts on reboot)"
}

pair_whatsapp() {
    local bot_path
    bot_path="$(cd "$BOT_DIR" && pwd)"

    echo ""
    printf "${BOLD}Pair your WhatsApp:${NC}\n"
    printf "  Phone number (international format, e.g. 923001234567): "
    read -r phone_number

    if [ -z "$phone_number" ]; then
        warn "No phone number provided. Run manually: cd $BOT_DIR && ./zaelix --phone-number <number>"
        return 0
    fi

    info "Starting bot for pairing..."
    cd "$BOT_DIR"
    ./zaelix --phone-number "$phone_number"
    cd ..
}

start_service() {
    [ "$OS_TYPE" = "termux" ] || [ "$OS_TYPE" = "darwin" ] && return 0
    ! command -v systemctl > /dev/null 2>&1 && return 0

    local bot_path
    bot_path="$(cd "$BOT_DIR" && pwd)"

    info "Starting bot service..."
    run_privileged systemctl start zaelix
    success "Bot is running in background"
    info "Logs: tail -f $bot_path/bot.log"
}

print_done() {
    local bot_path
    bot_path="$(cd "$BOT_DIR" && pwd)"
    echo ""
    printf "${GREEN}${BOLD}╔══════════════════════════════════════╗${NC}\n"
    printf "${GREEN}${BOLD}║        Bot is running!               ║${NC}\n"
    printf "${GREEN}${BOLD}╚══════════════════════════════════════╝${NC}\n"
    echo ""
    printf "  ${BOLD}Logs:${NC}   tail -f $bot_path/bot.log\n"
    printf "  ${BOLD}Stop:${NC}   sudo systemctl stop zaelix\n"
    printf "  ${BOLD}Start:${NC}  sudo systemctl start zaelix\n"
    printf "  ${BOLD}Status:${NC} sudo systemctl status zaelix\n"
    echo ""
}

# ── Main ─────────────────────────────────────────────────────────────────────
> "$LOG_FILE"

echo ""
printf "${BOLD}${CYAN}  Zaelix WhatsApp Bot — Installer${NC}\n"
echo ""

printf "  Bot directory name (default: whatsapp-bot): "
read -r bot_name_input
BOT_DIR="${bot_name_input:-whatsapp-bot}"
info "Installing into: ./$BOT_DIR"

detect_os
add_swap
install_deps
clone_repo
setup_env
build_bot
setup_service
pair_whatsapp
start_service
INSTALL_SUCCESS=1
print_done

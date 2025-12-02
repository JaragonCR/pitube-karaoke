#!/bin/bash

# PiTube Karaoke Installer (Pi 3B+ Optimized)
# Repo: https://github.com/JaragonCR/pitube-karaoke

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

if [ "$EUID" -eq 0 ]; then
  log_error "Please run as your normal user (e.g. jaragon), NOT root."
fi

echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}   PiTube Karaoke Installer (Pi 3B+)     ${NC}"
echo -e "${GREEN}=========================================${NC}"

# 1. UPGRADE NODE.JS TO v20
log_info "Upgrading Node.js to v20 (Required for yt-dlp speed)..."
sudo apt remove -y nodejs npm
sudo apt autoremove -y
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt update
sudo apt install -y nodejs
NODE_VER=$(node -v)
log_info "Node.js installed: $NODE_VER"

# 2. INSTALL SYSTEM DEPENDENCIES
log_info "Installing system tools (mpv, ffmpeg, kiosk utils)..."
sudo apt install -y golang ffmpeg mpv python3 unclutter x11-xserver-utils qrencode imagemagick git unzip
if [ ! -f /usr/local/bin/node ]; then
    sudo ln -sf /usr/bin/nodejs /usr/local/bin/node
fi

# 3. INSTALL YT-DLP (Zip Bundle)
log_info "Installing yt-dlp..."
sudo rm -f /usr/local/bin/yt-dlp
rm -f /tmp/yt-dlp_linux_armv7l.zip

wget -P /tmp https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux_armv7l.zip
sudo mkdir -p /opt/yt-dlp
sudo unzip -o /tmp/yt-dlp_linux_armv7l.zip -d /opt/yt-dlp
sudo chmod +x /opt/yt-dlp/yt-dlp_linux_armv7l
sudo ln -sf /opt/yt-dlp/yt-dlp_linux_armv7l /usr/local/bin/yt-dlp

# 4. SETUP PROJECT
INSTALL_DIR="$HOME/pitube-karaoke"
if [ ! -d "$INSTALL_DIR" ]; then
    log_info "Cloning repository..."
    git clone "https://github.com/JaragonCR/pitube-karaoke.git" "$INSTALL_DIR"
fi

cd "$INSTALL_DIR"
log_info "Building Go Application..."
if [ ! -f "go.mod" ]; then
    go mod init pitube
    go get modernc.org/sqlite
fi
go mod tidy
go build -o pitube
chmod +x run.sh gen_ui.sh

# 5. SETUP SEAMLESS AUTOSTART (LXDE Injection)
log_info "Configuring Seamless Autostart..."

# A. Clean up old .desktop method (which causes visible windows)
rm -f "$HOME/.config/autostart/pitube.desktop"

# B. Setup LXDE Session folder
LXDE_DIR="$HOME/.config/lxsession/LXDE-pi"
mkdir -p "$LXDE_DIR"

# C. Create autostart file if missing
if [ ! -f "$LXDE_DIR/autostart" ]; then
    if [ -f "/etc/xdg/lxsession/LXDE-pi/autostart" ]; then
        cp "/etc/xdg/lxsession/LXDE-pi/autostart" "$LXDE_DIR/autostart"
    else
        # Fallback default
        echo "@lxpanel --profile LXDE-pi" > "$LXDE_DIR/autostart"
        echo "@pcmanfm --desktop --profile LXDE-pi" >> "$LXDE_DIR/autostart"
        echo "@xscreensaver -no-splash" >> "$LXDE_DIR/autostart"
    fi
fi

# D. Inject our script if not present
if ! grep -q "pitube-karaoke/run.sh" "$LXDE_DIR/autostart"; then
    echo "@bash $INSTALL_DIR/run.sh" >> "$LXDE_DIR/autostart"
    log_info "Added to LXDE Autostart."
else
    log_info "Already in Autostart."
fi

# 6. DISABLE SCREENSAVER (Prevent screen sleeping)
log_info "Disabling Screensaver logic..."
if ! grep -q "xset s off" "$LXDE_DIR/autostart"; then
    echo "@xset s noblank" >> "$LXDE_DIR/autostart"
    echo "@xset s off" >> "$LXDE_DIR/autostart"
    echo "@xset -dpms" >> "$LXDE_DIR/autostart"
fi

echo ""
echo -e "${GREEN}SUCCESS!${NC} Setup complete."
echo -e "1. Place your 'cookies.txt' in: $INSTALL_DIR/cookies.txt"
echo -e "2. Reboot to start Kiosk mode (No terminal window)."

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

# 1. UPGRADE NODE.JS TO v20 (Required for High-Speed YouTube Downloads)
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

# 5. SETUP AUTOSTART (Autorun on Boot)
log_info "Configuring Autostart..."
mkdir -p "$HOME/.config/autostart"
cat << EOF > "$HOME/.config/autostart/pitube.desktop"
[Desktop Entry]
Type=Application
Name=PiTube Karaoke
Exec=$INSTALL_DIR/run.sh
Terminal=false
Hidden=false
EOF

# 6. DISABLE SCREENSAVER
log_info "Disabling Screensaver..."
LXDE_CONFIG="/etc/xdg/lxsession/LXDE-pi/autostart"
[ ! -f "$LXDE_CONFIG" ] && LXDE_CONFIG="/etc/xdg/lxsession/LXDE/autostart"

if ! grep -q "xset s off" "$LXDE_CONFIG"; then
    sudo bash -c "echo '@xset s noblank' >> $LXDE_CONFIG"
    sudo bash -c "echo '@xset s off' >> $LXDE_CONFIG"
    sudo bash -c "echo '@xset -dpms' >> $LXDE_CONFIG"
fi

echo ""
echo -e "${GREEN}SUCCESS!${NC} Setup complete."
echo -e "1. Place your 'cookies.txt' in: $INSTALL_DIR/cookies.txt"
echo -e "2. Reboot to start Kiosk mode."

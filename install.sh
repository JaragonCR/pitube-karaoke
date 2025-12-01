#!/bin/bash

# PiTube Karaoke Installer for Raspberry Pi 3B+
# Repo: https://github.com/JaragonCR/pitube-karaoke

# --- COLORS ---
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# --- CONFIG ---
REPO_URL="https://github.com/JaragonCR/pitube-karaoke.git"
INSTALL_DIR="$HOME/pitube-karaoke"
AUTOSTART_DIR="$HOME/.config/autostart"

# --- HELPER FUNCTIONS ---
log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

check_error() {
    if [ $? -ne 0 ]; then
        log_error "$1"
    fi
}

# --- PRE-FLIGHT CHECKS ---
if [ "$EUID" -eq 0 ]; then
  log_error "Please run this script as your normal user (e.g., jaragon), NOT as root/sudo.\nThe script will ask for sudo password when necessary."
fi

clear
echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}   PiTube Karaoke Installer (Pi 3B+)     ${NC}"
echo -e "${GREEN}=========================================${NC}"
echo ""

# 1. UPDATE SYSTEM & INSTALL DEPENDENCIES
log_info "Updating system and installing dependencies..."
sudo apt update
sudo apt install -y golang ffmpeg mpv python3 unclutter x11-xserver-utils qrencode imagemagick nodejs git unzip
check_error "Failed to install system dependencies."

# 2. INSTALL YT-DLP (The Special Zip Bundle)
log_info "Installing yt-dlp (armv7l zip bundle)..."
# Clean up old
sudo rm -f /usr/local/bin/yt-dlp
rm -f /tmp/yt-dlp_linux_armv7l.zip

# Download & Install
wget -P /tmp https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux_armv7l.zip
check_error "Failed to download yt-dlp."

sudo mkdir -p /opt/yt-dlp
sudo unzip -o /tmp/yt-dlp_linux_armv7l.zip -d /opt/yt-dlp
check_error "Failed to unzip yt-dlp."

sudo chmod +x /opt/yt-dlp/yt-dlp_linux_armv7l
sudo ln -sf /opt/yt-dlp/yt-dlp_linux_armv7l /usr/local/bin/yt-dlp
check_error "Failed to link yt-dlp."

# Verify
YTDLP_VER=$(yt-dlp --version)
log_info "yt-dlp installed successfully (Version: $YTDLP_VER)"

# 3. SETUP PROJECT DIRECTORY
log_info "Setting up project folder..."
if [ -d "$INSTALL_DIR" ]; then
    log_warn "Directory $INSTALL_DIR already exists."
    read -p "Do you want to delete it and reinstall clean? (y/n): " confirm
    if [[ $confirm == [yY] || $confirm == [yY][eE][sS] ]]; then
        rm -rf "$INSTALL_DIR"
        git clone "$REPO_URL" "$INSTALL_DIR"
    else
        log_info "Updating existing repo..."
        cd "$INSTALL_DIR"
        git pull
    fi
else
    git clone "$REPO_URL" "$INSTALL_DIR"
fi
check_error "Failed to setup project repository."

# 4. BUILD GO APPLICATION
log_info "Building Go application..."
cd "$INSTALL_DIR"

# Initialize mod if missing (handles fresh clones)
if [ ! -f "go.mod" ]; then
    go mod init pitube
    go get modernc.org/sqlite
fi

go mod tidy
go build -o pitube
check_error "Failed to compile Go application."

# Make scripts executable
chmod +x run.sh gen_ui.sh
check_error "Failed to chmod scripts."

# 5. SETUP AUTOSTART
log_info "Configuring Autostart..."
mkdir -p "$AUTOSTART_DIR"
cat << EOF > "$AUTOSTART_DIR/pitube.desktop"
[Desktop Entry]
Type=Application
Name=PiTube Karaoke
Exec=$INSTALL_DIR/run.sh
Terminal=false
Hidden=false
EOF
check_error "Failed to create autostart entry."

# 6. DISABLE SCREENSAVER (LXDE)
log_info "Disabling Screensaver..."
LXDE_CONFIG="/etc/xdg/lxsession/LXDE-pi/autostart"
if [ ! -f "$LXDE_CONFIG" ]; then
    LXDE_CONFIG="/etc/xdg/lxsession/LXDE/autostart"
fi

if [ -f "$LXDE_CONFIG" ]; then
    # Only append if not already there
    if ! grep -q "xset s off" "$LXDE_CONFIG"; then
        sudo bash -c "echo '@xset s noblank' >> $LXDE_CONFIG"
        sudo bash -c "echo '@xset s off' >> $LXDE_CONFIG"
        sudo bash -c "echo '@xset -dpms' >> $LXDE_CONFIG"
    fi
fi

# 7. CLEANUP
log_info "Cleaning up temporary files..."
rm -f /tmp/yt-dlp_linux_armv7l.zip
# Note: We do NOT remove Go, as you will need it to compile updates later.

# 8. SUMMARY
echo ""
echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}       INSTALLATION SUCCESSFUL!          ${NC}"
echo -e "${GREEN}=========================================${NC}"
echo ""
echo -e "1. ${YELLOW}COOKIES:${NC} Don't forget to copy your 'cookies.txt' to:"
echo -e "   $INSTALL_DIR/cookies.txt"
echo ""
echo -e "2. ${YELLOW}START:${NC} You can start it now by running:"
echo -e "   $INSTALL_DIR/run.sh"
echo ""
echo -e "3. ${YELLOW}AUTOSTART:${NC} It will start automatically on next reboot."
echo ""

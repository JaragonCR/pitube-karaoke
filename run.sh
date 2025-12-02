#!/bin/bash
cd "$(dirname "$0")"

# 1. Wait for X11 (Display)
export DISPLAY=:0
echo "Waiting for X11..."
until xset -q >/dev/null 2>&1; do
    sleep 1
    echo "..."
done

# 2. Setup Kiosk Environment
xsetroot -solid black        # Black background
unclutter -idle 0.1 &        # Hide mouse immediately
xset s off -dpms             # Disable screensaver

# 3. Generate Wallpaper (QR Code)
./gen_ui.sh

# 4. Run PiTube (Foreground Mode - DO NOT BACKGROUND)
#    We pipe output to log, but we keep the process attached to this script.
./pitube > pitube_runtime.log 2>&1

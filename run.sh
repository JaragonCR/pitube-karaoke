#!/bin/bash
cd "$(dirname "$0")"

# 1. Wait for X11
export DISPLAY=:0
echo "Waiting for X11..."
until xset -q >/dev/null 2>&1; do
    sleep 1
    echo "..."
done

# 2. Setup Kiosk Environment (Seamless Black)
xsetroot -solid black        # Black background
unclutter -idle 0.1 &        # Hide mouse immediately
xset s off -dpms             # Disable screensaver

# 3. Generate new Splash with correct IP
./gen_ui.sh

# 4. Run PiTube (Logged to file, no visible terminal)
./pitube > pitube_runtime.log 2>&1

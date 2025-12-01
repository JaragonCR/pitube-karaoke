#!/bin/bash

# 1. Get Local IP Address
IP=$(hostname -I | awk '{print $1}')
URL="http://$IP:8080"

echo "Detected IP: $IP"
echo "Generating Kiosk UI..."

# 2. Generate QR Code for the URL
qrencode -o qr.png -s 10 -l H -m 2 "$URL"

# 3. Create the Background Image (1920x1080)
# Uses ImageMagick to compose text and QR code
convert -size 1920x1080 xc:black \
    -font DejaVu-Sans-Bold -pointsize 100 -fill "#bb86fc" -gravity North -annotate +0+100 "PiTube Karaoke" \
    -font DejaVu-Sans -pointsize 60 -fill white -gravity Center -annotate +0-100 "Scan to Sing! 📷" \
    -font DejaVu-Sans-Mono -pointsize 40 -fill "#03dac6" -gravity SouthWest -annotate +50+50 "Connect: $URL" \
    qr.png -gravity SouthEast -geometry +50+50 -composite \
    background.png

echo "Done! background.png created."

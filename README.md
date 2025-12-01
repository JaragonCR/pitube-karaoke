# 🎤 PiTube Karaoke (Pi 3B+ Edition)

A dedicated, self-hosted Karaoke appliance designed specifically for the **Raspberry Pi 3B+**. 
It turns your Pi into a Kiosk that displays a QR code on your TV. Guests scan the code to search for songs, queue them up, and control playback from their phones.

> **⚠️ HARDWARE REQUIREMENT:** This project is **NOT Headless**. It requires a TV/Monitor connected via HDMI to function. It relies on the X11 window system to display the video and overlay text.

## 🚀 Why this fork?
Modern YouTube playback (1080p/60fps VP9) causes the Raspberry Pi 3B+ to freeze and overheat. This project is heavily optimized for this specific hardware:
- **Forced 480p:** Limits downloads to SD quality to ensure smooth CPU decoding.
- **X11 Video Driver:** Bypasses the unstable 3B+ GPU overlay drivers to prevent system hangs.
- **Smart Queue:** Downloads the next song while the current one plays.
- **Kiosk Mode:** Hides the mouse, terminal, and desktop for a seamless appliance feel.

## 🛠️ Prerequisites

Running on **Raspberry Pi OS (Legacy/Bullseye or Bookworm)** with Desktop.

```bash
# 1. Install System Dependencies
sudo apt update
sudo apt install -y golang ffmpeg mpv python3 unclutter x11-xserver-utils qrencode imagemagick nodejs

# 2. Install yt-dlp (The Downloader)
# Note: Do not use apt for yt-dlp, it is too old. Use the official binary.
sudo wget [https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux_armv7l](https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux_armv7l) -O /usr/local/bin/yt-dlp
sudo chmod a+rx /usr/local/bin/yt-dlp

📦 Installation
Clone or Copy the files to a folder named pitube-karaoke.

Initialize Go:
cd pitube-karaoke
go mod init pitube
go get modernc.org/sqlite
go mod tidy

Build the Application:
go build -o pitube
chmod +x run.sh gen_ui.sh

🍪 Fixing 403 Forbidden Errors (If song downloads fail)
YouTube aggressively blocks server IP addresses (Data Center IPs). If downloads fail instantly, you must provide your browser cookies to prove you are human.

Install a "Get cookies.txt LOCALLY" extension on your PC (Chrome/Firefox).
Go to YouTube.com and export your cookies as a file.
Rename the file to cookies.txt.
Place cookies.txt inside the pitube-karaoke folder on your Pi.
Restart the app. It will automatically detect the file and use it to bypass blocks.

🏃‍♂️ Running it
Manual Start
./run.sh

Auto-Start on Boot
To make it run automatically when you plug in the Pi:

Create a desktop entry:

Bash

mkdir -p ~/.config/autostart --If autostart doesn't exist
nano ~/.config/autostart/pitube.desktop

Paste this content:
[Desktop Entry]
Type=Application
Name=PiTube Karaoke
Exec=/home/pi/pitube-karaoke/run.sh
Terminal=false
Hidden=false

🎮 How to Use
Turn on the Pi and TV.
Wait for the QR Code Splash Screen to appear.
Scan the QR code with your phone (or type the IP address shown).
Enter your Name.
Search for a song (e.g., "Bohemian Rhapsody") and tap "Select".
The song will download and start playing automatically on the TV!

🗑️ Maintenance
Update System: There is an "Update System" link at the bottom of the web interface to update the downloader.
Clear Database: Delete the pitube.db file to reset your history/queue.
Clear Downloads: Delete files in the downloads/ folder to free up space.

# 🎤 PiTube Karaoke (Pi 3B+ Edition)

A dedicated, self-hosted Karaoke appliance designed specifically for the **Raspberry Pi 3B+**. 

It turns your Pi into a Kiosk that displays a QR code on your TV. Guests scan the code to search for songs, queue them up, and control playback from their phones.

> **⚠️ HARDWARE REQUIREMENT:** This project is **NOT Headless**. It requires a TV/Monitor connected via HDMI to function. It relies on the X11 window system to display the video and overlay text.

## 🚀 Why this fork?
Modern YouTube playback (1080p/60fps VP9) causes the Raspberry Pi 3B+ to freeze and overheat. This project is heavily optimized for this specific hardware:
- **Forced 480p:** Limits downloads to SD quality to ensure smooth CPU decoding.
- **X11 Video Driver:** Bypasses the unstable 3B+ GPU overlay drivers to prevent system hangs.
- **Node.js Accelerated:** Uses a custom Node.js 20+ runtime to bypass YouTube download throttling.
- **Smart Queue:** Downloads the next song while the current one plays.
- **Kiosk Mode:** Hides the mouse, terminal, and desktop for a seamless appliance feel.

## 🛠️ Installation (The Easy Way)

We have provided an automatic installer script that handles dependencies (Node.js 20, mpv, ffmpeg), installs the correct version of `yt-dlp`, and sets up the autostart service.

Run this single command on your Raspberry Pi:

```bash
wget [https://raw.githubusercontent.com/JaragonCR/pitube-karaoke/main/install.sh](https://raw.githubusercontent.com/JaragonCR/pitube-karaoke/main/install.sh)
chmod +x install.sh
./install.sh
```
⚙️ Manual Installation (The Hard Way)
If you prefer to install manually, you must ensure you have Node.js v20+ (default apt version is too old) and the yt-dlp zip bundle.

Install Dependencies:

# Remove old Node
sudo apt remove -y nodejs npm
# Add Node v20 Repo
curl -fsSL [https://deb.nodesource.com/setup_20.x](https://deb.nodesource.com/setup_20.x) | sudo -E bash -
# Install System Tools
sudo apt install -y nodejs golang ffmpeg mpv python3 unclutter x11-xserver-utils qrencode imagemagick git unzip

Install yt-dlp (Zip Bundle):

wget [https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux_armv7l.zip](https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux_armv7l.zip)
sudo mkdir -p /opt/yt-dlp
sudo unzip -o yt-dlp_linux_armv7l.zip -d /opt/yt-dlp/
sudo chmod +x /opt/yt-dlp/yt-dlp_linux_armv7l
sudo ln -sf /opt/yt-dlp/yt-dlp_linux_armv7l /usr/local/bin/yt-dlp

Build & Run:

git clone [https://github.com/JaragonCR/pitube-karaoke.git](https://github.com/JaragonCR/pitube-karaoke.git)
cd pitube-karaoke
go mod init pitube
go get modernc.org/sqlite
go mod tidy
go build -o pitube
./run.sh

🍪 Fixing 403 Forbidden Errors (Optional)
YouTube aggressively blocks server IP addresses (Data Center IPs). If downloads fail instantly, you must provide your browser cookies to prove you are human.

Install a "Get cookies.txt LOCALLY" extension on your PC (Chrome/Firefox).
Go to YouTube.com and export your cookies as a file.
Rename the file to cookies.txt.
Transfer this file to your Pi: /home/YOUR_USER/pitube-karaoke/cookies.txt.
Restart the app. It will automatically detect the file and use it to bypass blocks.

🎮 How to Use

Turn on the Pi and TV.
Wait for the QR Code Splash Screen to appear.
Scan the QR code with your phone (or type the IP address shown).
Enter your Name.
Search for a song (e.g., "Bohemian Rhapsody") and tap "Select".
The song will download and start playing automatically on the TV!

🗑️ Maintenance

Update System: There is an "Update System" link at the bottom of the web interface to update the downloader.
Logs: Check pitube_debug.log if something goes wrong (it auto-rotates at 1MB).
Clear Database: Delete the pitube.db file to reset your history/queue.

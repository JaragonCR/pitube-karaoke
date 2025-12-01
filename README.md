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

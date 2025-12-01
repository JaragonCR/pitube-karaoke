# 🎤 PiTube Karaoke (Pi 3B+ Edition)

A dedicated, self-hosted Karaoke appliance designed specifically for the **Raspberry Pi 3B+**. 

It turns your Pi into a Kiosk that displays a QR code on your TV. Guests scan the code to search for songs, queue them up, and control playback from their phones.

> **⚠️ HARDWARE REQUIREMENT:** This project is **NOT Headless**. It requires a TV/Monitor connected via HDMI to function. It relies on the X11 window system to display the video and overlay text.

## 🚀 Why this fork?
Modern YouTube playback (1080p/60fps VP9) causes the Raspberry Pi 3B+ to freeze. This project is heavily optimized for this specific hardware:
- **CPU Decoding (480p):** Limits downloads to SD quality to ensure smooth CPU decoding (GPU is unstable on Pi 3B+ with overlays).
- **Node.js Accelerated:** Uses a custom Node.js runtime to bypass YouTube download throttling.
- **Smart Queue:** Downloads the next song while the current one plays.
- **Kiosk Mode:** Hides the mouse, terminal, and desktop for a seamless appliance feel.

## 🛠️ Installation

### Option 1: Automatic Installer (Recommended)
Run this single command on your Raspberry Pi:

```bash
wget [https://raw.githubusercontent.com/JaragonCR/pitube-karaoke/main/install.sh](https://raw.githubusercontent.com/JaragonCR/pitube-karaoke/main/install.sh)
chmod +x install.sh
./install.sh

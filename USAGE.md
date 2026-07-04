# How to Use Whisper Voice Utility

Whisper Voice Utility is a tool that lets you talk into your computer and have it type out what you say. It can also read text back to you using computer voices.

---

## 🚀 Quick Install (one line)

If you're on Debian / Ubuntu / Pop!_OS / Linux Mint / KDE neon / Zorin, just paste this into a terminal:

```bash
curl -fsSL https://github.com/yourusername/whisper-voice-util/releases/latest/download/install.sh | bash
```

That's it. The script will:
1. Download the latest release tarball from GitHub.
2. Extract it to `/opt/whisper-voice-util/`.
3. Install the system libraries (GTK 3, system tray, clipboard, etc.) with `apt-get`.
4. Link `whisper-voice-util` into your `$PATH`.
5. Add an entry to your app menu.

When it finishes, just type `whisper-voice-util` (or click the icon in your app menu). The **setup wizard** will open the first time, walk you through picking a language and downloading the model, and the tray icon will appear.

To uninstall later:

```bash
sudo rm -rf /opt/whisper-voice-util
sudo rm -f /usr/local/bin/whisper-voice-util /usr/local/bin/whisper-voice-overlay
sudo rm -f /usr/local/share/applications/whisper-voice-util.desktop
```

---

## 📦 Manual Install (if you prefer to see what runs)

If you'd rather not pipe a remote script into `bash`, or you're on a non-Debian distro, you can do it by hand. It takes 3 commands:

### 1. Download and extract

Grab `whisper-voice-util-vX.Y.Z-linux-amd64.tar.gz` from the [GitHub Releases page](https://github.com/yourusername/whisper-voice-util/releases) and extract it:

```bash
tar xzf whisper-voice-util-vX.Y.Z-linux-amd64.tar.gz
cd whisper-voice-util-vX.Y.Z
```

The folder contains:
```
whisper-voice-util          # the main binary
whisper-voice-overlay       # the recording indicator window
engines/                    # bundled whisper.cpp (+ piper if built)
README.md
USAGE.md
install.sh                  # the one-liner installer (re-runnable)
install-deps.sh             # apt installer for system libraries
config.yaml.example         # template; the wizard fills this in for you
```

### 2. Install system dependencies (one time)

`install-deps.sh` installs the libraries the App needs at runtime: GTK 3, the system-tray library, xclip / xdotool, the audio stack, espeak-ng (for piper TTS).

```bash
sudo ./install-deps.sh
```

It detects your distro, prepends `sudo` if you're not root, skips already-installed packages, and exits cleanly. Re-runnable.

### 3. Run

```bash
./whisper-voice-util
```

The first launch detects that no setup state exists and opens the **setup wizard** — a small GTK window that walks you through:

1. **Welcome** — a quick overview
2. **Language** — pick your speech language (English is default; the manifest lists 99)
3. **Whisper model** — choose a size and watch the download progress bar
4. **Hotkey** — pick a preset or capture your own key combination
5. **Piper voice** — (optional) pick a TTS voice and watch the download
6. **Finish** — "Start" writes `state.json` and `config.yaml` with the resolved paths

After the wizard finishes, the system tray icon appears and the App is ready. Hold your hotkey, speak, release, and your words appear at the cursor.

### 4. (Optional) Install globally

If you'd like `whisper-voice-util` on your `$PATH` and a desktop file in your app menu, run `sudo make install` from the extracted directory. Use `sudo make uninstall` to remove.

---

## 🛠️ How to Use

### Talking to Type
1. Move your cursor to where you want to type (like an email or a document).
2. **Hold down** the `Right Control` and `Left Arrow` keys on your keyboard.
3. Speak clearly into your microphone.
4. **Release** the keys when you are finished.
5. Wait a second, and the text will appear where you were typing!

### Hearing the Clipboard
1. Copy some text (like you normally do with `Ctrl+C`).
2. Press the `F10` key.
3. The computer will read the text back to you.

### Changing Settings
Right-click the **microphone icon** in the corner of your screen (the "System Tray") to see options:
*   Change which transcription or voice engine is active.
*   **Run setup again...** — re-do the wizard (great for picking a different model or hotkey).
*   **Check for updates** — see if a new version is available.
*   **Open App-managed folder** — see the downloaded model files and your setup state.
*   **Open Config** — edit advanced settings in `config.yaml`.
*   **Quit** — close the program.

---

## ❓ Things to Remember
*   **First Run:** The first time you open the program, the setup wizard runs. You don't need to edit any config files by hand — the wizard does it for you.
*   **API Keys:** If you use "Cloud" engines (OpenAI for transcription, ElevenLabs for voice), put your API keys in `config.yaml` (or use the `${OPENAI_API_KEY}` and `${ELEVENLABS_API_KEY}` env-var syntax). The wizard doesn't ask for these.
*   **Updates:** When a new version is published on GitHub, a "Update available" item appears at the top of the tray menu. Click it to download and restart automatically.
*   **Need Help?** See the [README Troubleshooting section](README.md#troubleshooting) for common issues (wizard not appearing, tray icon missing, hotkey not working, models not downloading).

# Whisper Voice Utility

Whisper Voice Utility is a Linux desktop application that provides system-wide push-to-talk voice transcription and Text-to-Speech (TTS) capabilities. It integrates deeply with X11/Wayland to listen for global hotkeys, record your microphone, transcribe your speech using local or cloud AI models, and automatically type the transcribed text into any active window.

## Features

- **Push-to-Talk Transcription:** Hold a configurable hotkey (e.g., `<rightctrl>+<left>`), speak, and have the text instantly auto-typed wherever your cursor is.
- **Clipboard TTS:** Press a hotkey to instantly speak aloud whatever text is currently in your system clipboard.
- **Multiple AI Engines:**
  - **Transcription:** `whisper.cpp` (local, instant) or `openai_api` (cloud, highly accurate).
  - **TTS:** `piper` (local, fast) or `elevenlabs` (cloud, ultra-realistic).
- **Bundled Engines:** The release tarball ships with `whisper-cli` and (optionally) `piper` pre-built — no need to compile them yourself.
- **First-Run Wizard:** A GTK setup wizard walks you through language choice, model download, and hotkey selection on first launch.
- **System Tray Integration:** A clean tray application gives you instant visual feedback on recording status and lets you quickly toggle between engines, re-run the wizard, check for updates, or open configuration files.
- **Native Notifications:** Uses DBus notifications to keep you informed of errors or transcription background processes without interrupting your workflow.
- **Auto-Update:** The tray menu surfaces new GitHub releases as a clickable "Update available (vX.Y.Z)" item; click it to download the new binary, swap it in, and restart.
- **Single Instance & Autostart:** Easily configure the utility to start on boot, while preventing duplicate instances.

## Installation (end-user, from a release tarball)

Works on any Debian-family Linux distribution (Ubuntu, Pop!_OS, Linux Mint, elementary, KDE neon, Zorin). Other distros can use the manual path below.

### Quick install (one line)

```bash
curl -fsSL https://github.com/spanexx/voces/releases/latest/download/install.sh | bash
```

That's it. The script downloads the latest tarball, extracts it to `/opt/whisper-voice-util/`, runs `install-deps.sh` to install the system libraries, links the binaries into your `$PATH`, and adds an app-menu entry. When it finishes, type `whisper-voice-util` (or click the menu entry) — the setup wizard will open on first launch.

To uninstall:

```bash
sudo rm -rf /opt/whisper-voice-util
sudo rm -f /usr/local/bin/whisper-voice-util /usr/local/bin/whisper-voice-overlay
sudo rm -f /usr/local/share/applications/whisper-voice-util.desktop
```

### Manual install

If you'd rather see what runs (or you're on a non-Debian distro), the manual path is: download, extract, install system deps, then run. There are also two optional follow-ups: install globally, or build from source (see below).

### 1. Download the latest release

Grab `whisper-voice-util-vX.Y.Z-linux-amd64.tar.gz` from the [GitHub Releases page](https://github.com/spanexx/voces/releases). Place it anywhere — your home directory is fine.

### 2. Extract

```bash
tar xzf whisper-voice-util-vX.Y.Z-linux-amd64.tar.gz
cd whisper-voice-util-vX.Y.Z
```

You will see:
```
whisper-voice-util          # the main binary
whisper-voice-overlay       # the recording indicator window
engines/                    # bundled whisper.cpp (+ piper if the build succeeded)
README.md
USAGE.md
install.sh                  # the one-liner installer (re-runnable)
install-deps.sh             # one-time system dep installer
config.yaml.example         # template; the wizard fills this in for you
```

### 3. Install system dependencies (one time)

`install-deps.sh` installs the libraries the App needs at runtime: GTK 3, the system-tray library, xclip / xdotool, the audio stack, espeak-ng (for piper TTS).

```bash
sudo ./install-deps.sh
```

It detects your distro, prepends `sudo` if you're not root, skips already-installed packages, and exits cleanly. Re-runnable.

### 4. Run

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

### 5. (Optional) Install globally

If you'd like `whisper-voice-util` on your `$PATH` and a desktop file in your app menu, run `sudo make install` from the extracted directory. Use `sudo make uninstall` to remove.

## Building from source

If you want the very latest code or to modify the App:

```bash
git clone https://github.com/spanexx/voces.git
cd voces
make build                  # builds the two Go binaries
make whispercpp-build       # compiles the vendored whisper.cpp
sudo ./scripts/install-deps.sh   # same as the tarball step
./bin/whisper-voice-util
```

`make release VERSION=v0.2.0` produces a release tarball at `builds/whisper-voice-util-v0.2.0-linux-amd64.tar.gz`. This is what gets uploaded to GitHub Releases.

## Configuration

On first launch, the App runs the **setup wizard**, which writes two files:

- `~/.local/share/whisper-voice-util/state.json` — tracks the wizard version, your language, model choice, hotkey. The wizard re-runs when this file is missing or the App version has changed.
- `$XDG_CONFIG_HOME/whisper-voice-util/config.yaml` — the engine paths, model paths, API keys, hotkeys, and behavior flags. You can edit this by hand; the next "Run setup again..." from the tray will regenerate it.

A `config.yaml.example` is shipped in the tarball for reference. All engine paths are **placeholders** — the wizard fills them in based on where it downloaded the model files and where the bundled engines live.

## Usage

Once running, you'll see a microphone icon in your system tray.

### Default Hotkeys

The application supports system-wide hotkeys, meaning they work regardless of which window is in focus.

- **Record & Auto-Type:** `Right Control` + `Left Arrow` (Hold to Record, Release to Transcribe & Type).
- **Read Clipboard (TTS):** `F10` (Press once to generate and play audio from your clipboard contents).
- **Toggle TTS Engine:** `F11` (Quickly cycle between Piper and ElevenLabs).
- **Toggle Transcription Engine:** `F12` (Quickly cycle between Whisper.cpp and OpenAI API).

*Note: All hotkeys can be customized in `config.yaml` under the `hotkeys` section, or by re-running the wizard from the tray.*

### System Tray Menu

Right-click the microphone icon in your tray to:
- See the current system state (Idle, Recording, Transcribing, Error).
- Manually select transcription / TTS engines.
- **Run setup again...** — re-launch the wizard (new window).
- **Check for updates** — query GitHub Releases; shows a notification on result.
- **Open App-managed folder** — opens `~/.local/share/whisper-voice-util/` in your file manager.
- **Open Config** — opens `config.yaml` in your default editor.
- **View Logs** — opens the log file.
- Enable/Disable "Start on Login".
- Quit the application safely.

## Troubleshooting

### "Models won't download"

The wizard downloads the AI model files from HuggingFace. If the download fails:
- Check your internet connection. The downloads are 141 MB (base model) to 466 MB (small model).
- Check that `engines/models.json` exists in the tarball directory. If missing, your extraction may have been incomplete.
- Look at the log file (`$XDG_CONFIG_HOME/whisper-voice-util/logs/whisper-voice-util.log`) for the specific HTTP error.
- You can re-run the wizard from the tray menu to retry the download.

### "The wizard doesn't appear"

- The wizard requires a graphical session (X11 or Wayland). If you're running headless (SSH, no `$DISPLAY`), the App will fail to start. Use `xvfb-run` to test, or run on a desktop session.
- Check that the system-tray library is installed: `apt list --installed libayatana-appindicator3-1`. If missing, re-run `install-deps.sh`.
- On some Wayland compositors, GTK 3 apps may need XWayland fallback. The tray icon will appear via XWayland.

### "Tray icon is missing"

- The tray icon depends on `libayatana-appindicator3-1`. Install it: `sudo apt install libayatana-appindicator3-1`.
- Some desktop environments (notably GNOME) hide tray icons by default. Install the [AppIndicator extension](https://extensions.gnome.org/extension/615/appindicator-support/) for GNOME.
- KDE Plasma, XFCE, MATE, Cinnamon, LXQt all show tray icons out of the box.

### "Hotkey doesn't work"

- The default hotkey is `Right Control + Left Arrow`. Make sure you're holding both keys simultaneously.
- Other apps (window managers, screen recorders, KVM switches) can intercept global hotkeys. Check the App's log file to see whether the hotkey manager detected the keypress.
- On Wayland, global hotkeys are restricted by the compositor. The App uses X11 key grab via `xdotool`-compatible APIs, which works under XWayland.
- You can change the hotkey by re-running the wizard (tray → "Run setup again..." → "Hotkey" step) and capturing a different combination.

### "TTS doesn't speak"

- Piper requires `espeak-ng` to generate phonemes. Install it: `sudo apt install espeak-ng`.
- The default TTS engine is `piper` (local). If the piper build was skipped in the tarball (heavy build deps), you'll need to install piper system-wide: `sudo apt install piper`.
- Alternatively, switch to the cloud TTS engine: edit `config.yaml` and set `tts.default_engine: elevenlabs`, then put your ElevenLabs API key in `tts.elevenlabs.api_key` (`${ELEVENLABS_API_KEY}` for env-var substitution).

### "Update available" menu item is missing

- The auto-updater runs in the background 10 seconds after launch, then every restart. It queries `https://api.github.com/repos/.../releases/latest`.
- If your network blocks GitHub, the check fails silently and the badge never appears. You can still download updates manually from the GitHub Releases page.
- The updater looks for a tarball whose name ends in `-linux-amd64.tar.gz`. Custom asset names will be silently skipped.

### "Audio doesn't record"

- The App uses ALSA / PulseAudio to capture from your default input device. Make sure your mic is unmuted and is the system default: `pavucontrol` → "Input Devices".
- Some virtual audio devices (JACK, PipeWire with strict profiles) may not be auto-detected. Switch to PulseAudio or install the appropriate bridge.

### "I edited config.yaml and the App broke"

- The App reads `config.yaml` on launch. To re-validate, re-launch.
- If the file is invalid YAML, the App will log the parse error and exit. Re-run the wizard (tray → "Run setup again...") to regenerate a valid one.

## License

MIT License — see [LICENSE](LICENSE).

## See also

- [USAGE.md](USAGE.md) — end-user quick-start guide (simpler than this README).
- [docs/public-setup/IMPL-public-setup.md](docs/public-setup/IMPL-public-setup.md) — the 9-phase plan that built this.

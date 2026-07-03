# Whisper Voice Utility

Whisper Voice Utility is a Linux desktop application that provides system-wide push-to-talk voice transcription and Text-to-Speech (TTS) capabilities. It integrates deeply with X11/Wayland to listen for global hotkeys, record your microphone, transcribe your speech using local or cloud AI models, and automatically type the transcribed text into any active window.

## Features

- **Push-to-Talk Transcription:** Hold a configurable hotkey (e.g., `<rightctrl>+<left>`), speak, and have the text instantly auto-typed wherever your cursor is.
- **Clipboard TTS:** Press a hotkey to instantly speak aloud whatever text is currently in your system clipboard.
- **Multiple AI Engines:**
  - **Transcription:** `whisper.cpp` (local, instant) or `openai_api` (cloud, highly accurate).
  - **TTS:** `piper` (local, fast) or `elevenlabs` (cloud, ultra-realistic).
- **System Tray Integration:** A clean tray application gives you instant visual feedback on recording status and lets you quickly toggle between engines or open configuration files.
- **Native Notifications:** Uses DBus notifications to keep you informed of errors or transcription background processes without interrupting your workflow.
- **Single Instance & Autostart:** Easily configure the utility to start on boot, while preventing duplicate instances.

## Installation

### Dependencies

This application requires several Linux system libraries for clipboard, key state detection, keyboard simulation, and DBus communication:

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install -y libx11-dev libxext-dev libxtst-dev xclip xdotool libayatana-appindicator3-dev
```

Additionally, you need the engine binaries if you plan to use local inference:
- [whisper.cpp](https://github.com/ggerganov/whisper.cpp)
- [piper](https://github.com/rhasspy/piper)

### Building from Source

1. Clone the repository:
```bash
git clone https://github.com/yourusername/whisper-voice-util.git
cd whisper-voice-util
```

2. Build the application using the Makefile:
```bash
make build
```

3. Install globally:

```bash
sudo make install
```

If you already built and only want to install artifacts from `bin/`:

```bash
sudo make install-fast
```

4. The compiled binary will be located at `bin/whisper-voice-util`. Run it!

```bash
./bin/whisper-voice-util
```

## Configuration

On the first launch, the application generates a default `config.yaml` file in the same directory as the executable.

You'll need to update this file to reflect the actual paths to your local AI binaries/models, or input your API keys for cloud engines.

Example `config.yaml` layout:

```yaml
transcription:
  default_engine: whisper_cpp
  whisper_cpp:
    binary_path: /opt/whisper.cpp/main
    model: /opt/whisper.cpp/models/ggml-small.bin
  openai_api:
    api_key: YOUR_API_KEY
    
tts:
  default_engine: piper
  piper:
    binary_path: /opt/piper/piper
    model: /opt/piper/models/en_US-lessac-medium.onnx
```

## Usage

Once running, you'll see a microphone icon in your system tray. 

### Default Hotkeys

The application supports system-wide hotkeys, meaning they work regardless of which window is in focus.

- **Record & Auto-Type:** `Right Control` + `Left Arrow` (Hold to Record, Release to Transcribe & Type).
- **Read Clipboard (TTS):** `F10` (Press once to generate and play audio from your clipboard contents).
- **Toggle TTS Engine:** `F11` (Quickly cycle between Piper and ElevenLabs).
- **Toggle Transcription Engine:** `F12` (Quickly cycle between Whisper.cpp and OpenAI API).

*Note: All hotkeys can be customized in `config.yaml` under the `hotkeys` section.*

### System Tray

Right-click the microphone icon in your tray to:
- See the current system state (Idle, Recording, Transcribing, Error).
- Manually select engines.
- Open Settings (Automatically opens `config.yaml` in your default text editor).
- View Application Logs.
- Enable/Disable "Start on Login".
- Quit the application safely.

## Troubleshooting

- **"arecord not found" / "aplay not found"**: Make sure you have `alsa-utils` installed as the application uses it for fast audio chunking.
- **Keys not auto-typing**: Ensure your window manager isn't stealing focus. Wayland support for `xdotool` features (which the keyboard simulator uses) may vary depending on XWayland compatibility layers.
- **`sudo make install` appears stuck**: Run `sudo -v` first, then retry `sudo make install`. If you already built binaries, use `sudo make install-fast` to skip rebuilding.

## License
MIT License

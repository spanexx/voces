# How to Use Whisper Voice Utility

Whisper Voice Utility is a tool that lets you talk into your computer and have it type out what you say. It can also read text back to you using computer voices.

---

## 🚀 How to Install

### 1. Get the Software
Download the latest release from the [GitHub Releases page](https://github.com/yourusername/whisper-voice-util/releases) — look for a file ending in `-linux-amd64.tar.gz`.

*   **Ready-to-use:** Download the `.tar.gz` file, extract it anywhere:
    ```bash
    tar xzf whisper-voice-util-vX.Y.Z-linux-amd64.tar.gz
    cd whisper-voice-util-vX.Y.Z
    ```
*   **Build from scratch:** If you have the source code, open a terminal in the project folder and type:
    ```bash
    make build
    ```
    This creates the program in a folder called `bin`.

### 2. Install System Helpers (one time)
Your computer needs a few extra "helpers" to handle the clipboard, keyboard, and system tray. From inside the extracted folder, run:
```bash
sudo ./install-deps.sh
```
This is safe to re-run. It only installs what's missing.

### 3. First Run
Find the `whisper-voice-util` file and double-click it, or run it from the terminal:
```bash
./whisper-voice-util
```
On first launch, a **setup wizard** window appears. It walks you through:
- Picking your language
- Downloading the speech recognition "brain" (a model file)
- Picking a hotkey
- (Optional) Downloading a voice for the speaking feature

Click "Start" at the end. The wizard saves your choices and the tray icon (a small microphone in your system tray) appears. You're ready to go.

### 4. (Optional) Global Installation
To make the program available everywhere on your system (adding it to your app menu and letting you run `whisper-voice-util` from any terminal), use the following command:
```bash
sudo make install
```
This will:
- Copy the program to your system path.
- Add an icon to your application menu.
- Register the app so it appears in your launcher.

*To remove it later, run `sudo make uninstall`.*

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

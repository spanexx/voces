# Whisper Voice Utility

Whisper Voice Utility is a tool that lets you talk into your computer and have it type out what you say. It can also read text back to you using computer voices.

---

## 🚀 How to Install

### 1. Get the Software
You can either download a ready-to-use version or build it yourself:
*   **Ready-to-use:** Look for the file `whisper-voice-util-vX.Y.Z-linux-amd64.tar.gz` in the [GitHub Releases page](https://github.com/spanexx/voces/releases) for the latest version. Extract it anywhere you like.
*   **Build from scratch:** If you have the code, just open a terminal and type:
    ```bash
    make build
    ```
    This will create the program in a folder called `bin`.

### 2. Global Installation (Recommended)
To make the program available everywhere on your system (adding it to your app menu and allowing you to run it from any terminal), use the following command:
```bash
sudo make install
```
This will:
- Copy the program to your system path.
- Add an icon to your application menu.
- Register the app so it appears in your launcher.

*To remove it later, you can use `sudo make uninstall`.*

### 3. Install Needed Tools
Your computer needs a few extra "helpers" to handle the clipboard and keyboard. Run this command in your terminal:
```bash
sudo apt-get update
sudo apt-get install -y xclip xdotool libx11-dev libxtst-dev libayatana-appindicator3-dev
```

### 3. (Optional) Set Up Local AI
If you want the program to work without the internet, you'll need the "brains" (models) for transcription and speaking:
*   **For typing:** Get [whisper.cpp](https://github.com/ggerganov/whisper.cpp).
*   **For speaking:** Get [piper](https://github.com/rhasspy/piper).

---

## 🛠️ How to Use

### Starting the Program
Find the `whisper-voice-util` file and double-click it, or run it from the terminal:
```bash
./bin/whisper-voice-util
```

### Talking to Type
1. Move your cursor to where you want to type (like an email or a document).
2. **Hold down** the `Right Control` and `Left Arrow` keys on your keyboard.
3. Speak clearly into your microphone.
4. **Release** the keys when you are finished.
5. Wait a second, and the text will appear where you were typing!

### Hearing the Clipboard
1. Copy some text (like you normally do with Ctrl+C).
2. Press the `F10` key.
3. The computer will read the text back to you.

### Changing Settings
Look for the **Microphone icon** in the corner of your screen (the "System Tray"):
*   **Right-click** it to change which "engine" (brain) the program uses.
*   Click **Open Settings** to change hotkeys or file paths. The program will open a text file called `config.yaml` for you to edit.

---

## ❓ Things to Remember
*   **First Run:** The first time you open the program, it creates a `config.yaml` file. You may need to edit this file to tell the program where you saved your AI models.
*   **API Keys:** If you use "Cloud" versions (like OpenAI or ElevenLabs), you must put your secret API keys in the `config.yaml` file.

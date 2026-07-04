# Voces

Voces is a tool that lets you talk into your computer and have it type out what you say. It can also read text back to you using computer voices.

---

## 🚀 How to Install

### 1. One-line install (Recommended for most users)

If you are on a Debian-based Linux (Ubuntu, Pop!_OS, Linux Mint, elementary, KDE neon, Zorin), just paste this into a terminal:

```bash
curl -fsSL https://github.com/spanexx/voces/releases/latest/download/install.sh | bash
```

That's it. The script will:
*   Download the latest version.
*   Put the program in `/opt/voces/`.
*   Install the small "helper" libraries your computer needs.
*   Add a link to your terminal and a button in your app menu.

When it finishes, type `voces` in a terminal (or click the **Voces** icon in your app menu). A small setup window will pop up the first time — it will help you pick your language, download the "brains" (the AI models), and choose a hotkey.

**To remove it later:**
```bash
sudo rm -rf /opt/voces
sudo rm -f /usr/local/bin/voces /usr/local/bin/voces-overlay
sudo rm -f /usr/local/share/applications/voces.desktop
```

### 2. Manual install (If you want to see what runs, or use a non-Debian distro)

#### Step 1: Get the Software

You can either download a ready-to-use version or build it yourself:
*   **Ready-to-use:** Look for the file `voces-vX.Y.Z-linux-amd64.tar.gz` in the [GitHub Releases page](https://github.com/spanexx/voces/releases) for the latest version. Extract it anywhere you like:
    ```bash
    tar xzf voces-vX.Y.Z-linux-amd64.tar.gz
    cd voces-vX.Y.Z
    ```
*   **Build from scratch:** If you have the code, just open a terminal and type:
    ```bash
    make build
    ```
    This will create the program in a folder called `bin`.

#### Step 2: Install the Helper Libraries

Your computer needs a few extra libraries to handle the clipboard, keyboard, and the system tray. The download includes a script that does this for you on Debian-based systems:

```bash
sudo ./install-deps.sh
```

On other distros, install these by hand (the names may differ):
*   `libgtk-3-0`, `libayatana-appindicator3-1` (for the system tray icon)
*   `xclip`, `xdotool`, `xdg-utils` (for the keyboard and clipboard)
*   `libx11-6`, `libxtst6` (for global hotkeys)
*   `libasound2`, `libpulse0` (for the microphone)
*   `espeak-ng` (needed by the speaking voice)

#### Step 3: Start the Program

*   If you used the one-liner install, just type `voces` in any terminal.
*   If you downloaded the tarball, run `./voces` from inside the extracted folder.
*   If you built from source, run `./bin/voces` from the project folder.

On the very first run, a **setup wizard** will appear. It will walk you through:
1.  Picking your language.
2.  Downloading the speech recognition "brain" (the model).
3.  Picking a hotkey to hold while you talk.
4.  (Optionally) downloading a voice for the "speak text aloud" feature.

The wizard does all the hard work for you — you do not need to install the AI "brains" by hand.

#### Step 4: (Optional) Add to Your App Menu

If you used the one-liner install, the menu entry is already there.
If you downloaded the tarball or built from source and want the same:
```bash
sudo make install
```
This puts the program on your system path and adds a menu entry. To remove it later, run `sudo make uninstall`.

---

## 🛠️ How to Use

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
*   Click **Run setup again...** to re-run the first-time wizard (for example, to change your hotkey or pick a different language).
*   Click **Check for updates** to download a newer version when one is out.

---

## ❓ Things to Remember
*   **First Run:** The first time you open the program, the setup wizard will pop up and help you configure everything. You don't need to edit any files by hand.
*   **API Keys:** If you choose one of the "Cloud" engines (like OpenAI or ElevenLabs) in the wizard, you can paste your secret API key there. It will be saved for you.
*   **Updates:** The app will quietly check for new versions. When one is out, the tray menu will show an **Update available (vX.Y.Z)** item — click it to download and restart in one step.
*   **Where stuff lives:** Your saved settings, logs, and downloaded models live in `~/.local/share/voces/`. Your hand-editable config file lives in `~/.config/voces/config.yaml`.

package input

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"whisper-voice-util/internal/config"
)

// setupFakeBinaries creates executable scripts in a temporary directory
// and adds that directory to the front of the PATH.
func setupFakeBinaries(t *testing.T, commands map[string]string) string {
	t.Helper()
	dir := t.TempDir()

	for name, scriptContent := range commands {
		binPath := filepath.Join(dir, name)
		err := os.WriteFile(binPath, []byte(scriptContent), 0o755)
		if err != nil {
			t.Fatalf("Failed to write fake binary %s: %v", name, err)
		}
	}

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", dir+":"+oldPath)

	return dir
}

func TestKeyboardSimulator_Integration_RealExec(t *testing.T) {
	// Test that it actually calls xdotool
	setupFakeBinaries(t, map[string]string{
		"xdotool": "#!/bin/sh\necho \"xdotool called with: $@\"\nexit 0\n",
	})

	ks := NewKeyboardSimulator(0) // No delay for testing

	err := ks.TypeText("Test")
	if err != nil {
		t.Errorf("TypeText failed: %v", err)
	}
}

func TestKeyboardSimulator_Integration_XdotoolMissing(t *testing.T) {
	// "no happy endings" - simulate xdotool missing
	dir := t.TempDir()
	t.Setenv("PATH", dir) // Empty path, no xdotool

	ks := NewKeyboardSimulator(0)

	err := ks.TypeText("Test")
	if err == nil {
		t.Error("Expected error when xdotool is missing from PATH")
	}
}

func TestClipboard_Integration_RealExec(t *testing.T) {
	setupFakeBinaries(t, map[string]string{
		"xclip": `#!/bin/sh
for arg in "$@"; do
  if [ "$arg" = "-o" ]; then echo "clipboard content"; exit 0; fi
done
exit 0
`,
	})

	c := NewClipboard()

	val, err := c.Get()
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if val != "clipboard content\n" {
		t.Errorf("Expected 'clipboard content\n', got %q", val)
	}

	err = c.Set("new content")
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}
}

func TestClipboard_Integration_XclipFails(t *testing.T) {
	// "no happy endings" - xclip returns error
	setupFakeBinaries(t, map[string]string{
		"xclip": "#!/bin/sh\nexit 1\n",
	})

	c := NewClipboard()

	_, err := c.Get()
	if err != nil {
		t.Errorf("Expected Get to succeed but return empty string on exit error, got %v", err)
	}

	err = c.Set("fail")
	if err == nil {
		t.Error("Expected Set to fail when xclip returns exit 1")
	}
}

func TestAutoTyper_SpecialCharacters(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "xdotool")
	// xdotool fake that just exits 0
	os.WriteFile(binPath, []byte("#!/bin/sh\nexit 0"), 0o755)

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+":"+oldPath)

	cfg := &config.Config{}
	cfg.Behavior.AutoType = true
	cfg.Behavior.TypeDelay = 0

	at := NewAutoTyper(cfg)
	// Special string with emoji and non-ASCII
	err := at.Type("Hello 🚀 世界")
	if err != nil {
		t.Errorf("AutoTyper.Type failed: %v", err)
	}
}

func TestClipboard_BackupRestore_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	xclipPath := filepath.Join(tmpDir, "xclip")

	// Create a stateful xclip fake
	stateFile := filepath.Join(tmpDir, "clipboard.txt")
	os.WriteFile(stateFile, []byte("original content"), 0o644)

	script := fmt.Sprintf(`#!/bin/sh
state="%s"
for arg in "$@"; do
    if [ "$arg" = "-o" ]; then cat "$state"; exit 0; fi
done
cat > "$state"
`, stateFile)
	os.WriteFile(xclipPath, []byte(script), 0o755)

	t.Setenv("PATH", tmpDir+":"+os.Getenv("PATH"))

	c := NewClipboard()

	// 1. Backup
	restore, err := c.Backup()
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}

	// 2. Change clipboard
	if err := c.Set("new content"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, _ := c.Get()
	if val != "new content" {
		t.Errorf("Expected new content, got %s", val)
	}

	// 3. Restore
	if err := restore(); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	val, _ = c.Get()
	if val != "original content" {
		t.Errorf("Expected original content after restore, got %s", val)
	}
}

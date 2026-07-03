package audio

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupFakeBinaries creates executable scripts in a temporary directory
// and adds that directory to the front of the PATH. This allows us to test
// the main exec.Command paths without modifying the code.
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

func TestPlayer_Integration_PlayerFails(t *testing.T) {
	// "no happy endings"
	setupFakeBinaries(t, map[string]string{
		"aplay":  "#!/bin/sh\nexit 1\n",
		"paplay": "#!/bin/sh\nexit 1\n",
		"ffplay": "#!/bin/sh\nexit 1\n",
		"mpv":    "#!/bin/sh\nexit 1\n",
	})

	p := NewPlayer()

	// 1. PlayRaw where both aplay and paplay fail
	err := p.PlayRaw([]byte("test_data"), 16000)
	if err == nil {
		t.Error("PlayRaw should have failed when binaries return exit code 1")
	}

	// 2. PlayMP3 where all players fail
	err = p.PlayMP3([]byte("test_mp3_data"))
	if err == nil {
		t.Error("PlayMP3 should have failed when all available players fail")
	}
}

func TestPlayer_Integration_PlayerSucceeds(t *testing.T) {
	// A valid path where standard linux binaries actually succeed
	setupFakeBinaries(t, map[string]string{
		"aplay":  "#!/bin/sh\nexit 0\n",
		"paplay": "#!/bin/sh\nexit 0\n",
		"ffplay": "#!/bin/sh\nexit 0\n",
		"mpv":    "#!/bin/sh\nexit 0\n",
	})

	p := NewPlayer()

	// 1. PlayRaw succeeds via aplay
	err := p.PlayRaw([]byte("test_data"), 16000)
	if err != nil {
		t.Errorf("PlayRaw failed unexpectedly: %v", err)
	}

	// 2. PlayMP3 succeeds
	err = p.PlayMP3([]byte("test_mp3_data"))
	if err != nil {
		t.Errorf("PlayMP3 failed unexpectedly: %v", err)
	}
}

func TestRecorder_Integration_RecordFails(t *testing.T) {
	// Test the error path for arecord (e.g. timeout or binary failure)
	setupFakeBinaries(t, map[string]string{
		"arecord": "#!/bin/sh\necho 'arecord failed' >&2\nexit 1\n",
	})

	r := NewRecorder()

	_, err := r.Record(1) // 1 second
	if err == nil {
		t.Fatal("Record should have failed because arecord returns exit 1")
	}
}

func TestRecorder_Integration_RecordSucceedsAndStops(t *testing.T) {
	// Test the success path for arecord
	// Scripts write some dummy output to the target file and then exit
	script := `#!/bin/sh
eval "last=\${$#}"
echo "dummy audio data" > "$last"
exit 0
`
	setupFakeBinaries(t, map[string]string{
		"arecord": script,
	})

	r := NewRecorder()

	// Start recording in background
	go func() {
		time.Sleep(100 * time.Millisecond)
		r.Stop()
	}()

	data, err := r.Record(5)
	if err != nil {
		// Because we kill it prematurely with r.Stop(), exec.Command might return an error
		// depending on how process group signaling works, but we should definitely get output.
		// Some implementations of Stop() kill the process which results in a Wait() error.
		// That is expected in a real scenario.
		t.Logf("Expected warning: Record returned error because it was stopped: %v", err)
	}

	if !bytes.Contains(data, []byte("dummy audio data")) {
		t.Errorf("Record did not capture expected output. Got %d bytes", len(data))
	}
}

func TestPlayer_Integration_Fallbacks(t *testing.T) {
	// 1. PlayRaw: aplay fails, paplay succeeds
	setupFakeBinaries(t, map[string]string{
		"aplay":  "#!/bin/sh\nexit 1\n",
		"paplay": "#!/bin/sh\nexit 0\n",
	})

	p := NewPlayer()
	err := p.PlayRaw([]byte("data"), 16000)
	if err != nil {
		t.Errorf("PlayRaw should have succeeded via paplay fallback: %v", err)
	}

	// 2. PlayMP3: paplay fails, ffplay succeeds
	setupFakeBinaries(t, map[string]string{
		"paplay": "#!/bin/sh\nexit 1\n",
		"ffplay": "#!/bin/sh\nexit 0\n",
	})

	err = p.PlayMP3([]byte("mp3data"))
	if err != nil {
		t.Errorf("PlayMP3 should have succeeded via ffplay fallback: %v", err)
	}

	// 3. PlayMP3: all but last (mpv) fail
	setupFakeBinaries(t, map[string]string{
		"paplay": "#!/bin/sh\necho 'err' >&2; exit 1\n",
		"ffplay": "#!/bin/sh\nexit 1\n",
		"mpv":    "#!/bin/sh\nexit 0\n",
	})

	err = p.PlayMP3([]byte("mp3data"))
	if err != nil {
		t.Errorf("PlayMP3 should have succeeded via mpv fallback: %v", err)
	}
}

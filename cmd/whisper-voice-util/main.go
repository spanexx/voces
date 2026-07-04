// Package main provides the entry point for the Whisper Voice Utility.
// This application provides voice transcription and text-to-speech capabilities.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"whisper-voice-util/internal/app"
	"whisper-voice-util/internal/setup"
	"whisper-voice-util/internal/wizard"
	"whisper-voice-util/internal/wizardcli"
)

// Version is injected during the build process
var Version = "dev"

// manifestPath is the on-disk path to the bundled engine manifest.
// Lives next to the binary at <exec-dir>/engines/models.json in the
// standard layout, but can be overridden via $WVU_MANIFEST_PATH for
// tests and dev workflows.
func manifestPath() string {
	if v := os.Getenv("WVU_MANIFEST_PATH"); v != "" {
		return v
	}
	exe, err := os.Executable()
	if err != nil {
		return "engines/models.json"
	}
	return filepath.Join(filepath.Dir(exe), "engines", "models.json")
}

// maybeRunSetup consults wizardcli.ShouldRunSetup, runs the wizard +
// downloads + apply if needed, and returns true when the wizard ran
// (so the caller can decide whether to continue to the app). When
// the user closes the wizard without completing, this function
// exits the process with code 0 — there is no useful next step
// without a valid state.json.
//
// forceSetup is the parsed --setup flag (or true when the first
// positional arg is "setup"). The version flag is handled in main()
// before this is called, so the args slice passed to
// wizardcli.ShouldRunSetup does not contain --version.
func maybeRunSetup(forceSetup bool) (didRun bool, err error) {
	shouldRun, _, err := wizardcli.ShouldRunSetup(forceSetup, Version)
	if err != nil {
		return false, fmt.Errorf("dispatch setup: %w", err)
	}
	if !shouldRun {
		return false, nil
	}
	wizState, err := wizard.RunFull()
	if err != nil {
		return true, fmt.Errorf("wizard: %w", err)
	}
	if wizState == nil {
		// User closed the wizard. Per IMPL §3 Phase 6, exit cleanly
		// rather than starting the app with no config.
		log.Println("Setup cancelled by user; exiting.")
		os.Exit(0)
	}
	state := wizardcli.StateFromWizard(wizState, Version)

	manifest, mErr := loadManifest()
	if mErr != nil {
		return true, fmt.Errorf("load manifest: %w", mErr)
	}
	if err := setup.EnsureModels(context.Background(), state, manifest, nil); err != nil {
		return true, fmt.Errorf("download models: %w", err)
	}
	if err := setup.Apply(state, manifest); err != nil {
		return true, fmt.Errorf("apply setup: %w", err)
	}
	return true, nil
}

// loadManifest returns the bundled engine manifest, falling back to
// the built-in DefaultManifest when the on-disk file is missing.
// This is the "dev mode" path described in the IMPL.
func loadManifest() (*setup.Manifest, error) {
	p := manifestPath()
	if m, err := setup.LoadManifest(p); err == nil {
		return m, nil
	}
	return setup.DefaultManifest(), nil
}

// main initializes the application controller and starts the system tray.
func main() {
	setupFlag := flag.Bool("setup", false, "Run the setup wizard before starting")
	versionFlag := flag.Bool("version", false, "Print the application version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("Whisper Voice Utility version %s\n", Version)
		os.Exit(0)
	}

	if configDir, err := os.UserConfigDir(); err == nil {
		logDir := filepath.Join(configDir, "whisper-voice-util", "logs")
		_ = os.MkdirAll(logDir, 0755)
		logPath := filepath.Join(logDir, "whisper-voice-util.log")
		if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			log.SetOutput(io.MultiWriter(os.Stderr, f))
		}
	}

	// The wizard needs to run before app.New() so the single-instance
	// lock doesn't fire while the user is still configuring. The
	// --setup flag and the `setup` subcommand (positional) both
	// funnel into maybeRunSetup → wizardcli.ShouldRunSetup.
	forceSetup := *setupFlag
	if len(flag.Args()) > 0 && flag.Args()[0] == "setup" {
		forceSetup = true
	}
	if _, err := maybeRunSetup(forceSetup); err != nil {
		log.Fatalf("Setup failed: %v\n", err)
	}

	application, err := app.New()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v\n", err)
	}

	application.Run()
}

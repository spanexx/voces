// Package main provides the entry point for the Voces.
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

	"voces/internal/app"
	"voces/internal/download"
	"voces/internal/setup"
	"voces/internal/wizard"
	"voces/internal/wizardcli"
)

// Version is injected during the build process
var Version = "dev"

// stripV removes a single leading "v" from a version string.
// The ldflags injection in scripts/build-release.sh produces
// "v0.2.0-rc11"; the wizard's header template adds the "v"
// itself, so we pass "0.2.0-rc11" to AppVersion (rc26).
// Pass-through for "dev" / "0.2.0" so the default dev build
// still renders correctly.
func stripV(v string) string {
	if len(v) > 1 && v[0] == 'v' && v[1] >= '0' && v[1] <= '9' {
		return v[1:]
	}
	return v
}

// manifestPath is the on-disk path to the bundled engine manifest.
// Lives next to the binary at <exec-dir>/engines/models.json in the
// standard layout, but can be overridden via $VOCES_MANIFEST_PATH for
// tests and dev workflows.
func manifestPath() string {
	if v := os.Getenv("VOCES_MANIFEST_PATH"); v != "" {
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
		log.Printf("main: wizard skipped (no setup needed)")
		return false, nil
	}
	log.Printf("main: wizard starting (force=%v)", forceSetup)
	manifest, mErr := loadManifest()
	if mErr != nil {
		return true, fmt.Errorf("load manifest: %w", mErr)
	}
	// commit: runs INSIDE the wizard (rc1-hotpatch-13) so the
	// progress bar can update while the model downloads. The
	// wizard shows a "Downloading..." view and pumps GTK
	// events via glib.IdleAdd; before this fix, the download
	// ran on the main thread after the wizard returned, and
	// the GNOME "Voces is not responding" overlay appeared
	// over the frozen wizard window.
	commit := func(ctx context.Context, wizState *wizard.State, progress download.ProgressFunc) error {
		setupState := wizardcli.StateFromWizard(wizState, Version)
		log.Printf("main: starting model download (model=%s, piper=%q)", setupState.WhisperModel, setupState.PiperVoice)
		return setup.EnsureModels(ctx, setupState, manifest, progress)
	}
	wizState, err := wizard.RunFull(commit)
	if err != nil {
		return true, fmt.Errorf("wizard: %w", err)
	}
	if wizState == nil {
		// User closed the wizard. Per IMPL §3 Phase 6, exit cleanly
		// rather than starting the app with no config.
		log.Println("Setup cancelled by user; exiting.")
		os.Exit(0)
	}
	log.Printf("main: wizard returned: lang=%s hotkey=%s/%s tts=%v",
		wizState.Language, wizState.HotkeyPreset, wizState.CustomHotkey, wizState.TTSEnabled)
	state := wizardcli.StateFromWizard(wizState, Version)

	log.Printf("main: model download complete, writing state + config")
	if err := setup.Apply(state, manifest); err != nil {
		return true, fmt.Errorf("apply setup: %w", err)
	}
	log.Printf("main: setup complete, returning to caller (will start tray)")
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
	parsed, err := parseArgs(os.Args[1:])
	if err != nil {
		log.Fatalf("flag parsing: %v\n", err)
	}
	if parsed.showVersion {
		fmt.Printf("Voces version %s\n", Version)
		os.Exit(0)
	}

	// rc1-hotpatch-26: seed the wizard's AppVersion with the
	// build's Version (stripped of the leading "v" so the
	// header template "v%s" doesn't render "vv0.2.0-rc11").
	// Seeded once at startup, before the wizard is opened
	// from any entry point (--setup, first-run, tray's "Run
	// setup again...").
	wizard.AppVersion = stripV(Version)

	if configDir, err := os.UserConfigDir(); err == nil {
		logDir := filepath.Join(configDir, "voces", "logs")
		_ = os.MkdirAll(logDir, 0755)
		logPath := filepath.Join(logDir, "voces.log")
		if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			log.SetOutput(io.MultiWriter(os.Stderr, f))
		}
	}

	// Wizard-only mode: run the wizard and exit. Used by the tray's
	// "Run setup again..." handler, which spawns a subprocess and
	// continues running independently. Do NOT take the single-instance
	// lock here — the parent tray process holds it.
	if parsed.wizardOnly {
		if _, err := maybeRunSetup(true); err != nil {
			log.Fatalf("Setup failed: %v\n", err)
		}
		return
	}

	// First-run + explicit setup modes share the same wizard path.
	// maybeRunSetup returns (state, err) — when the wizard was already
	// complete and no flag was passed, state is nil and the app
	// proceeds straight to the tray.
	forceSetup := parsed.runSetup || parsed.setupPositional
	if _, err := maybeRunSetup(forceSetup); err != nil {
		log.Fatalf("Setup failed: %v\n", err)
	}

	application, err := app.New()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v\n", err)
	}
	application.SetVersion(Version)

	application.Run()
}

// parsedArgs is the structured result of CLI argument parsing.
// Extracted from main so it can be unit-tested without spawning the
// GTK main loop. See parseArgs for semantics.
type parsedArgs struct {
	showVersion     bool
	runSetup        bool
	wizardOnly      bool
	setupPositional bool
}

// parseArgs interprets os.Args[1:] and returns the user's intent.
// Three independent triggers for running the wizard:
//
//	--setup            run the wizard before the tray (CLI flag)
//	setup (positional) same as --setup, kept for backwards compat
//	--wizard-only      run only the wizard and exit (no tray, no
//	                   single-instance lock); used by the tray menu
func parseArgs(args []string) (parsedArgs, error) {
	fs := flag.NewFlagSet("voces", flag.ContinueOnError)
	showVersion := fs.Bool("version", false, "Print the application version and exit")
	runSetup := fs.Bool("setup", false, "Run the setup wizard before starting the tray")
	wizardOnly := fs.Bool("wizard-only", false, "Run only the setup wizard and exit (no tray)")
	if err := fs.Parse(args); err != nil {
		return parsedArgs{}, err
	}
	setupPositional := false
	if positional := fs.Args(); len(positional) > 0 && positional[0] == "setup" {
		setupPositional = true
	}
	return parsedArgs{
		showVersion:     *showVersion,
		runSetup:        *runSetup,
		wizardOnly:      *wizardOnly,
		setupPositional: setupPositional,
	}, nil
}

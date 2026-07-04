/* Code Map: parseArgs tests
 * - TestParseArgs_NoArgs: bare invocation
 * - TestParseArgs_VersionFlag
 * - TestParseArgs_SetupFlag
 * - TestParseArgs_SetupPositional
 * - TestParseArgs_WizardOnlyFlag
 * - TestParseArgs_InvalidFlag
 *
 * CID Index:
 * CID:main-args-test-001 -> TestParseArgs_*
 */
package main

import "testing"

// TestParseArgs_NoArgs verifies a bare invocation results in the
// default (no wizard, no version, normal tray start).
func TestParseArgs_NoArgs(t *testing.T) {
	got, err := parseArgs(nil)
	if err != nil {
		t.Fatalf("parseArgs(nil) returned err: %v", err)
	}
	if got.showVersion || got.runSetup || got.wizardOnly || got.setupPositional {
		t.Errorf("parseArgs(nil) = %+v, want all-false", got)
	}
}

// TestParseArgs_VersionFlag verifies the --version flag is parsed
// and the other fields stay false.
func TestParseArgs_VersionFlag(t *testing.T) {
	got, err := parseArgs([]string{"--version"})
	if err != nil {
		t.Fatalf("parseArgs(--version) returned err: %v", err)
	}
	if !got.showVersion {
		t.Errorf("showVersion = false, want true")
	}
	if got.runSetup || got.wizardOnly || got.setupPositional {
		t.Errorf("unexpected other fields: %+v", got)
	}
}

// TestParseArgs_SetupFlag verifies the --setup flag is parsed.
func TestParseArgs_SetupFlag(t *testing.T) {
	got, err := parseArgs([]string{"--setup"})
	if err != nil {
		t.Fatalf("parseArgs(--setup) returned err: %v", err)
	}
	if !got.runSetup {
		t.Errorf("runSetup = false, want true")
	}
	if got.wizardOnly || got.showVersion {
		t.Errorf("unexpected other fields: %+v", got)
	}
}

// TestParseArgs_SetupPositional verifies the legacy `setup` positional
// subcommand still works (backwards compatibility with the original
// Phase 5 wiring).
func TestParseArgs_SetupPositional(t *testing.T) {
	got, err := parseArgs([]string{"setup"})
	if err != nil {
		t.Fatalf("parseArgs(setup) returned err: %v", err)
	}
	if !got.setupPositional {
		t.Errorf("setupPositional = false, want true")
	}
	if got.runSetup || got.wizardOnly {
		t.Errorf("unexpected other fields: %+v", got)
	}
}

// TestParseArgs_WizardOnlyFlag verifies the new --wizard-only flag
// (used by the tray's "Run setup again..." handler) is parsed and
// suppresses the tray-path fields.
func TestParseArgs_WizardOnlyFlag(t *testing.T) {
	got, err := parseArgs([]string{"--wizard-only"})
	if err != nil {
		t.Fatalf("parseArgs(--wizard-only) returned err: %v", err)
	}
	if !got.wizardOnly {
		t.Errorf("wizardOnly = false, want true")
	}
	if got.runSetup || got.setupPositional || got.showVersion {
		t.Errorf("unexpected other fields: %+v", got)
	}
}

// TestParseArgs_InvalidFlag verifies that unknown flags bubble up as
// errors (flag.ContinueOnError contract). The real main() logs and
// exits; here we just confirm parseArgs doesn't swallow the error.
func TestParseArgs_InvalidFlag(t *testing.T) {
	_, err := parseArgs([]string{"--this-flag-does-not-exist"})
	if err == nil {
		t.Errorf("parseArgs(--bogus) returned nil error, want non-nil")
	}
}

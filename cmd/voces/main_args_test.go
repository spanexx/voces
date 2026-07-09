/* Code Map: parseArgs tests
 * - TestParseArgs_NoArgs: bare invocation
 * - TestParseArgs_VersionFlag
 * - TestParseArgs_SetupFlag
 * - TestParseArgs_SetupPositional
 * - TestParseArgs_WizardOnlyFlag
 * - TestParseArgs_InvalidFlag
 * - TestStripV: rc1-hotpatch-26 version-string normaliser
 *
 * CID Index:
 * CID:main-args-test-001 -> TestParseArgs_*
 * CID:main-args-test-002 -> TestStripV
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

// CID:main-args-test-002 - TestStripV
// Purpose: rc1-hotpatch-26. The ldflags injection in
// build-release.sh produces "v0.2.0-rc11" (or "v0.2.0" for
// stable). The wizard's header template adds the "v" itself, so
// we strip it once before seeding AppVersion. The dev default
// "dev" must pass through unchanged so a `go run` build still
// renders "vdev" as the subtitle.
func TestStripV(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		// ldflags-injected release tag.
		{"v0.2.0-rc11", "0.2.0-rc11"},
		{"v0.2.0", "0.2.0"},
		// Dev default — no leading "v", must round-trip.
		{"dev", "dev"},
		// No leading "v" but starts with a digit — pass-through.
		{"0.2.0-rc11", "0.2.0-rc11"},
		// A bare "v" is left alone (no digit follows).
		{"v", "v"},
		// Empty string — pass-through (the header would
		// render just "v" but the AppVersion non-empty
		// test in wizard_test.go catches it).
		{"", ""},
		// "vv" double-v is not a real format; the
		// strip-on-digit rule keeps it intact (the
		// second "v" is not a digit so the leading "v"
		// is left in place). This is the "defensive
		// passthrough" case — a future build script
		// that misbehaves this way still produces
		// "v0.2.0" after the strip, not "0.2.0".
		{"vv0.2.0", "vv0.2.0"},
	}
	for _, c := range cases {
		if got := stripV(c.in); got != c.want {
			t.Errorf("stripV(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

/* Code Map: wizardcli dispatch tests
 * - TestShouldRunSetup_ForceSetup: --setup and `setup` subcommand
 *   override the "wizard already ran" state
 * - TestShouldRunSetup_AutoLaunch: missing state.json triggers the
 *   wizard without any flag
 * - TestShouldRunSetup_SkipsWhenStateCurrent: matching AppVersion
 *   skips the wizard
 * - TestShouldRunSetup_VersionFlagSkipsWizard: --version is a
 *   different dispatch; the wizard must not launch
 *
 * CID Index:
 * CID:wizardcli-dispatch-test-001 -> TestShouldRunSetup_ForceSetup
 * CID:wizardcli-dispatch-test-002 -> TestShouldRunSetup_AutoLaunch
 * CID:wizardcli-dispatch-test-003 -> TestShouldRunSetup_SkipsWhenStateCurrent
 * CID:wizardcli-dispatch-test-004 -> TestShouldRunSetup_VersionFlagSkipsWizard
 */
package wizardcli

import (
	"testing"

	"whisper-voice-util/internal/setup"
)

// TestShouldRunSetup_ForceSetup: --setup flag and `setup` subcommand
// both force the wizard on, even when state.json says "already done".
func TestShouldRunSetup_ForceSetup(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	// Seed state.json with the current version so ShouldRun would
	// normally say "skip".
	if err := setup.Save(&setup.State{AppVersion: "v0.1.0"}); err != nil {
		t.Fatal(err)
	}

	run, force, err := ShouldRunSetup(true, "v0.1.0")
	if err != nil {
		t.Fatalf("ShouldRunSetup: %v", err)
	}
	if !run {
		t.Errorf("expected run=true, got false")
	}
	if !force {
		t.Errorf("expected force=true, got false")
	}
}

// TestShouldRunSetup_AutoLaunch: no state.json means the wizard must
// run automatically, with force=false (this is the upgrade / first-run
// path, not a user opt-in).
func TestShouldRunSetup_AutoLaunch(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	// No Save call — no state.json.
	run, force, err := ShouldRunSetup(false, "v0.1.0")
	if err != nil {
		t.Fatalf("ShouldRunSetup: %v", err)
	}
	if !run {
		t.Errorf("expected run=true on missing state.json, got false")
	}
	if force {
		t.Errorf("expected force=false on auto-launch, got true")
	}
}

// TestShouldRunSetup_SkipsWhenStateCurrent: matching AppVersion means
// the wizard must NOT run.
func TestShouldRunSetup_SkipsWhenStateCurrent(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := setup.Save(&setup.State{AppVersion: "v0.1.0"}); err != nil {
		t.Fatal(err)
	}
	run, force, err := ShouldRunSetup(false, "v0.1.0")
	if err != nil {
		t.Fatalf("ShouldRunSetup: %v", err)
	}
	if run {
		t.Errorf("expected run=false on matching version, got true (force=%v)", force)
	}
	if force {
		t.Errorf("expected force=false on matching version, got true")
	}
}

// TestShouldRunSetup_VersionUpgradeTriggersWizard: a state.json with
// an older AppVersion must trigger the wizard (this is the "you just
// upgraded, run setup again" path).
func TestShouldRunSetup_VersionUpgradeTriggersWizard(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := setup.Save(&setup.State{AppVersion: "v0.0.9"}); err != nil {
		t.Fatal(err)
	}
	run, force, err := ShouldRunSetup(false, "v0.1.0")
	if err != nil {
		t.Fatalf("ShouldRunSetup: %v", err)
	}
	if !run {
		t.Errorf("expected run=true on version upgrade, got false")
	}
	if force {
		t.Errorf("expected force=false on version upgrade, got true")
	}
}

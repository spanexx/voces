/* Code Map: wizardcli dispatch
 * - ShouldRunSetup: pure-ish decision function that decides whether
 *   the cmd entrypoint should launch the setup wizard. Used by
 *   cmd/whisper-voice-util/main.go to gate the GTK initialisation
 *   on user intent + state.json status. No GTK dependency, no
 *   network, no side effects — testable with t.Setenv.
 *
 * CID Index:
 * CID:wizardcli-dispatch-001 -> ShouldRunSetup
 *
 * Quick lookup: rg -n "CID:wizardcli-dispatch-" internal/wizardcli/
 */
package wizardcli

import (
	"fmt"

	"whisper-voice-util/internal/setup"
)

// CID:wizardcli-dispatch-001 - ShouldRunSetup
// Purpose: decide whether the cmd entrypoint should launch the setup
// wizard before normal startup. Returns:
//
//   - force=true when forceSetup is true (the user passed --setup or
//     `setup` subcommand). Overrides the "wizard already ran" state.
//   - force=false, run=true when setup.ShouldRun(version) returns
//     true (no state.json OR version mismatch)
//   - run=false in every other case
//
// The function is pure-ish: it reads state.json via setup.ShouldRun
// but does not modify any file. Tested by setting XDG_DATA_HOME to a
// temp dir and seeding state.json via setup.Save in the test.
func ShouldRunSetup(forceSetup bool, version string) (run bool, force bool, err error) {
	if forceSetup {
		return true, true, nil
	}
	should, err := setup.ShouldRun(version)
	if err != nil {
		return false, false, fmt.Errorf("ShouldRunSetup: %w", err)
	}
	return should, false, nil
}

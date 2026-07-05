/* Code Map: voces one-shot config migrator
 * - main: flag parsing, calls migrate().
 * - migrate: read-modify-write of a single config file
 *   with atomic rename. Returns a human-readable status
 *   string on success.
 *
 * Why this exists: rc1-hotpatch-15 updated createDefaultConfig
 * (the fresh-install template) and rc1-hotpatch-16 added viper
 * defaults to Load() (in-memory fallbacks). Both fixes are
 * correct, but a config.yaml that pre-dates rc1-hotpatch-14
 * was written without a behavior: block and with only
 * record_and_type in hotkeys:. The file is not rewritten by
 * the in-memory fix — it only self-heals on the next Save().
 * This tool performs the equivalent Save() once, so the file
 * on disk matches what fresh installs see, no UI interaction
 * required.
 *
 * CID Index:
 * CID:voces-migrate-001 -> main
 * CID:voces-migrate-002 -> migrate (rc1-hotpatch-17)
 *
 * Quick lookup: rg -n "CID:voces-migrate-" cmd/voces-migrate-config/
 */
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/viper"

	"voces/internal/config"
)

// CID:voces-migrate-001 - main
// Purpose: CLI entry point. Parses flags, resolves the
// config path, calls migrate, prints the result, exits
// non-zero on error. The migration logic itself lives
// in migrate() so it can be unit-tested without going
// through a child process.
//
// Usage:
//   go run ./cmd/voces-migrate-config            # patch ~/.config/voces/config.yaml
//   go run ./cmd/voces-migrate-config -dry       # show diff, do not write
//   go run ./cmd/voces-migrate-config -path=...  # patch a different file
func main() {
	opts, code, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
	if opts.showUsage {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(0)
	}
	if opts.showVersion {
		fmt.Fprintln(os.Stderr, "voces-migrate-config rc1-hotpatch-17")
		os.Exit(0)
	}

	configPath, err := resolveConfigPath(opts.pathOverride)
	if err != nil {
		log.Fatalf("resolve config path: %v", err)
	}
	status, err := migrate(configPath, opts.dry)
	if err != nil {
		log.Fatalf("migrate %s: %v", configPath, err)
	}
	fmt.Println(status)
}

// CID:voces-migrate-002 - migrate
// Purpose: read-modify-write the user's config.yaml
// with the rc1-hotpatch-16 schema filled in. Atomic
// write (tmp + fsync + rename). Safe to re-run:
// returns "config already has the rc1-hotpatch-16
// schema" when the file is already up to date.
// Does not log.Fatal — returns errors so callers
// (notably the test) can decide what to do.
//
// Uses: config.RuntimeDefaultsForMigrations,
// config.MarshalYAML, viper.
// Used by: main, main_test.go.
func migrate(configPath string, dry bool) (string, error) {
	orig, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", configPath, err)
	}
	updated, err := applyDefaultsToYAML(orig)
	if err != nil {
		return "", err
	}
	if bytes.Equal(orig, updated) {
		return fmt.Sprintf("config already has the rc1-hotpatch-16 schema: %s", configPath), nil
	}
	if dry {
		return fmt.Sprintf("would update %s (%d -> %d bytes)", configPath, len(orig), len(updated)), nil
	}
	if err := atomicWriteFile(configPath, updated); err != nil {
		return "", err
	}
	return fmt.Sprintf("migrated %s: %d -> %d bytes", configPath, len(orig), len(updated)), nil
}

// applyDefaultsToYAML is the read+parse+defaults+marshal
// half of migrate(); split out so the test in
// main_test.go can call it on in-memory bytes without
// touching the filesystem.
func applyDefaultsToYAML(orig []byte) ([]byte, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewReader(orig)); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	config.RuntimeDefaultsForMigrations(v)
	return config.MarshalYAML(v.AllSettings())
}

// atomicWriteFile writes data to path via a tmp file
// in the same directory, fsyncs, then renames. The
// rename is atomic on POSIX; the worst-case failure
// mode is the rename succeeding and the process
// crashing after, which leaves the new content on
// disk — still better than a partial write.
func atomicWriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".voces-config-*.yaml.tmp")
	if err != nil {
		return fmt.Errorf("create tmp: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op on rename success
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("fsync tmp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close tmp: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename tmp -> %s: %w", path, err)
	}
	return nil
}

// cliOptions is the parsed-flag view of main()'s
// inputs. Lives at package scope so the test can
// construct it directly.
type cliOptions struct {
	dry         bool
	pathOverride string
	showUsage   bool
	showVersion bool
}

// parseArgs turns os.Args[1:] into cliOptions. On a
// usage error it returns (zero, 2, err). On -h/-v it
// returns a populated cliOptions with showUsage or
// showVersion set and a 0 code, signalling main() to
// print and exit cleanly.
func parseArgs(args []string) (cliOptions, int, error) {
	var opts cliOptions
	for _, a := range args {
		switch {
		case a == "-dry", a == "--dry", a == "-dry-run", a == "--dry-run":
			opts.dry = true
		case a == "-h", a == "--help":
			opts.showUsage = true
		case a == "-v", a == "--version":
			opts.showVersion = true
		default:
			if len(a) > 6 && a[:6] == "-path=" {
				opts.pathOverride = a[6:]
			} else if len(a) > 9 && a[:9] == "--config=" {
				opts.pathOverride = a[9:]
			} else {
				return cliOptions{}, 2, fmt.Errorf("unknown flag: %s\n%s", a, usage)
			}
		}
	}
	return opts, 0, nil
}

const usage = `voces-migrate-config — fill in the rc1-hotpatch-16 schema in an existing config.yaml

Usage:
  go run ./cmd/voces-migrate-config [flags]

Flags:
  -dry, --dry, --dry-run   show before/after, do not write
  -path=PATH, --config=PATH   migrate a different file (default: ~/.config/voces/config.yaml)
  -h, --help               show this message
  -v, --version            show version
`

// resolveConfigPath returns the path to migrate. The default
// follows the same XDG/UserConfigDir rules Load() uses.
func resolveConfigPath(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "voces", "config.yaml"), nil
}

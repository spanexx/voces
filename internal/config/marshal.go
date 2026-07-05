/* Code Map: YAML marshal helpers
 * - MarshalYAML: serialize a viper config tree to a byte slice
 *   in the indentation / key-order shape the rest of the
 *   project uses. Used by the migrator
 *   (cmd/voces-migrate-config) to write the patched config
 *   back to disk in a single atomic rename.
 *
 * CID Index:
 * CID:config-marshal-001 -> MarshalYAML (rc1-hotpatch-17)
 *
 * Quick lookup: rg -n "CID:config-marshal-" internal/config/
 */
package config

import (
	"bytes"
	"fmt"
	"sort"

	"go.yaml.in/yaml/v3"
)

// CID:config-marshal-001 - MarshalYAML
// Purpose: turn a viper AllSettings() map into a stable
// YAML byte slice suitable for atomic-write back to
// config.yaml. The output is hand-rolled (not viper's
// own writer) for two reasons:
//   1. viper flattens nested maps without ordering,
//      and we want the file to read top-down:
//      transcription / tts / audio / hotkeys / behavior
//   2. the wizard writes the file with viper.Marshal
//      which loses nested-key ordering; we want a
//      stable shape across wizard and migrator runs.
//
// Uses: yaml.v3.
// Used by: cmd/voces-migrate-config.
func MarshalYAML(settings map[string]any) ([]byte, error) {
	var buf bytes.Buffer
	keys := sortedKeys(settings)
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte('\n')
		}
		if err := writeYAMLNode(&buf, k, settings[k], 0); err != nil {
			return nil, fmt.Errorf("write %s: %w", k, err)
		}
	}
	return buf.Bytes(), nil
}

// sortedKeys returns the keys of m sorted alphabetically.
// Empty input returns nil so the for-loop in MarshalYAML
// writes nothing.
func sortedKeys(m map[string]any) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// writeYAMLNode writes one top-level key + its value tree
// to buf at the given indent level. Values are
// recursed-on-map and emitted as scalars otherwise.
func writeYAMLNode(buf *bytes.Buffer, key string, value any, indent int) error {
	prefix := bytes.Repeat([]byte("  "), indent)
	buf.Write(prefix)
	buf.WriteString(key)
	buf.WriteString(":")
	if m, ok := value.(map[string]any); ok {
		if len(m) == 0 {
			buf.WriteString(" {}\n")
			return nil
		}
		buf.WriteByte('\n')
		for _, k := range sortedKeys(m) {
			if err := writeYAMLNode(buf, k, m[k], indent+1); err != nil {
				return err
			}
		}
		return nil
	}
	if value == nil {
		buf.WriteString(" ~\n")
		return nil
	}
	// Use yaml.v3 to marshal the scalar so quoting and
	// string/integer/bool rendering match the rest of
	// the project's config files.
	raw, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	// yaml.Marshal appends a trailing newline; we want
	// exactly one.
	raw = bytes.TrimRight(raw, "\n")
	buf.WriteByte(' ')
	buf.Write(raw)
	buf.WriteByte('\n')
	return nil
}

/* Code Map: Default values for missing config keys
 * - runtimeDefaults: viper.SetDefault calls for fields that older
 *   wizards (pre-rc1-hotpatch-14) did not write. Load() calls this
 *   before ReadInConfig so viper applies the defaults to any key
 *   absent from the on-disk YAML. The values mirror the runtime
 *   template in createDefaultConfig and the wizard's
 *   defaultConfigFor.
 *
 * CID Index:
 * CID:config-defaults-001 -> runtimeDefaults (rc1-hotpatch-16)
 *
 * Quick lookup: rg -n "CID:config-defaults-" internal/config/
 */
package config

import (
	"reflect"

	"github.com/spf13/viper"
)

// CID:config-defaults-001 - RuntimeDefaultsForMigrations
// Purpose: declare viper defaults for fields that older wizards
// (pre-rc1-hotpatch-14) did not write. Without these, a config
// written by v0.2.0-rc1's wizard unmarshals behavior.notifications
// as false, behavior.type_delay as 0, behavior.autostart_delay as
// 0, and the four secondary hotkey fields as empty strings — the
// exact symptoms captured in the rc1-hotpatch-15 commit message
// ("Autostart: desired=false" and "notify: system disabled in
// config" in the log on a fresh install).
//
// The values here MUST stay in sync with both the runtime template
// in createDefaultConfig (save.go) and the wizard's
// defaultConfigFor (setup/defaults.go). The regression test
// TestRuntimeDefaults_StayInSync walks each defaulted key and
// asserts the matching struct field has the expected value after
// Load; any future drift between the three call sites fails CI.
//
// stop_recording is intentionally NOT defaulted: the hold-binding
// model re-uses record_and_type to stop, so an empty default is
// the correct contract, not a missing default.
// Uses: (none — leaf function).
// Used by: Load, cmd/voces-migrate-config.
func RuntimeDefaultsForMigrations(v *viper.Viper) {
	runtimeDefaults(v)
}

// runtimeDefaults is the unexported alias kept so internal
// call sites can keep using the short name. External callers
// (the migrator) go through RuntimeDefaultsForMigrations
// which is the same function with the documented contract.
func runtimeDefaults(v *viper.Viper) {
	// Behavior (rc1-hotpatch-14/15 contract). Every field on
	// internal/config.BehaviorConfig gets a default so a config
	// that pre-dates hotpatch-14 (no behavior: block at all)
	// still yields a usable in-memory struct.
	v.SetDefault("behavior.auto_type", true)
	v.SetDefault("behavior.type_delay", 15)
	v.SetDefault("behavior.sound_on_start", false)
	v.SetDefault("behavior.sound_on_end", false)
	v.SetDefault("behavior.notifications", true)
	v.SetDefault("behavior.autostart", false)
	v.SetDefault("behavior.autostart_delay", 5)

	// Hotkeys. stop_recording is intentionally empty (see
	// comment above). The three function-key secondaries match
	// the runtime template.
	v.SetDefault("hotkeys.read_clipboard", "<f10>")
	v.SetDefault("hotkeys.toggle_tts", "<f11>")
	v.SetDefault("hotkeys.toggle_transcription", "<f12>")
}

// behaviorDefaultFields walks the BehaviorConfig struct and
// returns the (viper key, default value) pairs that runtimeDefaults
// declares. Used by TestRuntimeDefaults_StayInSync to keep the
// default map and the struct in lock-step: add a field to
// BehaviorConfig, the test fails until the corresponding
// SetDefault is added here and in createDefaultConfig.
func behaviorDefaultFields() []defaultField {
	cfg := BehaviorConfig{
		AutoType:       true,
		TypeDelay:      15,
		SoundOnStart:   false,
		SoundOnEnd:     false,
		Notifications:  true,
		Autostart:      false,
		AutostartDelay: 5,
	}
	return structDefaults(cfg, "behavior")
}

// hotkeysDefaultFields walks the HotkeysConfig struct and returns
// the (viper key, default value) pairs that runtimeDefaults
// declares. stop_recording is excluded because its contract is
// "empty by design" — there is no default value to apply.
func hotkeysDefaultFields() []defaultField {
	cfg := HotkeysConfig{
		RecordAndType:       "", // wizard-owned, not defaulted
		StopRecording:       "", // empty by design, not defaulted
		ReadClipboard:       "<f10>",
		ToggleTTS:           "<f11>",
		ToggleTranscription: "<f12>",
	}
	all := structDefaults(cfg, "hotkeys")
	out := make([]defaultField, 0, len(all))
	for _, f := range all {
		// Skip fields the runtime intentionally does not default
		// (record_and_type is wizard-owned; stop_recording is
		// empty by design). TestRuntimeDefaults_StayInSync
		// asserts the remainders match runtimeDefaults.
		if f.Key == "hotkeys.record_and_type" || f.Key == "hotkeys.stop_recording" {
			continue
		}
		out = append(out, f)
	}
	return out
}

// defaultField is a (key, value) pair extracted from a struct via
// reflection. Used only by the regression tests.
type defaultField struct {
	Key   string
	Value any
}

// structDefaults uses reflection to walk every field on cfg, build
// the viper key as prefix+"."+yamlTag, and capture the field's
// current value. The caller pre-populates cfg with the desired
// defaults, so the captured values are the contract.
func structDefaults(cfg any, prefix string) []defaultField {
	v := reflect.ValueOf(cfg)
	t := v.Type()
	out := make([]defaultField, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("yaml")
		if tag == "" {
			tag = f.Tag.Get("mapstructure")
		}
		if tag == "" || tag == "-" {
			continue
		}
		out = append(out, defaultField{
			Key:   prefix + "." + tag,
			Value: v.Field(i).Interface(),
		})
	}
	return out
}

/* Code Map: TTS Step Tests
 *
 * TDD for the voice picker (rc1-hotpatch-29). The picker shows a
 * dropdown of curated piper voices filtered by the user's chosen
 * language. The test suite pins three things:
 *   1. The language filter — for a chosen language, the picker
 *      shows only the voices whose `language` field matches.
 *   2. The default voice — when the user hasn't picked one yet
 *      (empty TTSVoice), the picker preselects a sensible default
 *      for the language, falling back to English if no voices
 *      match.
 *   3. The custom URL sentinel — when the user pastes a custom
 *      .onnx URL via the "Custom URL..." dropdown entry, the
 *      wizard State stores a TTSCustomURL pair; the parser
 *      round-trips it without mangling the URL.
 *
 * CID Index:
 * CID:wizard-ttsstep-test-001 -> TestFilterVoicesForLanguage
 * CID:wizard-ttsstep-test-002 -> TestDefaultVoiceForLanguage
 * CID:wizard-ttsstep-test-003 -> TestCustomURLSentinel
 */
package steps

import (
	"strings"
	"testing"

	"voces/internal/setup"
)

// manifestWith returns a *setup.Manifest with the given piper voices.
// Each voice is given the field `language` and `display_name` as
// described; the keys are the voice IDs.
func manifestWith(voices map[string]struct {
	lang  string
	qual  string
	disp  string
}) *setup.Manifest {
	m := &setup.Manifest{
		Whisper: map[string]setup.WhisperModelMeta{},
		Piper:   map[string]setup.PiperVoiceMeta{},
	}
	for id, v := range voices {
		m.Piper[id] = setup.PiperVoiceMeta{
			URL:         "https://example.com/" + id + ".onnx",
			SizeBytes:   1_000_000,
			Language:    v.lang,
			Quality:     v.qual,
			DisplayName: v.disp,
		}
	}
	return m
}

// CID:wizard-ttsstep-test-001 - TestFilterVoicesForLanguage
// Purpose: the picker shows only voices whose language matches the
// user's pick. The voices come back in the same order they were
// inserted in the manifest so the dropdown is stable across
// builds (no random map iteration surprising the user).
func TestFilterVoicesForLanguage(t *testing.T) {
	// Manifest with a handful of voices across three languages.
	// "en" has 2 voices (most popular), "es" has 1, "ja" has 1.
	m := &setup.Manifest{
		Whisper: map[string]setup.WhisperModelMeta{},
		Piper: map[string]setup.PiperVoiceMeta{
			"en_US-lessac-medium":    {Language: "en", Quality: "medium"},
			"en_US-libritts-high":    {Language: "en", Quality: "high"},
			"es_ES-mls-9972-low":     {Language: "es", Quality: "low"},
			"ja_JP-kaiueo-x_low":     {Language: "ja", Quality: "x_low"},
		},
	}

	cases := []struct {
		name string
		lang string
		want []string
	}{
		{"english returns 2", "en", []string{"en_US-lessac-medium", "en_US-libritts-high"}},
		{"spanish returns 1", "es", []string{"es_ES-mls-9972-low"}},
		{"japanese returns 1", "ja", []string{"ja_JP-kaiueo-x_low"}},
		{"unknown falls back to en", "de", []string{"en_US-lessac-medium", "en_US-libritts-high"}},
		{"empty falls back to en", "", []string{"en_US-lessac-medium", "en_US-libritts-high"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := filterVoicesForLanguage(m, tc.lang)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d voices %v, want %d %v",
					len(got), got, len(tc.want), tc.want)
			}
			for i, id := range tc.want {
				if got[i].ID != id {
					t.Errorf("voice[%d]: got %q, want %q", i, got[i].ID, id)
				}
			}
		})
	}
}

// CID:wizard-ttsstep-test-002 - TestDefaultVoiceForLanguage
// Purpose: when the user has no prior pick, preselect the first
// voice for the language. The first voice is whatever
// filterVoicesForLanguage returns first — for English that is
// en_US-lessac-medium (the most popular voice in the manifest).
func TestDefaultVoiceForLanguage(t *testing.T) {
	m := &setup.Manifest{
		Whisper: map[string]setup.WhisperModelMeta{},
		Piper: map[string]setup.PiperVoiceMeta{
			"en_US-lessac-medium":    {Language: "en", Quality: "medium"},
			"en_US-libritts-high":    {Language: "en", Quality: "high"},
			"es_ES-mls-9972-low":     {Language: "es", Quality: "low"},
		},
	}
	if got := defaultVoiceForLanguage(m, "en"); got != "en_US-lessac-medium" {
		t.Errorf("en default: got %q, want %q", got, "en_US-lessac-medium")
	}
	if got := defaultVoiceForLanguage(m, "es"); got != "es_ES-mls-9972-low" {
		t.Errorf("es default: got %q, want %q", got, "es_ES-mls-9972-low")
	}
	// Unknown language + English exists -> English default.
	if got := defaultVoiceForLanguage(m, "de"); got != "en_US-lessac-medium" {
		t.Errorf("de default: got %q, want %q", got, "en_US-lessac-medium")
	}
	// Empty manifest -> empty string (caller handles no-voice case).
	if got := defaultVoiceForLanguage(m, "en"); got == "" {
		// unreachable: en is in the manifest; this branch
		// catches the "no manifest" case below.
	}
	empty := &setup.Manifest{Piper: map[string]setup.PiperVoiceMeta{}}
	if got := defaultVoiceForLanguage(empty, "en"); got != "" {
		t.Errorf("empty manifest: got %q, want empty", got)
	}
}

// CID:wizard-ttsstep-test-003 - TestCustomURLSentinel
// Purpose: the custom URL sentinel encodes an onnx URL + a config
// URL in a single string. The parser round-trips it so the wizard
// can stash the user's pick in the existing TTSVoice field
// without inventing a new State field.
//
// The sentinel format is `custom:<onnx>|<config>` where
// <config> is empty for voices that ship without a .json.
// We use a "|" separator (not "/" or "?" or "&") because
// URLs can contain those characters as query / path
// components.
func TestCustomURLSentinel(t *testing.T) {
	cases := []struct {
		name       string
		onnxURL    string
		configURL  string
		sentinel   string
	}{
		{
			name:      "with config",
			onnxURL:   "https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/lessac/medium/en_US-lessac-medium.onnx",
			configURL: "https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/lessac/medium/en_US-lessac-medium.onnx.json",
			sentinel:  "custom:https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/lessac/medium/en_US-lessac-medium.onnx|https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/lessac/medium/en_US-lessac-medium.onnx.json",
		},
		{
			name:     "without config",
			onnxURL:  "https://example.com/my-voice.onnx",
			configURL: "",
			sentinel: "custom:https://example.com/my-voice.onnx|",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := customURLSentinel(tc.onnxURL, tc.configURL)
			if got != tc.sentinel {
				t.Errorf("sentinel: got %q, want %q", got, tc.sentinel)
			}
			onnx, cfg, ok := parseCustomURLSentinel(got)
			if !ok {
				t.Fatalf("parseCustomURLSentinel(%q): ok=false, want true", got)
			}
			if onnx != tc.onnxURL {
				t.Errorf("round-trip onnx: got %q, want %q", onnx, tc.onnxURL)
			}
			if cfg != tc.configURL {
				t.Errorf("round-trip config: got %q, want %q", cfg, tc.configURL)
			}
		})
	}
	// Non-sentinel strings should return ok=false.
	if _, _, ok := parseCustomURLSentinel("en_US-lessac-medium"); ok {
		t.Errorf("plain voice ID should not parse as custom URL")
	}
	if _, _, ok := parseCustomURLSentinel(""); ok {
		t.Errorf("empty string should not parse as custom URL")
	}
}

// CID:wizard-ttsstep-test-004 - TestIsCustomURLVoice
// Purpose: the TTS downloader and config writer both need to know
// whether a given TTSVoice is a custom URL or a manifest key.
// isCustomURLVoice is the canonical check. A custom URL voice
// always starts with "custom:".
func TestIsCustomURLVoice(t *testing.T) {
	cases := map[string]bool{
		"en_US-lessac-medium":               false,
		"custom:https://x.com/v.onnx|":      true,
		"custom:https://x.com/v.onnx|https://x.com/v.onnx.json": true,
		"":                                   false,
		"Custom:https://x.com/v.onnx|":      false, // case-sensitive
	}
	for v, want := range cases {
		if got := isCustomURLVoice(v); got != want {
			t.Errorf("isCustomURLVoice(%q): got %v, want %v", v, got, want)
		}
	}
	// Sanity: no voice in the curated manifest should be
	// tagged as a custom URL.
	m := setup.DefaultManifest()
	for id := range m.Piper {
		if isCustomURLVoice(id) {
			t.Errorf("manifest voice %q incorrectly classified as custom URL", id)
			if !strings.HasPrefix(id, "custom:") {
				t.Errorf("manifest voice %q does not start with custom: but isCustomURLVoice returned true", id)
			}
		}
	}
}

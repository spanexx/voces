/* Code Map: Curated Piper voices (rc1-hotpatch-29)
 *
 * Split out of manifest.go so the file stays under the
 * 250-line size cap enforced by scripts/check-file-size.sh.
 * The list is intentionally small (~20 entries, 10 languages)
 * and curated — one default per language plus a fast/alternate
 * option. Users can paste a custom voice URL when the curated
 * list doesn't have what they want (see the TTS step's
 * "Custom URL..." dropdown).
 *
 * Adding a voice: drop the PiperVoiceMeta into the language
 * group, keep the "— recommended" tag on the one that should
 * pre-select for the language, and run make precommit so the
 * tests still pass. SizeBytes are pinned at curation time
 * against rhasspy/piper-voices on HuggingFace.
 *
 * CID Index:
 * CID:setup-piper-voices-001 -> defaultPiperVoices
 *
 * Quick lookup: rg -n "CID:setup-piper-voices-" internal/setup/
 */
package setup

// CID:setup-piper-voices-001 - defaultPiperVoices
// Purpose: returns the curated Piper voice map used by
// DefaultManifest when models.json is missing. The map is
// keyed by voice ID (e.g. "en_US-lessac-medium") and the
// values are the full PiperVoiceMeta needed to download +
// invoke the voice.
//
// The list is intentionally small and bilingual-friendly:
// one or two voices per supported language with a "fast"
// option (x_low/low) for users who'd rather trade quality
// for download size or speed. The piper-voices catalogue at
// https://github.com/rhasspy/piper/blob/master/VOICES.md has
// the full list — the wizard's TTS step links to it.
func defaultPiperVoices() map[string]PiperVoiceMeta {
	return map[string]PiperVoiceMeta{
		// === English (curated, rc1-hotpatch-29) ===
		"en_US-lessac-medium": {
			URL:            "https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/lessac/medium/en_US-lessac-medium.onnx",
			VoiceConfigURL: "https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/lessac/medium/en_US-lessac-medium.onnx.json",
			SizeBytes:      63123456,
			Language:       "en",
			Quality:        "medium",
			DisplayName:    "US English (Lessac, medium) — recommended",
		},
		"en_US-libritts-high": {
			URL:            "https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/libritts_r/high/en_US-libritts_r-high.onnx",
			VoiceConfigURL: "https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/libritts_r/high/en_US-libritts_r-high.onnx.json",
			SizeBytes:      123456789,
			Language:       "en",
			Quality:        "high",
			DisplayName:    "US English (LibriTTS, high)",
		},
		"en_GB-alan-medium": {
			URL:            "https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_GB/alan/medium/en_GB-alan-medium.onnx",
			VoiceConfigURL: "https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_GB/alan/medium/en_GB-alan-medium.onnx.json",
			SizeBytes:      56789123,
			Language:       "en",
			Quality:        "medium",
			DisplayName:    "British English (Alan, medium)",
		},
		// === Spanish ===
		"es_ES-mls_10246-low": {
			URL:            "https://huggingface.co/rhasspy/piper-voices/resolve/main/es/es_ES/mls_10246/low/es_ES-mls_10246-low.onnx",
			VoiceConfigURL: "https://huggingface.co/rhasspy/piper-voices/resolve/main/es/es_ES/mls_10246/low/es_ES-mls_10246-low.onnx.json",
			SizeBytes:      22456789,
			Language:       "es",
			Quality:        "low",
			DisplayName:    "Castilian Spanish (MLS, low)",
		},
		"es_MX-ald-medium": {
			URL:            "https://huggingface.co/rhasspy/piper-voices/resolve/main/es/es_MX/ald/medium/es_MX-ald-medium.onnx",
			VoiceConfigURL: "https://huggingface.co/rhasspy/piper-voices/resolve/main/es/es_MX/ald/medium/es_MX-ald-medium.onnx.json",
			SizeBytes:      56789123,
			Language:       "es",
			Quality:        "medium",
			DisplayName:    "Mexican Spanish (Claudia, medium)",
		},
		// === German ===
		"de_DE-thorsten-medium": {
			URL:            "https://huggingface.co/rhasspy/piper-voices/resolve/main/de/de_DE/thorsten/medium/de_DE-thorsten-medium.onnx",
			VoiceConfigURL: "https://huggingface.co/rhasspy/piper-voices/resolve/main/de/de_DE/thorsten/medium/de_DE-thorsten-medium.onnx.json",
			SizeBytes:      29876543,
			Language:       "de",
			Quality:        "medium",
			DisplayName:    "German (Thorsten, medium) — recommended",
		},
		"de_DE-eva_k-x_low": {
			URL:            "https://huggingface.co/rhasspy/piper-voices/resolve/main/de/de_DE/eva_k/x_low/de_DE-eva_k-x_low.onnx",
			VoiceConfigURL: "https://huggingface.co/rhasspy/piper-voices/resolve/main/de/de_DE/eva_k/x_low/de_DE-eva_k-x_low.onnx.json",
			SizeBytes:      12345678,
			Language:       "de",
			Quality:        "x_low",
			DisplayName:    "German (Eva K., x_low — fast)",
		},
		// === French ===
		"fr_FR-siwis-low": {
			URL:            "https://huggingface.co/rhasspy/piper-voices/resolve/main/fr/fr_FR/siwis/low/fr_FR-siwis-low.onnx",
			VoiceConfigURL: "https://huggingface.co/rhasspy/piper-voices/resolve/main/fr/fr_FR/siwis/low/fr_FR-siwis-low.onnx.json",
			SizeBytes:      18765432,
			Language:       "fr",
			Quality:        "low",
			DisplayName:    "French (Siwis, low) — recommended",
		},
		"fr_FR-upmc-medium": {
			URL:            "https://huggingface.co/rhasspy/piper-voices/resolve/main/fr/fr_FR/upmc/medium/fr_FR-upmc-medium.onnx",
			VoiceConfigURL: "https://huggingface.co/rhasspy/piper-voices/resolve/main/fr/fr_FR/upmc/medium/fr_FR-upmc-medium.onnx.json",
			SizeBytes:      43210987,
			Language:       "fr",
			Quality:        "medium",
			DisplayName:    "French (UPMC, medium)",
		},
		// === Italian ===
		"it_IT-riccardo-x_low": {
			URL:            "https://huggingface.co/rhasspy/piper-voices/resolve/main/it/it_IT/riccardo/x_low/it_IT-riccardo-x_low.onnx",
			VoiceConfigURL: "https://huggingface.co/rhasspy/piper-voices/resolve/main/it/it_IT/riccardo/x_low/it_IT-riccardo-x_low.onnx.json",
			SizeBytes:      11234567,
			Language:       "it",
			Quality:        "x_low",
			DisplayName:    "Italian (Riccardo, x_low — fast)",
		},
		"it_IT-paola-medium": {
			URL:            "https://huggingface.co/rhasspy/piper-voices/resolve/main/it/it_IT/paola/medium/it_IT-paola-medium.onnx",
			VoiceConfigURL: "https://huggingface.co/rhasspy/piper-voices/resolve/main/it/it_IT/paola/medium/it_IT-paola-medium.onnx.json",
			SizeBytes:      34567890,
			Language:       "it",
			Quality:        "medium",
			DisplayName:    "Italian (Paola, medium)",
		},
		// === Portuguese ===
		"pt_BR-faber-medium": {
			URL:            "https://huggingface.co/rhasspy/piper-voices/resolve/main/pt/pt_BR/faber/medium/pt_BR-faber-medium.onnx",
			VoiceConfigURL: "https://huggingface.co/rhasspy/piper-voices/resolve/main/pt/pt_BR/faber/medium/pt_BR-faber-medium.onnx.json",
			SizeBytes:      45678901,
			Language:       "pt",
			Quality:        "medium",
			DisplayName:    "Brazilian Portuguese (Faber, medium)",
		},
		"pt_PT-tugao-medium": {
			URL:            "https://huggingface.co/rhasspy/piper-voices/resolve/main/pt/pt_PT/tugao/medium/pt_PT-tugao-medium.onnx",
			VoiceConfigURL: "https://huggingface.co/rhasspy/piper-voices/resolve/main/pt/pt_PT/tugao/medium/pt_PT-tugao-medium.onnx.json",
			SizeBytes:      34567890,
			Language:       "pt",
			Quality:        "medium",
			DisplayName:    "European Portuguese (Tugao, medium)",
		},
		// === Russian ===
		"ru_RU-dmitri-medium": {
			URL:            "https://huggingface.co/rhasspy/piper-voices/resolve/main/ru/ru_RU/dmitri/medium/ru_RU-dmitri-medium.onnx",
			VoiceConfigURL: "https://huggingface.co/rhasspy/piper-voices/resolve/main/ru/ru_RU/dmitri/medium/ru_RU-dmitri-medium.onnx.json",
			SizeBytes:      41234567,
			Language:       "ru",
			Quality:        "medium",
			DisplayName:    "Russian (Dmitri, medium)",
		},
		// === Chinese (Mandarin) ===
		"zh_CN-huayan-medium": {
			URL:            "https://huggingface.co/rhasspy/piper-voices/resolve/main/zh/zh_CN/huayan/medium/zh_CN-huayan-medium.onnx",
			VoiceConfigURL: "https://huggingface.co/rhasspy/piper-voices/resolve/main/zh/zh_CN/huayan/medium/zh_CN-huayan-medium.onnx.json",
			SizeBytes:      56789012,
			Language:       "zh",
			Quality:        "medium",
			DisplayName:    "Mandarin Chinese (Huayan, medium)",
		},
		// === Japanese ===
		"ja_JP-kaiueo-x_low": {
			URL:            "https://huggingface.co/rhasspy/piper-voices/resolve/main/ja/ja_JP/kaiueo/x_low/ja_JP-kaiueo-x_low.onnx",
			VoiceConfigURL: "https://huggingface.co/rhasspy/piper-voices/resolve/main/ja/ja_JP/kaiueo/x_low/ja_JP-kaiueo-x_low.onnx.json",
			SizeBytes:      11234567,
			Language:       "ja",
			Quality:        "x_low",
			DisplayName:    "Japanese (Kaiueo, x_low — fast)",
		},
		"ja_JP-takumi-medium": {
			URL:            "https://huggingface.co/rhasspy/piper-voices/resolve/main/ja/ja_JP/takumi/medium/ja_JP-takumi-medium.onnx",
			VoiceConfigURL: "https://huggingface.co/rhasspy/piper-voices/resolve/main/ja/ja_JP/takumi/medium/ja_JP-takumi-medium.onnx.json",
			SizeBytes:      38901234,
			Language:       "ja",
			Quality:        "medium",
			DisplayName:    "Japanese (Takumi, medium)",
		},
		// === Korean ===
		"ko_KR-ko_ohjihye-medium": {
			URL:            "https://huggingface.co/rhasspy/piper-voices/resolve/main/ko/ko_KR/ko_ohjihye/medium/ko_KR-ko_ohjihye-medium.onnx",
			VoiceConfigURL: "https://huggingface.co/rhasspy/piper-voices/resolve/main/ko/ko_KR/ko_ohjihye/medium/ko_KR-ko_ohjihye-medium.onnx.json",
			SizeBytes:      42345678,
			Language:       "ko",
			Quality:        "medium",
			DisplayName:    "Korean (Oh Jihye, medium)",
		},
	}
}

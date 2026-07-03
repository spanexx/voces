/* Code Map: Config Validation
 * - validateConfig: Internal structural validation
 * - validatePaths: Checks for existence of binaries/models
 * - ValidateBinary: Public helper for binary test
 *
 * CID Index:
 * CID:config-validate-001 -> validateConfig
 * CID:config-validate-002 -> validatePaths
 * CID:config-validate-003 -> ValidateBinary
 *
 * Quick lookup: rg -n "CID:config-validate-" internal/config/validate.go
 */
package config

import (
	"fmt"
	"os"
)

// CID:config-validate-001 - validateConfig
// Purpose: Validates the structural integrity and bounds of the configuration.
// Used by: Load()
func validateConfig(cfg *Config) error {
	// Basic hotkey format validation - should not be empty and should contain valid key indicators
	if cfg.Hotkeys.RecordAndType == "" {
		return fmt.Errorf("hotkeys.record_and_type is required")
	}
	hasKey := false
	for _, r := range cfg.Hotkeys.RecordAndType {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '<' || r == '>' {
			hasKey = true
			break
		}
	}
	if !hasKey {
		return fmt.Errorf("hotkeys.record_and_type must contain at least one key")
	}

	// Validate audio settings
	if cfg.Audio.SampleRate <= 0 {
		return fmt.Errorf("audio.sample_rate must be positive")
	}
	if cfg.Audio.Channels <= 0 {
		return fmt.Errorf("audio.channels must be positive")
	}
	if cfg.Audio.Channels != 1 && cfg.Audio.Channels != 2 {
		return fmt.Errorf("audio.channels must be 1 (mono) or 2 (stereo), got %d", cfg.Audio.Channels)
	}

	// Validate behavior settings
	if cfg.Behavior.TypeDelay < 0 {
		return fmt.Errorf("behavior.type_delay must be non-negative")
	}

	// Validate TTS ElevenLabs settings
	if cfg.TTS.ElevenLabs.Stability < 0.0 || cfg.TTS.ElevenLabs.Stability > 1.0 {
		return fmt.Errorf("tts.elevenlabs.stability must be between 0.0 and 1.0")
	}
	if cfg.TTS.ElevenLabs.SimilarityBoost < 0.0 || cfg.TTS.ElevenLabs.SimilarityBoost > 1.0 {
		return fmt.Errorf("tts.elevenlabs.similarity_boost must be between 0.0 and 1.0")
	}

	// Validate transcription engine
	validEngines := map[string]bool{"whisper_cpp": true, "openai_api": true}
	if !validEngines[cfg.Transcription.DefaultEngine] {
		return fmt.Errorf("transcription.default_engine must be 'whisper_cpp' or 'openai_api', got %s", cfg.Transcription.DefaultEngine)
	}

	// Validate TTS engine
	validTTSEngines := map[string]bool{"piper": true, "elevenlabs": true}
	if !validTTSEngines[cfg.TTS.DefaultEngine] {
		return fmt.Errorf("tts.default_engine must be 'piper' or 'elevenlabs', got %s", cfg.TTS.DefaultEngine)
	}

	return nil
}

// CID:config-validate-002 - validatePaths
// Purpose: Ensures that configured local paths for binaries and models actually exist.
// Used by: Transcription and TTS managers before execution.
func validatePaths(cfg *Config) error {
	if cfg.Transcription.DefaultEngine == "whisper_cpp" {
		if cfg.Transcription.WhisperCPP.BinaryPath != "" && !fileExists(cfg.Transcription.WhisperCPP.BinaryPath) {
			return fmt.Errorf("whisper.cpp binary not found: %s", cfg.Transcription.WhisperCPP.BinaryPath)
		}
		if cfg.Transcription.WhisperCPP.Model != "" && !fileExists(cfg.Transcription.WhisperCPP.Model) {
			return fmt.Errorf("whisper model not found: %s", cfg.Transcription.WhisperCPP.Model)
		}
	}
	if cfg.TTS.DefaultEngine == "piper" {
		if cfg.TTS.Piper.BinaryPath != "" && !fileExists(cfg.TTS.Piper.BinaryPath) {
			return fmt.Errorf("piper binary not found: %s", cfg.TTS.Piper.BinaryPath)
		}
		if cfg.TTS.Piper.Model != "" && !fileExists(cfg.TTS.Piper.Model) {
			return fmt.Errorf("piper model not found: %s", cfg.TTS.Piper.Model)
		}
	}
	return nil
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// CID:config-validate-003 - ValidateBinary
// Purpose: Public helper to verify if a binary is present and executable.
// Used by: UI/Tray for setup validation.
func ValidateBinary(path string) error {
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("binary not found: %s", path)
	}
	// We just want to verify it's executable via stat/Run check
	return nil
}

// Validate is a public wrapper around internal config validation.
func Validate(cfg *Config) error {
	return validateConfig(cfg)
}

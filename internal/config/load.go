/* Code Map: Config Loading
 * - Load: Main entry point for reading config
 * - substituteEnvVars: Replaces variable patterns in config strings
 * - substituteEnvVar: Logic for single string substitution
 *
 * CID Index:
 * CID:config-load-001 -> Load
 * CID:config-load-002 -> substituteEnvVars
 *
 * Quick lookup: rg -n "CID:config-load-" internal/config/load.go
 */
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// CID:config-load-001 - Load
// Purpose: Reads application configuration from executable directory or current path.
// Uses: viper, createDefaultConfig, validateConfig, substituteEnvVars
// Used by: internal/app/lifecycle.go
func Load() (*Config, error) {
	// Get executable directory
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)

	// Set up viper
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// 1. User config directory (XDG spec)
	if configDir, err := os.UserConfigDir(); err == nil {
		v.AddConfigPath(filepath.Join(configDir, "voces"))
		// Tests and some dev workflows historically place config.yaml directly in XDG_CONFIG_HOME.
		// Keep that working under go test only.
		if isRunningUnderGoTest() {
			v.AddConfigPath(configDir)
		}
	}

	// 2. Executable directory
	v.AddConfigPath(execDir)

	// Note: we intentionally do not search the current working directory in production.
	// Running the installed binary from a source checkout would otherwise pick up ./config.yaml unexpectedly.
	if isRunningUnderGoTest() {
		v.AddConfigPath(".")
	}

	// Enable environment variable substitution
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found - create default
			if err := createDefaultConfig(""); err != nil {
				return nil, fmt.Errorf("failed to create default config: %w", err)
			}
			// Try reading again
			if err := v.ReadInConfig(); err != nil {
				return nil, fmt.Errorf("failed to read config after creation: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal into struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Substitute environment variables for sensitive fields
	substituteEnvVars(&cfg)

	// Validate required fields
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed (%s): %w", v.ConfigFileUsed(), err)
	}

	return &cfg, nil
}

// CID:config-load-002 - substituteEnvVars
// Purpose: Replaces variable patterns with environment variable values.
// Supports ${VAR_NAME} syntax.
func substituteEnvVars(cfg *Config) {
	// Transcription - OpenAI API Key
	cfg.Transcription.OpenAIAPI.APIKey = substituteEnvVar(cfg.Transcription.OpenAIAPI.APIKey)

	// TTS - ElevenLabs API Key
	cfg.TTS.ElevenLabs.APIKey = substituteEnvVar(cfg.TTS.ElevenLabs.APIKey)
}

// substituteEnvVar substitutes a single environment variable if the value matches ${VAR_NAME} pattern.
func substituteEnvVar(value string) string {
	if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
		envVar := value[2 : len(value)-1]
		if envVal := os.Getenv(envVar); envVal != "" {
			return envVal
		}
	}
	return value
}

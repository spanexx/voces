// Package main provides the entry point for the Whisper Voice Utility.
// This application provides voice transcription and text-to-speech capabilities.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"whisper-voice-util/internal/app"
)

// Version is injected during the build process
var Version = "dev"

// main initializes the application controller and starts the system tray.
func main() {
	versionFlag := flag.Bool("version", false, "Print the application version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("Whisper Voice Utility version %s\n", Version)
		os.Exit(0)
	}

	if configDir, err := os.UserConfigDir(); err == nil {
		logDir := filepath.Join(configDir, "whisper-voice-util", "logs")
		_ = os.MkdirAll(logDir, 0755)
		logPath := filepath.Join(logDir, "whisper-voice-util.log")
		if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			log.SetOutput(io.MultiWriter(os.Stderr, f))
		}
	}

	application, err := app.New()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v\n", err)
	}

	application.Run()
}

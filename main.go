package main

import (
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"smallFileSync/internal/model"
	"smallFileSync/internal/storage"
	"strconv"
	"strings"
)

//go:embed internal/web/static/*
var staticFS embed.FS

func main() {
	args := os.Args[1:]

	if len(args) > 0 && args[0] == "web" {
		port := 8080
		if len(args) > 1 {
			if p, err := strconv.Atoi(args[1]); err == nil && p > 0 && p < 65536 {
				port = p
			}
		}
		startWebMode(port)
		return
	}

	if len(args) > 0 && args[0] == "--import-config" {
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: sfs --import-config <base64>")
			os.Exit(1)
		}
		handleImportConfig(args[1])
		return
	}

	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nUsage: sfs [web [port]]\n", args[0])
		os.Exit(1)
	}

	startTerminalMode()
}


// maskString masks a sensitive string for display.
func maskString(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= 4 {
		return "****"
	}
	return string(runes[:2]) + "****" + string(runes[len(runes)-2:])
}

func handleImportConfig(b64 string) {
	// Decode base64
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to decode config: %v\n", err)
		os.Exit(1)
	}

	// Parse JSON
	var settings model.AppSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse config: %v\n", err)
		os.Exit(1)
	}

	// Check for empty config
	if settings.Storage.WebDAV.Endpoint == "" && settings.Storage.WebDAV.Username == "" && settings.Storage.WebDAV.Password == "" {
		fmt.Fprintln(os.Stderr, "Invalid or empty configuration.")
		os.Exit(1)
	}

	// Print field details with masked sensitive fields
	fmt.Println("Configuration to import:")
	fmt.Printf("  Language: %s\n", settings.Language)
	fmt.Printf("  AutoSync: %v\n", settings.AutoSync)
	fmt.Printf("  Storage Type: %s\n", settings.Storage.Type)
	fmt.Printf("  WebDAV Endpoint: %s\n", settings.Storage.WebDAV.Endpoint)
	fmt.Printf("  WebDAV Username: %s\n", settings.Storage.WebDAV.Username)

	// Mask password for display
	maskedPwd := maskString(settings.Storage.WebDAV.Password)
	fmt.Printf("  WebDAV Password: %s\n", maskedPwd)
	fmt.Printf("  WebDAV BasePath: %s\n", settings.Storage.WebDAV.BasePath)

	// Security warning
	fmt.Println()
	fmt.Println("⚠  Warning: This configuration contains sensitive credentials.")
	fmt.Println("   Do not share it with untrusted parties.")
	fmt.Println()

	// Ask user confirmation
	fmt.Print("Import this configuration? (y/N): ")
	var answer string
	fmt.Scanln(&answer)
	answer = strings.ToLower(strings.TrimSpace(answer))

	if answer != "y" && answer != "yes" {
		fmt.Println("Import cancelled.")
		return
	}

	// Save settings
	store, err := storage.NewLocalStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open local store: %v\n", err)
		os.Exit(1)
	}

	if err := store.SaveSettings(settings); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save settings: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Configuration imported successfully.")
}

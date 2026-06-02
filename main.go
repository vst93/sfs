package main

import (
	"embed"
	"fmt"
	"os"
	"strconv"
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

	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nUsage: sfs [web [port]]\n", args[0])
		os.Exit(1)
	}

	startTerminalMode()
}

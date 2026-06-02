package main

import (
	"fmt"
	"smallFileSync/internal/web"
	"os"
)

func startWebMode(port int) {
	server, err := web.NewServer(staticFS)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create web server: %v\n", err)
		os.Exit(1)
	}

	if err := server.Start(port); err != nil {
		fmt.Fprintf(os.Stderr, "Web server error: %v\n", err)
		os.Exit(1)
	}
}

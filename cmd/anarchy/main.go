package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/popododo0720/anarchy/internal/cli"
)

func main() {
	apiBaseURL := os.Getenv("ANARCHY_API_URL")
	if apiBaseURL == "" {
		apiBaseURL = "http://127.0.0.1:8080"
	}

	if err := cli.Run(os.Args[1:], apiBaseURL, http.DefaultClient, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

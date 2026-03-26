package cli

import (
	"fmt"
	"io"
	"net/http"

	cliimage "github.com/popododo0720/anarchy/internal/cli/image"
	clinode "github.com/popododo0720/anarchy/internal/cli/node"
	clisystem "github.com/popododo0720/anarchy/internal/cli/system"
	clivm "github.com/popododo0720/anarchy/internal/cli/vm"
)

func Run(args []string, apiBaseURL string, httpClient *http.Client, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("missing command")
	}

	switch args[0] {
	case "system":
		return clisystem.Run(args[1:], apiBaseURL, httpClient, out)
	case "node":
		return clinode.Run(args[1:], apiBaseURL, httpClient, out)
	case "image":
		return cliimage.Run(args[1:], apiBaseURL, httpClient, out)
	case "vm":
		return clivm.Run(args[1:], apiBaseURL, httpClient, out)
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

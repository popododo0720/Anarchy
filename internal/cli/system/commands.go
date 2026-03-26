package system

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func Run(args []string, apiBaseURL string, httpClient *http.Client, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("missing system subcommand")
	}

	client := Client{BaseURL: strings.TrimRight(apiBaseURL, "/"), HTTPClient: httpClient}

	switch args[0] {
	case "health":
		return runHealth(client, out)
	case "version":
		return runVersion(client, out)
	case "capabilities":
		return runCapabilities(client, out)
	default:
		return fmt.Errorf("unknown system subcommand: %s", args[0])
	}
}

type healthResponse struct {
	Status              string   `json:"status"`
	KubernetesReachable bool     `json:"kubernetesReachable"`
	KubeVirtReady       bool     `json:"kubevirtReady"`
	CDIReady            bool     `json:"cdiReady"`
	ReadyNodes          int      `json:"readyNodes"`
	TotalNodes          int      `json:"totalNodes"`
	Warnings            []string `json:"warnings"`
}

type versionResponse struct {
	CLIVersion           string   `json:"cliVersion"`
	APIVersion           string   `json:"apiVersion"`
	ServerVersion        string   `json:"serverVersion"`
	SupportedAPIVersions []string `json:"supportedApiVersions"`
	KubernetesVersion    string   `json:"kubernetesVersion"`
	KubeVirtVersion      string   `json:"kubevirtVersion"`
}

type capabilitiesResponse struct {
	VMLifecycleSupported    bool     `json:"vmLifecycleSupported"`
	ImageInventorySupported bool     `json:"imageInventorySupported"`
	DiagnosticsSupported    bool     `json:"diagnosticsSupported"`
	PublicIPSupported       bool     `json:"publicIpSupported"`
	Capabilities            []string `json:"capabilities"`
}

func runHealth(client Client, out io.Writer) error {
	var resp healthResponse
	if err := client.getJSON("/api/v1/system/health", &resp); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out,
		"Status: %s\nKubernetes reachable: %t\nKubeVirt ready: %t\nCDI ready: %t\nReady nodes: %d/%d\n",
		resp.Status, resp.KubernetesReachable, resp.KubeVirtReady, resp.CDIReady, resp.ReadyNodes, resp.TotalNodes,
	)
	return err
}

func runVersion(client Client, out io.Writer) error {
	var resp versionResponse
	if err := client.getJSON("/api/v1/system/version", &resp); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out,
		"CLI version: %s\nAPI version: %s\nServer version: %s\n",
		valueOrUnknown(resp.CLIVersion), valueOrUnknown(resp.APIVersion), valueOrUnknown(resp.ServerVersion),
	)
	return err
}

func runCapabilities(client Client, out io.Writer) error {
	var resp capabilitiesResponse
	if err := client.getJSON("/api/v1/system/capabilities", &resp); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out,
		"VM lifecycle supported: %t\nImage inventory supported: %t\nDiagnostics supported: %t\nPublic IP supported: %t\n",
		resp.VMLifecycleSupported, resp.ImageInventorySupported, resp.DiagnosticsSupported, resp.PublicIPSupported,
	)
	return err
}

func (c Client) getJSON(path string, target any) error {
	resp, err := c.HTTPClient.Get(c.BaseURL + path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("api error: %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func valueOrUnknown(v string) string {
	if v == "" {
		return "unknown"
	}
	return v
}

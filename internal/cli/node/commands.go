package node

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

type nodeSummary struct {
	Name                  string `json:"name"`
	Class                 string `json:"class"`
	Ready                 bool   `json:"ready"`
	Schedulable           bool   `json:"schedulable"`
	VirtualizationCapable bool   `json:"virtualizationCapable"`
}

type nodeDetail struct {
	Name                  string   `json:"name"`
	Class                 string   `json:"class"`
	Ready                 bool     `json:"ready"`
	Schedulable           bool     `json:"schedulable"`
	VirtualizationCapable bool     `json:"virtualizationCapable"`
	Capabilities          []string `json:"capabilities"`
}

func Run(args []string, apiBaseURL string, httpClient *http.Client, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("missing node subcommand")
	}
	client := Client{BaseURL: strings.TrimRight(apiBaseURL, "/"), HTTPClient: httpClient}
	switch args[0] {
	case "list":
		return runList(client, out)
	case "show":
		if len(args) < 2 {
			return fmt.Errorf("missing node name")
		}
		return runShow(client, args[1], out)
	default:
		return fmt.Errorf("unknown node subcommand: %s", args[0])
	}
}

func runList(client Client, out io.Writer) error {
	var nodes []nodeSummary
	if err := client.getJSON("/api/v1/nodes", &nodes); err != nil {
		return err
	}
	for _, n := range nodes {
		if _, err := fmt.Fprintf(out, "Name: %s\nClass: %s\nReady: %t\nSchedulable: %t\nVirtualization capable: %t\n\n", n.Name, n.Class, n.Ready, n.Schedulable, n.VirtualizationCapable); err != nil {
			return err
		}
	}
	return nil
}

func runShow(client Client, name string, out io.Writer) error {
	var node nodeDetail
	if err := client.getJSON("/api/v1/nodes/"+name, &node); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "Name: %s\nClass: %s\nReady: %t\nSchedulable: %t\nVirtualization capable: %t\nCapabilities: %s\n", node.Name, node.Class, node.Ready, node.Schedulable, node.VirtualizationCapable, strings.Join(node.Capabilities, ", "))
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

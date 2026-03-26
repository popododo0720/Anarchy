package network

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

type networkSummary struct {
	Name          string `json:"name"`
	Default       bool   `json:"default"`
	DefaultSubnet string `json:"defaultSubnet"`
}

type networkDetail struct {
	Name          string   `json:"name"`
	Default       bool     `json:"default"`
	Router        string   `json:"router"`
	DefaultSubnet string   `json:"defaultSubnet"`
	Subnets       []string `json:"subnets"`
}

func Run(args []string, apiBaseURL string, httpClient *http.Client, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("missing network subcommand")
	}
	client := Client{BaseURL: strings.TrimRight(apiBaseURL, "/"), HTTPClient: httpClient}
	switch args[0] {
	case "list":
		return runList(client, out)
	case "show":
		if len(args) < 2 {
			return fmt.Errorf("missing network name")
		}
		return runShow(client, args[1], out)
	default:
		return fmt.Errorf("unknown network subcommand: %s", args[0])
	}
}

func runList(client Client, out io.Writer) error {
	var networks []networkSummary
	if err := client.getJSON("/api/v1/networks", &networks); err != nil {
		return err
	}
	for _, network := range networks {
		if _, err := fmt.Fprintf(out, "Name: %s\nDefault: %t\nDefault subnet: %s\n\n", network.Name, network.Default, network.DefaultSubnet); err != nil {
			return err
		}
	}
	return nil
}

func runShow(client Client, name string, out io.Writer) error {
	var network networkDetail
	if err := client.getJSON("/api/v1/networks/"+name, &network); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "Name: %s\nDefault: %t\nRouter: %s\nDefault subnet: %s\nSubnets: %s\n", network.Name, network.Default, network.Router, network.DefaultSubnet, strings.Join(network.Subnets, ", "))
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

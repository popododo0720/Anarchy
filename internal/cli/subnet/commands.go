package subnet

import (
	"bytes"
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

type subnetSummary struct {
	Name     string `json:"name"`
	CIDR     string `json:"cidr"`
	Gateway  string `json:"gateway"`
	Protocol string `json:"protocol"`
	Network  string `json:"network"`
}

type subnetDetail struct {
	Name       string   `json:"name"`
	CIDR       string   `json:"cidr"`
	Gateway    string   `json:"gateway"`
	Protocol   string   `json:"protocol"`
	Provider   string   `json:"provider"`
	VLAN       string   `json:"vlan"`
	Network    string   `json:"network"`
	Namespaces []string `json:"namespaces"`
}

type createSubnetRequest struct {
	Name       string   `json:"name"`
	CIDR       string   `json:"cidr"`
	Gateway    string   `json:"gateway"`
	Protocol   string   `json:"protocol"`
	Provider   string   `json:"provider"`
	Network    string   `json:"network"`
	Namespaces []string `json:"namespaces,omitempty"`
}

func Run(args []string, apiBaseURL string, httpClient *http.Client, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("missing subnet subcommand")
	}
	client := Client{BaseURL: strings.TrimRight(apiBaseURL, "/"), HTTPClient: httpClient}
	switch args[0] {
	case "list":
		return runList(client, out)
	case "show":
		if len(args) < 2 {
			return fmt.Errorf("missing subnet name")
		}
		return runShow(client, args[1], out)
	case "create":
		if len(args) < 7 {
			return fmt.Errorf("usage: subnet create <name> <cidr> <gateway> <protocol> <provider> <network> [namespace...]")
		}
		req := createSubnetRequest{Name: args[1], CIDR: args[2], Gateway: args[3], Protocol: args[4], Provider: args[5], Network: args[6]}
		if len(args) > 7 {
			req.Namespaces = args[7:]
		}
		return runCreate(client, req, out)
	default:
		return fmt.Errorf("unknown subnet subcommand: %s", args[0])
	}
}

func runList(client Client, out io.Writer) error {
	var subnets []subnetSummary
	if err := client.getJSON("/api/v1/subnets", &subnets); err != nil {
		return err
	}
	for _, subnet := range subnets {
		if _, err := fmt.Fprintf(out, "Name: %s\nCIDR: %s\nGateway: %s\nProtocol: %s\nNetwork: %s\n\n", subnet.Name, subnet.CIDR, subnet.Gateway, subnet.Protocol, subnet.Network); err != nil {
			return err
		}
	}
	return nil
}

func runShow(client Client, name string, out io.Writer) error {
	var subnet subnetDetail
	if err := client.getJSON("/api/v1/subnets/"+name, &subnet); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "Name: %s\nCIDR: %s\nGateway: %s\nProtocol: %s\nProvider: %s\nVLAN: %s\nNetwork: %s\nNamespaces: %s\n", subnet.Name, subnet.CIDR, subnet.Gateway, subnet.Protocol, valueOrUnknown(subnet.Provider), valueOrUnknown(subnet.VLAN), subnet.Network, strings.Join(subnet.Namespaces, ", "))
	return err
}

func runCreate(client Client, req createSubnetRequest, out io.Writer) error {
	var subnet subnetDetail
	if err := client.postJSON("/api/v1/subnets", req, &subnet); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "Created subnet: %s\nCIDR: %s\n", subnet.Name, subnet.CIDR)
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

func (c Client) postJSON(path string, body any, target any) error {
	data, _ := json.Marshal(body)
	resp, err := c.HTTPClient.Post(c.BaseURL+path, "application/json", bytes.NewReader(data))
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

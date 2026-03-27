package diagnose

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

type clusterReport struct {
	Status   string   `json:"status"`
	Findings []string `json:"findings"`
}

type vmReport struct {
	Name     string   `json:"name"`
	Phase    string   `json:"phase"`
	Findings []string `json:"findings"`
}

type publicIPReport struct {
	Name     string   `json:"name"`
	Status   string   `json:"status"`
	Findings []string `json:"findings"`
}

func Run(args []string, apiBaseURL string, httpClient *http.Client, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("missing diagnose subcommand")
	}
	client := Client{BaseURL: strings.TrimRight(apiBaseURL, "/"), HTTPClient: httpClient}
	switch args[0] {
	case "cluster":
		return runCluster(client, out)
	case "vm":
		if len(args) < 2 {
			return fmt.Errorf("missing vm name")
		}
		return runVM(client, args[1], out)
	case "publicip":
		if len(args) < 2 {
			return fmt.Errorf("missing public ip name")
		}
		return runPublicIP(client, args[1], out)
	default:
		return fmt.Errorf("unknown diagnose subcommand: %s", args[0])
	}
}

func runCluster(client Client, out io.Writer) error {
	var resp clusterReport
	if err := client.getJSON("/api/v1/diagnose/cluster", &resp); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Status: %s\n", resp.Status); err != nil {
		return err
	}
	for _, finding := range resp.Findings {
		if _, err := fmt.Fprintf(out, "- %s\n", finding); err != nil {
			return err
		}
	}
	return nil
}

func runVM(client Client, name string, out io.Writer) error {
	var resp vmReport
	if err := client.getJSON("/api/v1/diagnose/vms/"+name, &resp); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Name: %s\nPhase: %s\n", resp.Name, resp.Phase); err != nil {
		return err
	}
	for _, finding := range resp.Findings {
		if _, err := fmt.Fprintf(out, "- %s\n", finding); err != nil {
			return err
		}
	}
	return nil
}

func runPublicIP(client Client, name string, out io.Writer) error {
	var resp publicIPReport
	if err := client.getJSON("/api/v1/diagnose/public-ips/"+name, &resp); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Name: %s\nStatus: %s\n", resp.Name, resp.Status); err != nil {
		return err
	}
	for _, finding := range resp.Findings {
		if _, err := fmt.Fprintf(out, "- %s\n", finding); err != nil {
			return err
		}
	}
	return nil
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

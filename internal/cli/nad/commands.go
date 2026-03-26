package nad

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

type nadSummary struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Type      string `json:"type"`
	Provider  string `json:"provider"`
}

type nadDetail struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Type      string `json:"type"`
	Provider  string `json:"provider"`
	RawConfig string `json:"rawConfig"`
}

type createNADRequest struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Type         string `json:"type"`
	Provider     string `json:"provider"`
	ServerSocket string `json:"serverSocket"`
}

func Run(args []string, apiBaseURL string, httpClient *http.Client, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("missing nad subcommand")
	}
	client := Client{BaseURL: strings.TrimRight(apiBaseURL, "/"), HTTPClient: httpClient}
	switch args[0] {
	case "list":
		return runList(client, out)
	case "show":
		if len(args) < 3 {
			return fmt.Errorf("usage: nad show <namespace> <name>")
		}
		return runShow(client, args[1], args[2], out)
	case "create":
		if len(args) < 6 {
			return fmt.Errorf("usage: nad create <name> <namespace> <type> <provider> <serverSocket>")
		}
		return runCreate(client, createNADRequest{Name: args[1], Namespace: args[2], Type: args[3], Provider: args[4], ServerSocket: args[5]}, out)
	default:
		return fmt.Errorf("unknown nad subcommand: %s", args[0])
	}
}

func runList(client Client, out io.Writer) error {
	var nads []nadSummary
	if err := client.getJSON("/api/v1/nads", &nads); err != nil {
		return err
	}
	for _, nad := range nads {
		if _, err := fmt.Fprintf(out, "Name: %s\nNamespace: %s\nType: %s\nProvider: %s\n\n", nad.Name, nad.Namespace, nad.Type, nad.Provider); err != nil {
			return err
		}
	}
	return nil
}

func runShow(client Client, namespace, name string, out io.Writer) error {
	var nad nadDetail
	if err := client.getJSON("/api/v1/nads/"+namespace+"/"+name, &nad); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "Name: %s\nNamespace: %s\nType: %s\nProvider: %s\nRaw config: %s\n", nad.Name, nad.Namespace, nad.Type, nad.Provider, nad.RawConfig)
	return err
}

func runCreate(client Client, req createNADRequest, out io.Writer) error {
	var nad nadDetail
	if err := client.postJSON("/api/v1/nads", req, &nad); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "Created NAD: %s\nNamespace: %s\n", nad.Name, nad.Namespace)
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

package publicip

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

type publicIPSummary struct {
	Name             string `json:"name"`
	Address          string `json:"address"`
	Attached         bool   `json:"attached"`
	AttachmentTarget string `json:"attachmentTarget"`
}

type publicIPDetail struct {
	Name             string `json:"name"`
	Address          string `json:"address"`
	Attached         bool   `json:"attached"`
	AttachmentTarget string `json:"attachmentTarget"`
	Type             string `json:"type"`
}

type attachPublicIPRequest struct {
	Name             string `json:"name"`
	AttachmentTarget string `json:"attachmentTarget"`
}

func Run(args []string, apiBaseURL string, httpClient *http.Client, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("missing publicip subcommand")
	}
	client := Client{BaseURL: strings.TrimRight(apiBaseURL, "/"), HTTPClient: httpClient}
	switch args[0] {
	case "list":
		return runList(client, out)
	case "show":
		if len(args) < 2 {
			return fmt.Errorf("usage: publicip show <name>")
		}
		return runShow(client, args[1], out)
	case "attach":
		if len(args) < 3 {
			return fmt.Errorf("usage: publicip attach <name> <vm:nic>")
		}
		return runAttach(client, attachPublicIPRequest{Name: args[1], AttachmentTarget: args[2]}, out)
	case "detach":
		if len(args) < 2 {
			return fmt.Errorf("usage: publicip detach <name>")
		}
		return runDetach(client, args[1], out)
	default:
		return fmt.Errorf("unknown publicip subcommand: %s", args[0])
	}
}

func runList(client Client, out io.Writer) error {
	var items []publicIPSummary
	if err := client.getJSON("/api/v1/public-ips", &items); err != nil {
		return err
	}
	for _, item := range items {
		if _, err := fmt.Fprintf(out, "Name: %s\nAddress: %s\nAttached: %t\nTarget: %s\n\n", item.Name, item.Address, item.Attached, item.AttachmentTarget); err != nil {
			return err
		}
	}
	return nil
}

func runShow(client Client, name string, out io.Writer) error {
	var item publicIPDetail
	if err := client.getJSON("/api/v1/public-ips/"+name, &item); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "Name: %s\nAddress: %s\nType: %s\nAttached: %t\nTarget: %s\n", item.Name, item.Address, item.Type, item.Attached, item.AttachmentTarget)
	return err
}

func runAttach(client Client, req attachPublicIPRequest, out io.Writer) error {
	var item publicIPDetail
	if err := client.postJSON("/api/v1/public-ips/"+req.Name+"/attach", req, &item); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "Attached public IP: %s -> %s\n", item.Name, item.AttachmentTarget)
	return err
}

func runDetach(client Client, name string, out io.Writer) error {
	var item publicIPDetail
	if err := client.postJSON("/api/v1/public-ips/"+name+"/detach", map[string]any{}, &item); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "Detached public IP: %s\n", item.Name)
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

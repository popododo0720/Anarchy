package image

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

type imageSummary struct {
	Name       string `json:"name"`
	SourceType string `json:"sourceType"`
	Ready      bool   `json:"ready"`
	Size       string `json:"size"`
}

type imageDetail struct {
	Name        string   `json:"name"`
	SourceType  string   `json:"sourceType"`
	Ready       bool     `json:"ready"`
	Size        string   `json:"size"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

func Run(args []string, apiBaseURL string, httpClient *http.Client, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("missing image subcommand")
	}
	client := Client{BaseURL: strings.TrimRight(apiBaseURL, "/"), HTTPClient: httpClient}
	switch args[0] {
	case "list":
		return runList(client, out)
	case "show":
		if len(args) < 2 {
			return fmt.Errorf("missing image name")
		}
		return runShow(client, args[1], out)
	default:
		return fmt.Errorf("unknown image subcommand: %s", args[0])
	}
}

func runList(client Client, out io.Writer) error {
	var images []imageSummary
	if err := client.getJSON("/api/v1/images", &images); err != nil {
		return err
	}
	for _, img := range images {
		if _, err := fmt.Fprintf(out, "Name: %s\nSource type: %s\nReady: %t\nSize: %s\n\n", img.Name, img.SourceType, img.Ready, img.Size); err != nil {
			return err
		}
	}
	return nil
}

func runShow(client Client, name string, out io.Writer) error {
	var img imageDetail
	if err := client.getJSON("/api/v1/images/"+name, &img); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "Name: %s\nSource type: %s\nReady: %t\nSize: %s\nDescription: %s\nTags: %s\n", img.Name, img.SourceType, img.Ready, img.Size, img.Description, strings.Join(img.Tags, ", "))
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

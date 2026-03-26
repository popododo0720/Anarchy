package vm

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

type vmSummary struct {
	Name               string              `json:"name"`
	Phase              string              `json:"phase"`
	Image              string              `json:"image"`
	Network            string              `json:"network,omitempty"`
	SubnetRef          string              `json:"subnetRef,omitempty"`
	PrivateIP          string              `json:"privateIp"`
	NetworkAttachments []networkAttachment `json:"networkAttachments,omitempty"`
}
type vmDetail struct {
	Name               string              `json:"name"`
	Phase              string              `json:"phase"`
	Image              string              `json:"image"`
	CPU                int                 `json:"cpu"`
	Memory             string              `json:"memory"`
	Network            string              `json:"network"`
	SubnetRef          string              `json:"subnetRef,omitempty"`
	PrivateIP          string              `json:"privateIp"`
	NetworkAttachments []networkAttachment `json:"networkAttachments,omitempty"`
}

type networkAttachment struct {
	Name      string `json:"name"`
	Network   string `json:"network"`
	SubnetRef string `json:"subnetRef,omitempty"`
	Primary   bool   `json:"primary"`
}

type createVMRequest struct {
	Name               string              `json:"name"`
	Image              string              `json:"image"`
	CPU                int                 `json:"cpu"`
	Memory             string              `json:"memory"`
	Network            string              `json:"network"`
	SubnetRef          string              `json:"subnetRef,omitempty"`
	NetworkAttachments []networkAttachment `json:"networkAttachments,omitempty"`
}

func Run(args []string, apiBaseURL string, httpClient *http.Client, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("missing vm subcommand")
	}
	client := Client{BaseURL: strings.TrimRight(apiBaseURL, "/"), HTTPClient: httpClient}
	switch args[0] {
	case "create":
		if len(args) < 6 {
			return fmt.Errorf("usage: vm create <name> <image> <cpu> <memory> <network> [subnetRef]")
		}
		cpu := 0
		fmt.Sscanf(args[3], "%d", &cpu)
		req := createVMRequest{Name: args[1], Image: args[2], CPU: cpu, Memory: args[4], Network: args[5]}
		remaining := args[6:]
		if len(remaining) > 0 && remaining[0] != "--attachments-json" {
			req.SubnetRef = remaining[0]
			remaining = remaining[1:]
		}
		if len(remaining) >= 2 && remaining[0] == "--attachments-json" {
			if err := json.Unmarshal([]byte(remaining[1]), &req.NetworkAttachments); err != nil {
				return fmt.Errorf("invalid attachments json: %w", err)
			}
		}
		if len(req.NetworkAttachments) == 0 {
			req.NetworkAttachments = []networkAttachment{{Name: "nic0", Network: req.Network, SubnetRef: req.SubnetRef, Primary: true}}
		}
		if req.SubnetRef == "" && len(req.NetworkAttachments) > 0 {
			req.SubnetRef = req.NetworkAttachments[0].SubnetRef
		}
		return runCreate(client, req, out)
	case "list":
		return runList(client, out)
	case "show":
		if len(args) < 2 {
			return fmt.Errorf("missing vm name")
		}
		return runShow(client, args[1], out)
	case "start", "stop", "restart", "delete":
		if len(args) < 2 {
			return fmt.Errorf("missing vm name")
		}
		return runAction(client, args[0], args[1], out)
	default:
		return fmt.Errorf("unknown vm subcommand: %s", args[0])
	}
}

func runCreate(client Client, req createVMRequest, out io.Writer) error {
	var detail vmDetail
	if err := client.postJSON("/api/v1/vms", req, &detail); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "Created VM: %s\nPhase: %s\n", detail.Name, detail.Phase)
	return err
}
func runList(client Client, out io.Writer) error {
	var vms []vmSummary
	if err := client.getJSON("/api/v1/vms", &vms); err != nil {
		return err
	}
	for _, vm := range vms {
		if _, err := fmt.Fprintf(out, "Name: %s\nPhase: %s\nImage: %s\nNetwork: %s\nSubnet: %s\nPrivate IP: %s\n\n", vm.Name, vm.Phase, vm.Image, valueOrUnknown(vm.Network), valueOrUnknown(vm.SubnetRef), vm.PrivateIP); err != nil {
			return err
		}
	}
	return nil
}
func runShow(client Client, name string, out io.Writer) error {
	var vm vmDetail
	if err := client.getJSON("/api/v1/vms/"+name, &vm); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "Name: %s\nPhase: %s\nImage: %s\nCPU: %d\nMemory: %s\nNetwork: %s\nSubnet: %s\nPrivate IP: %s\nAttachments: %s\n", vm.Name, vm.Phase, vm.Image, vm.CPU, vm.Memory, vm.Network, valueOrUnknown(vm.SubnetRef), vm.PrivateIP, formatAttachments(vm.NetworkAttachments))
	return err
}
func runAction(client Client, action, name string, out io.Writer) error {
	if err := client.postNoBody("/api/v1/vms/" + name + "/" + action); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "Action accepted: %s %s\n", action, name)
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
func (c Client) postNoBody(path string) error {
	req, _ := http.NewRequest(http.MethodPost, c.BaseURL+path, nil)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("api error: %s", resp.Status)
	}
	return nil
}

func valueOrUnknown(v string) string {
	if v == "" {
		return "unknown"
	}
	return v
}

func formatAttachments(items []networkAttachment) string {
	if len(items) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%s(%s/%s, primary=%t)", item.Name, valueOrUnknown(item.Network), valueOrUnknown(item.SubnetRef), item.Primary))
	}
	return strings.Join(parts, ", ")
}

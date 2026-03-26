package kubernetes

import (
	"context"
	"encoding/json"

	kexec "github.com/popododo0720/anarchy/internal/adapters/kubernetes/exec"
	domainnetwork "github.com/popododo0720/anarchy/internal/domain/network"
)

type Provider struct {
	runner kexec.Runner
}

func NewProvider(runner kexec.Runner) Provider {
	return Provider{runner: runner}
}

type vpcListResponse struct {
	Items []vpcItem `json:"items"`
}

type vpcItem struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Status struct {
		Default              bool     `json:"default"`
		DefaultLogicalSwitch string   `json:"defaultLogicalSwitch"`
		Subnets              []string `json:"subnets"`
		Router               string   `json:"router"`
	} `json:"status"`
}

func (p Provider) ListNetworks(ctx context.Context) ([]domainnetwork.NetworkSummary, error) {
	out, err := p.runner.Run(ctx, "kubectl", "get", "vpcs.kubeovn.io", "-o", "json")
	if err != nil {
		return nil, err
	}
	var payload vpcListResponse
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		return nil, err
	}
	items := make([]domainnetwork.NetworkSummary, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, domainnetwork.NetworkSummary{
			Name:          item.Metadata.Name,
			Default:       item.Status.Default,
			DefaultSubnet: item.Status.DefaultLogicalSwitch,
		})
	}
	return items, nil
}

func (p Provider) GetNetwork(ctx context.Context, name string) (domainnetwork.NetworkDetail, error) {
	out, err := p.runner.Run(ctx, "kubectl", "get", "vpc.kubeovn.io", name, "-o", "json")
	if err != nil {
		return domainnetwork.NetworkDetail{}, err
	}
	var item vpcItem
	if err := json.Unmarshal([]byte(out), &item); err != nil {
		return domainnetwork.NetworkDetail{}, err
	}
	return domainnetwork.NetworkDetail{
		Name:          item.Metadata.Name,
		Default:       item.Status.Default,
		Router:        item.Status.Router,
		DefaultSubnet: item.Status.DefaultLogicalSwitch,
		Subnets:       item.Status.Subnets,
	}, nil
}

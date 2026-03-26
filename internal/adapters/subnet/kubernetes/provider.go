package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	kexec "github.com/popododo0720/anarchy/internal/adapters/kubernetes/exec"
	domainsubnet "github.com/popododo0720/anarchy/internal/domain/subnet"
)

type Provider struct {
	runner kexec.Runner
}

func NewProvider(runner kexec.Runner) Provider {
	return Provider{runner: runner}
}

type subnetListResponse struct {
	Items []subnetItem `json:"items"`
}

type subnetItem struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec struct {
		CIDRBlock  string   `json:"cidrBlock"`
		Gateway    string   `json:"gateway"`
		Protocol   string   `json:"protocol"`
		Provider   string   `json:"provider"`
		VLAN       string   `json:"vlan"`
		VPC        string   `json:"vpc"`
		Namespaces []string `json:"namespaces"`
	} `json:"spec"`
}

func (p Provider) ListSubnets(ctx context.Context) ([]domainsubnet.SubnetSummary, error) {
	out, err := p.runner.Run(ctx, "kubectl", "get", "subnets.kubeovn.io", "-o", "json")
	if err != nil {
		return nil, err
	}
	var payload subnetListResponse
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		return nil, err
	}
	items := make([]domainsubnet.SubnetSummary, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, domainsubnet.SubnetSummary{
			Name:     item.Metadata.Name,
			CIDR:     item.Spec.CIDRBlock,
			Gateway:  item.Spec.Gateway,
			Protocol: item.Spec.Protocol,
			Network:  item.Spec.VPC,
		})
	}
	return items, nil
}

func (p Provider) GetSubnet(ctx context.Context, name string) (domainsubnet.SubnetDetail, error) {
	out, err := p.runner.Run(ctx, "kubectl", "get", "subnet.kubeovn.io", name, "-o", "json")
	if err != nil {
		return domainsubnet.SubnetDetail{}, err
	}
	var item subnetItem
	if err := json.Unmarshal([]byte(out), &item); err != nil {
		return domainsubnet.SubnetDetail{}, err
	}
	return domainsubnet.SubnetDetail{
		Name:       item.Metadata.Name,
		CIDR:       item.Spec.CIDRBlock,
		Gateway:    item.Spec.Gateway,
		Protocol:   item.Spec.Protocol,
		Provider:   item.Spec.Provider,
		VLAN:       item.Spec.VLAN,
		Network:    item.Spec.VPC,
		Namespaces: item.Spec.Namespaces,
	}, nil
}

func (p Provider) CreateSubnet(ctx context.Context, req domainsubnet.CreateSubnetRequest) (domainsubnet.SubnetDetail, error) {
	manifest, err := p.writeManifest(req)
	if err != nil {
		return domainsubnet.SubnetDetail{}, err
	}
	defer os.Remove(manifest)
	if _, err := p.runner.Run(ctx, "kubectl", "apply", "-f", manifest); err != nil {
		return domainsubnet.SubnetDetail{}, err
	}
	return p.GetSubnet(ctx, req.Name)
}

func (p Provider) writeManifest(req domainsubnet.CreateSubnetRequest) (string, error) {
	file, err := os.CreateTemp("", "anarchy-subnet-*.yaml")
	if err != nil {
		return "", err
	}
	namespacesYAML := ""
	if len(req.Namespaces) > 0 {
		namespacesYAML = "  namespaces:\n"
		for _, ns := range req.Namespaces {
			namespacesYAML += fmt.Sprintf("    - %s\n", ns)
		}
	}
	manifest := fmt.Sprintf(`apiVersion: kubeovn.io/v1
kind: Subnet
metadata:
  name: %s
spec:
  cidrBlock: %s
  gateway: %s
  protocol: %s
  provider: %s
  vpc: %s
  natOutgoing: true
  private: false
  default: false
  gatewayType: distributed
%s`, req.Name, req.CIDR, req.Gateway, req.Protocol, req.Provider, req.Network, namespacesYAML)
	if !strings.Contains(manifest, "excludeIps:") {
		manifest += fmt.Sprintf("  excludeIps:\n    - %s\n", req.Gateway)
	}
	if _, err := file.WriteString(manifest); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return file.Name(), nil
}

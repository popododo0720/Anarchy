package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"

	kexec "github.com/popododo0720/anarchy/internal/adapters/kubernetes/exec"
	domainnode "github.com/popododo0720/anarchy/internal/domain/node"
)

type Provider struct{ runner kexec.Runner }

func NewProvider(runner kexec.Runner) Provider { return Provider{runner: runner} }

type nodesResponse struct {
	Items []struct {
		Metadata struct {
			Name   string            `json:"name"`
			Labels map[string]string `json:"labels"`
		} `json:"metadata"`
		Spec struct {
			Unschedulable bool `json:"unschedulable"`
		} `json:"spec"`
		Status struct {
			Conditions []struct {
				Type   string `json:"type"`
				Status string `json:"status"`
			} `json:"conditions"`
		} `json:"status"`
	} `json:"items"`
}

func (p Provider) ListNodes(ctx context.Context) ([]domainnode.NodeSummary, error) {
	out, err := p.runner.Run(ctx, "kubectl", "get", "nodes", "-o", "json")
	if err != nil {
		return nil, err
	}
	var resp nodesResponse
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		return nil, err
	}
	items := make([]domainnode.NodeSummary, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, summarize(item))
	}
	return items, nil
}

func (p Provider) GetNode(ctx context.Context, name string) (domainnode.NodeDetail, error) {
	nodes, err := p.ListNodes(ctx)
	if err != nil {
		return domainnode.NodeDetail{}, err
	}
	for _, node := range nodes {
		if node.Name == name {
			caps := []string{}
			if node.VirtualizationCapable {
				caps = append(caps, "kubevirt")
			}
			caps = append(caps, "kube-ovn")
			return domainnode.NodeDetail{Name: node.Name, Class: node.Class, Ready: node.Ready, Schedulable: node.Schedulable, VirtualizationCapable: node.VirtualizationCapable, Capabilities: caps}, nil
		}
	}
	return domainnode.NodeDetail{}, fmt.Errorf("node not found: %s", name)
}

func summarize(item struct {
	Metadata struct {
		Name   string            `json:"name"`
		Labels map[string]string `json:"labels"`
	} `json:"metadata"`
	Spec struct {
		Unschedulable bool `json:"unschedulable"`
	} `json:"spec"`
	Status struct {
		Conditions []struct {
			Type   string `json:"type"`
			Status string `json:"status"`
		} `json:"conditions"`
	} `json:"status"`
}) domainnode.NodeSummary {
	ready := false
	for _, c := range item.Status.Conditions {
		if c.Type == "Ready" && c.Status == "True" {
			ready = true
			break
		}
	}
	class := "worker"
	if _, ok := item.Metadata.Labels["node-role.kubernetes.io/control-plane"]; ok {
		class = "control-plane"
	}
	virtCap := item.Metadata.Labels["kubevirt.io/schedulable"] == "true"
	return domainnode.NodeSummary{Name: item.Metadata.Name, Class: class, Ready: ready, Schedulable: !item.Spec.Unschedulable, VirtualizationCapable: virtCap}
}

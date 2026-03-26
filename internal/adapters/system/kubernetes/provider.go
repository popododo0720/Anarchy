package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"

	kexec "github.com/popododo0720/anarchy/internal/adapters/kubernetes/exec"
	domainsystem "github.com/popododo0720/anarchy/internal/domain/system"
)

type Provider struct{ runner kexec.Runner }

func NewProvider(runner kexec.Runner) Provider { return Provider{runner: runner} }

func (p Provider) GetHealth(ctx context.Context) (domainsystem.HealthSummary, error) {
	nodes, nodesErr := p.nodeStatus(ctx)
	kubevirtInstalled, kubevirtReady, _ := p.kubeVirtStatus(ctx)
	cdiInstalled, cdiReady := p.cdiStatus(ctx)
	apiReachable := nodesErr == nil
	kubeReachable := nodesErr == nil

	total, ready := 0, 0
	if nodesErr == nil {
		total, ready = nodes.total, nodes.ready
	}
	status := domainsystem.StatusHealthy
	warnings := []string{}
	if !apiReachable || !kubeReachable || !kubevirtInstalled || !kubevirtReady || !cdiInstalled || !cdiReady {
		status = domainsystem.StatusDegraded
	}
	if nodesErr != nil {
		warnings = append(warnings, fmt.Sprintf("nodes unavailable: %v", nodesErr))
	}
	if !kubevirtInstalled {
		warnings = append(warnings, "kubevirt not installed")
	}
	if kubevirtInstalled && !kubevirtReady {
		warnings = append(warnings, "kubevirt not ready")
	}
	if !cdiInstalled {
		warnings = append(warnings, "cdi not installed")
	}
	if cdiInstalled && !cdiReady {
		warnings = append(warnings, "cdi not ready")
	}

	return domainsystem.HealthSummary{
		Status: status, APIReachable: apiReachable, KubernetesReachable: kubeReachable,
		KubeVirtInstalled: kubevirtInstalled, KubeVirtReady: kubevirtReady,
		CDIInstalled: cdiInstalled, CDIReady: cdiReady, TotalNodes: total, ReadyNodes: ready, Warnings: warnings,
	}, nil
}

func (p Provider) GetVersion(ctx context.Context) (domainsystem.VersionSummary, error) {
	kubeVersion, _ := p.kubernetesVersion(ctx)
	kubeVirtVersion, _, _ := p.kubeVirtVersionAndReady(ctx)
	return domainsystem.VersionSummary{CLIVersion: "dev", APIVersion: "v1", ServerVersion: "dev", SupportedAPIVersions: []string{"v1"}, KubernetesVersion: kubeVersion, KubeVirtVersion: kubeVirtVersion}, nil
}

func (p Provider) GetCapabilities(context.Context) (domainsystem.CapabilitiesSummary, error) {
	return domainsystem.CapabilitiesSummary{VMLifecycleSupported: true, ImageInventorySupported: true, DiagnosticsSupported: true, PublicIPSupported: false, Capabilities: []string{"vm-lifecycle", "image-inventory", "diagnostics"}}, nil
}

type nodesResponse struct {
	Items []struct {
		Status struct {
			Conditions []struct{ Type, Status string } `json:"conditions"`
		} `json:"status"`
	} `json:"items"`
}
type kubevirtResponse struct {
	Items []struct {
		Status struct {
			Phase                   string `json:"phase"`
			ObservedKubeVirtVersion string `json:"observedKubeVirtVersion"`
		} `json:"status"`
	} `json:"items"`
}
type deploymentResponse struct {
	Status struct {
		ReadyReplicas int `json:"readyReplicas"`
	} `json:"status"`
}
type versionResponse struct {
	ServerVersion struct {
		GitVersion string `json:"gitVersion"`
	} `json:"serverVersion"`
}

type nodeCounts struct{ total, ready int }

func (p Provider) nodeStatus(ctx context.Context) (nodeCounts, error) {
	out, err := p.runner.Run(ctx, "kubectl", "get", "nodes", "-o", "json")
	if err != nil {
		return nodeCounts{}, err
	}
	var resp nodesResponse
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		return nodeCounts{}, err
	}
	res := nodeCounts{total: len(resp.Items)}
	for _, item := range resp.Items {
		for _, c := range item.Status.Conditions {
			if c.Type == "Ready" && c.Status == "True" {
				res.ready++
				break
			}
		}
	}
	return res, nil
}
func (p Provider) kubeVirtStatus(ctx context.Context) (bool, bool, error) {
	_, ready, err := p.kubeVirtVersionAndReady(ctx)
	if err != nil {
		return false, false, err
	}
	out, err := p.runner.Run(ctx, "kubectl", "get", "kubevirt", "-A", "-o", "json")
	if err != nil {
		return false, false, err
	}
	var resp kubevirtResponse
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		return false, false, err
	}
	return len(resp.Items) > 0, ready, nil
}
func (p Provider) kubeVirtVersionAndReady(ctx context.Context) (string, bool, error) {
	out, err := p.runner.Run(ctx, "kubectl", "get", "kubevirt", "-A", "-o", "json")
	if err != nil {
		return "unknown", false, err
	}
	var resp kubevirtResponse
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		return "unknown", false, err
	}
	if len(resp.Items) == 0 {
		return "unknown", false, nil
	}
	item := resp.Items[0]
	return item.Status.ObservedKubeVirtVersion, item.Status.Phase == "Deployed", nil
}
func (p Provider) cdiStatus(ctx context.Context) (bool, bool) {
	out, err := p.runner.Run(ctx, "kubectl", "get", "deployment", "-n", "cdi", "cdi-deployment", "-o", "json")
	if err != nil {
		return false, false
	}
	var resp deploymentResponse
	if json.Unmarshal([]byte(out), &resp) != nil {
		return true, false
	}
	return true, resp.Status.ReadyReplicas > 0
}
func (p Provider) kubernetesVersion(ctx context.Context) (string, error) {
	out, err := p.runner.Run(ctx, "kubectl", "version", "-o", "json")
	if err != nil {
		return "unknown", err
	}
	var resp versionResponse
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		return "unknown", err
	}
	return resp.ServerVersion.GitVersion, nil
}

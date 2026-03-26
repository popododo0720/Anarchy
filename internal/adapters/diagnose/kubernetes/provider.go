package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	kexec "github.com/popododo0720/anarchy/internal/adapters/kubernetes/exec"
	domaindiag "github.com/popododo0720/anarchy/internal/domain/diagnose"
)

type Provider struct {
	runner    kexec.Runner
	namespace string
}

func NewProvider(runner kexec.Runner, namespace string) Provider {
	if namespace == "" {
		namespace = "anarchy-system"
	}
	return Provider{runner: runner, namespace: namespace}
}

type nodesResponse struct {
	Items []struct {
		Status struct {
			Conditions []struct {
				Type   string `json:"type"`
				Status string `json:"status"`
			} `json:"conditions"`
		} `json:"status"`
	} `json:"items"`
}

type kubevirtResponse struct {
	Items []struct {
		Status struct {
			Phase string `json:"phase"`
		} `json:"status"`
	} `json:"items"`
}

type deploymentResponse struct {
	Status struct {
		ReadyReplicas int `json:"readyReplicas"`
	} `json:"status"`
}

type vmResponse struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec struct {
		Template struct {
			Metadata struct {
				Annotations map[string]string `json:"annotations"`
			} `json:"metadata"`
		} `json:"template"`
	} `json:"spec"`
	Status struct {
		PrintableStatus string `json:"printableStatus"`
	} `json:"status"`
}

type vmiResponse struct {
	Status struct {
		Phase string `json:"phase"`
	} `json:"status"`
}

type dataVolumeResponse struct {
	Status struct {
		Phase      string `json:"phase"`
		Conditions []struct {
			Type    string `json:"type"`
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"conditions"`
	} `json:"status"`
}

func (p Provider) DiagnoseCluster(ctx context.Context) (domaindiag.ClusterReport, error) {
	findings := []string{}
	checks := []domaindiag.Check{}
	status := "healthy"

	nodesOut, err := p.runner.Run(ctx, "kubectl", "get", "nodes", "-o", "json")
	if err != nil {
		return domaindiag.ClusterReport{}, err
	}
	var nodes nodesResponse
	if err := json.Unmarshal([]byte(nodesOut), &nodes); err != nil {
		return domaindiag.ClusterReport{}, err
	}
	total, ready := len(nodes.Items), 0
	for _, item := range nodes.Items {
		for _, condition := range item.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == "True" {
				ready++
				break
			}
		}
	}
	if ready != total {
		status = "degraded"
		findings = append(findings, fmt.Sprintf("%d/%d nodes ready", ready, total))
		checks = append(checks, domaindiag.Check{Name: "nodes", Status: "degraded", Message: findings[len(findings)-1]})
	} else {
		checks = append(checks, domaindiag.Check{Name: "nodes", Status: "healthy", Message: fmt.Sprintf("%d/%d nodes ready", ready, total)})
	}

	kubevirtOut, err := p.runner.Run(ctx, "kubectl", "get", "kubevirt", "-A", "-o", "json")
	if err != nil {
		return domaindiag.ClusterReport{}, err
	}
	var kubevirt kubevirtResponse
	if err := json.Unmarshal([]byte(kubevirtOut), &kubevirt); err != nil {
		return domaindiag.ClusterReport{}, err
	}
	kubevirtReady := len(kubevirt.Items) > 0 && kubevirt.Items[0].Status.Phase == "Deployed"
	if !kubevirtReady {
		status = "degraded"
		findings = append(findings, "kubevirt not ready")
		checks = append(checks, domaindiag.Check{Name: "kubevirt", Status: "degraded", Message: "kubevirt not ready"})
	} else {
		checks = append(checks, domaindiag.Check{Name: "kubevirt", Status: "healthy", Message: "kubevirt ready"})
	}

	cdiOut, err := p.runner.Run(ctx, "kubectl", "get", "deployment", "-n", "cdi", "cdi-deployment", "-o", "json")
	if err != nil {
		return domaindiag.ClusterReport{}, err
	}
	var cdi deploymentResponse
	if err := json.Unmarshal([]byte(cdiOut), &cdi); err != nil {
		return domaindiag.ClusterReport{}, err
	}
	if cdi.Status.ReadyReplicas == 0 {
		status = "degraded"
		findings = append(findings, "cdi not ready")
		checks = append(checks, domaindiag.Check{Name: "cdi", Status: "degraded", Message: "cdi not ready"})
	} else {
		checks = append(checks, domaindiag.Check{Name: "cdi", Status: "healthy", Message: "cdi ready"})
	}

	if len(findings) == 0 {
		findings = append(findings, "cluster looks healthy")
	}
	return domaindiag.ClusterReport{Status: status, Findings: findings, Checks: checks}, nil
}

func (p Provider) DiagnoseVM(ctx context.Context, name string) (domaindiag.VMReport, error) {
	vmOut, err := p.runner.Run(ctx, "kubectl", "-n", p.namespace, "get", "virtualmachine", name, "-o", "json")
	if err != nil {
		return domaindiag.VMReport{}, err
	}
	var vm vmResponse
	if err := json.Unmarshal([]byte(vmOut), &vm); err != nil {
		return domaindiag.VMReport{}, err
	}
	findings := []string{}
	checks := []domaindiag.Check{{Name: "vm", Status: strings.ToLower(vm.Status.PrintableStatus), Message: "vm status: " + vm.Status.PrintableStatus}}
	findings = append(findings, "vm status: "+vm.Status.PrintableStatus)

	vmiOut, err := p.runner.Run(ctx, "kubectl", "-n", p.namespace, "get", "virtualmachineinstance", name, "-o", "json")
	if err == nil {
		var vmi vmiResponse
		if json.Unmarshal([]byte(vmiOut), &vmi) == nil && vmi.Status.Phase != "" {
			findings = append(findings, "vmi phase: "+vmi.Status.Phase)
			checks = append(checks, domaindiag.Check{Name: "vmi", Status: strings.ToLower(vmi.Status.Phase), Message: "vmi phase: " + vmi.Status.Phase})
		}
	}

	dvOut, err := p.runner.Run(ctx, "kubectl", "-n", p.namespace, "get", "datavolume", name+"-rootdisk", "-o", "json")
	if err == nil {
		var dv dataVolumeResponse
		if json.Unmarshal([]byte(dvOut), &dv) == nil && dv.Status.Phase != "" {
			findings = append(findings, "datavolume phase: "+dv.Status.Phase)
			checks = append(checks, domaindiag.Check{Name: "datavolume", Status: strings.ToLower(dv.Status.Phase), Message: "datavolume phase: " + dv.Status.Phase})
			for _, condition := range dv.Status.Conditions {
				if condition.Message != "" && !strings.EqualFold(condition.Status, "True") {
					findings = append(findings, condition.Message)
				}
			}
		}
	}

	if len(findings) == 0 {
		findings = append(findings, "no issues detected")
	}
	return domaindiag.VMReport{Name: vm.Metadata.Name, Phase: vm.Status.PrintableStatus, Findings: findings, Checks: checks}, nil
}

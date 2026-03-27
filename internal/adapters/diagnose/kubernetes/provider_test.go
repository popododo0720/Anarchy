package kubernetes_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	kubediag "github.com/popododo0720/anarchy/internal/adapters/diagnose/kubernetes"
)

type fakeRunner struct {
	responses map[string]string
	errors    map[string]error
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	key := name + " " + strings.Join(args, " ")
	if err, ok := f.errors[key]; ok {
		return "", err
	}
	if out, ok := f.responses[key]; ok {
		return out, nil
	}
	return "", errors.New("unexpected command: " + key)
}

func TestDiagnoseClusterReportsReadinessBlockers(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl get nodes -o json":                            `{"items":[{"status":{"conditions":[{"type":"Ready","status":"False"}]}}]}`,
		"kubectl get kubevirt -A -o json":                      `{"items":[{"status":{"phase":"Deploying","observedKubeVirtVersion":"v1.4.0"}}]}`,
		"kubectl get deployment -n cdi cdi-deployment -o json": `{"status":{"readyReplicas":0}}`,
	}}
	provider := kubediag.NewProvider(runner, "anarchy-system")

	report, err := provider.DiagnoseCluster(context.Background())
	if err != nil {
		t.Fatalf("DiagnoseCluster() error = %v", err)
	}
	if report.Status != "degraded" {
		t.Fatalf("status = %q, want degraded", report.Status)
	}
	joined := strings.Join(report.Findings, " | ")
	for _, want := range []string{"0/1 nodes ready", "kubevirt not ready", "cdi not ready"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("findings = %#v, want to contain %q", report.Findings, want)
		}
	}
}

func TestDiagnoseVMReportsProvisioningBlockers(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl -n anarchy-system get virtualmachine testvm -o json":         `{"metadata":{"name":"testvm"},"status":{"printableStatus":"Provisioning"},"spec":{"template":{"metadata":{"annotations":{"anarchy.io/image":"ubuntu-24.04"}}}}}`,
		"kubectl -n anarchy-system get virtualmachineinstance testvm -o json": `{"status":{"phase":"Pending"}}`,
		"kubectl -n anarchy-system get datavolume testvm-rootdisk -o json":    `{"status":{"phase":"WaitForFirstConsumer","conditions":[{"type":"Ready","status":"False","message":"PVC testvm-rootdisk Pending"}]}}`,
	}}
	provider := kubediag.NewProvider(runner, "anarchy-system")

	report, err := provider.DiagnoseVM(context.Background(), "testvm")
	if err != nil {
		t.Fatalf("DiagnoseVM() error = %v", err)
	}
	if report.Name != "testvm" || report.Phase != "Provisioning" {
		t.Fatalf("report = %#v", report)
	}
	joined := strings.Join(report.Findings, " | ")
	for _, want := range []string{"vm status: Provisioning", "vmi phase: Pending", "datavolume phase: WaitForFirstConsumer", "PVC testvm-rootdisk Pending"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("findings = %#v, want to contain %q", report.Findings, want)
		}
	}
}

func TestDiagnosePublicIPReportsPendingRealization(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl get ovneip.kubeovn.io fip-01 -o json": `{"metadata":{"name":"fip-01","annotations":{"anarchy.io/attachment-target":"vm1:nic1"}},"status":{"v4Ip":"203.0.113.10"}}`,
	}, errors: map[string]error{
		"kubectl get ovnfip.kubeovn.io fip-01 -o json": errors.New("NotFound"),
	}}
	provider := kubediag.NewProvider(runner, "anarchy-system")

	report, err := provider.DiagnosePublicIP(context.Background(), "fip-01")
	if err != nil {
		t.Fatalf("DiagnosePublicIP() error = %v", err)
	}
	if report.Name != "fip-01" || report.Status != "pending" {
		t.Fatalf("report = %#v", report)
	}
	if report.Reason != "ovnfip_missing" || report.Code != "public_ip_not_realized" {
		t.Fatalf("report = %#v", report)
	}
	joined := strings.Join(report.Findings, " | ")
	for _, want := range []string{"public ip address: 203.0.113.10", "attachment target: vm1:nic1", "floating ip rule not realized yet"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("findings = %#v, want to contain %q", report.Findings, want)
		}
	}
}

func TestDiagnosePublicIPReportsCRDUnavailable(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl get ovneip.kubeovn.io fip-01 -o json": `{"metadata":{"name":"fip-01","annotations":{"anarchy.io/attachment-target":"vm1:nic1"}},"status":{"v4Ip":"203.0.113.10"}}`,
	}, errors: map[string]error{
		"kubectl get ovnfip.kubeovn.io fip-01 -o json": errors.New("the server doesn't have a resource type \"ovnfip\""),
	}}
	provider := kubediag.NewProvider(runner, "anarchy-system")

	report, err := provider.DiagnosePublicIP(context.Background(), "fip-01")
	if err != nil {
		t.Fatalf("DiagnosePublicIP() error = %v", err)
	}
	if report.Reason != "ovnfip_resource_unavailable" || report.Code != "public_ip_runtime_unavailable" {
		t.Fatalf("report = %#v", report)
	}
}

func TestDiagnosePublicIPReportsDetachedState(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl get ovneip.kubeovn.io fip-01 -o json": `{"metadata":{"name":"fip-01"},"status":{"v4Ip":"203.0.113.10"}}`,
	}, errors: map[string]error{
		"kubectl get ovnfip.kubeovn.io fip-01 -o json": errors.New("NotFound"),
	}}
	provider := kubediag.NewProvider(runner, "anarchy-system")

	report, err := provider.DiagnosePublicIP(context.Background(), "fip-01")
	if err != nil {
		t.Fatalf("DiagnosePublicIP() error = %v", err)
	}
	if report.Status != "detached" || report.Reason != "detached" || report.Code != "public_ip_detached" {
		t.Fatalf("report = %#v", report)
	}
}

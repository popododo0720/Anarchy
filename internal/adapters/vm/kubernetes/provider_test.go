package kubernetes_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	kubevm "github.com/popododo0720/anarchy/internal/adapters/vm/kubernetes"
	domainvm "github.com/popododo0720/anarchy/internal/domain/vm"
)

type fakeRunner struct {
	responses       map[string]string
	errors          map[string]error
	calls           []string
	appliedManifest string
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	key := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, key)
	if len(args) >= 5 && name == "kubectl" && args[0] == "-n" && args[2] == "apply" && args[3] == "-f" {
		content, err := os.ReadFile(args[4])
		if err != nil {
			return "", err
		}
		f.appliedManifest = string(content)
	}
	if err, ok := f.errors[key]; ok {
		return "", err
	}
	for k, out := range f.responses {
		if strings.HasSuffix(k, "*") && strings.HasPrefix(key, strings.TrimSuffix(k, "*")) {
			return out, nil
		}
	}
	if out, ok := f.responses[key]; ok {
		return out, nil
	}
	return "", errors.New("unexpected command: " + key)
}

func TestListVMsParsesVirtualMachines(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl -n anarchy-system get virtualmachines -o json":         `{"items":[{"metadata":{"name":"vm1"},"spec":{"template":{"spec":{"domain":{"cpu":{"cores":2},"resources":{"requests":{"memory":"4Gi"}}}}}},"status":{"printableStatus":"Running"}}]}`,
		"kubectl -n anarchy-system get virtualmachineinstances -o json": `{"items":[{"metadata":{"name":"vm1"},"status":{"interfaces":[{"ipAddress":"10.0.0.10"}]}}]}`,
	}}
	provider := kubevm.NewProvider(runner, "anarchy-system")

	got, err := provider.ListVMs(context.Background())
	if err != nil {
		t.Fatalf("ListVMs() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "vm1" || got[0].PrivateIP != "10.0.0.10" {
		t.Fatalf("ListVMs() = %#v", got)
	}
}

func TestGetVMReturnsDetail(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl -n anarchy-system get virtualmachine vm1 -o json":         `{"metadata":{"name":"vm1"},"spec":{"template":{"spec":{"domain":{"cpu":{"cores":2},"resources":{"requests":{"memory":"4Gi"}}},"networks":[{"name":"default"}]},"metadata":{"annotations":{"anarchy.io/image":"ubuntu-24.04"}}}},"status":{"printableStatus":"Running"}}`,
		"kubectl -n anarchy-system get virtualmachineinstance vm1 -o json": `{"status":{"interfaces":[{"ipAddress":"10.0.0.10"}]}}`,
	}}
	provider := kubevm.NewProvider(runner, "anarchy-system")

	got, err := provider.GetVM(context.Background(), "vm1")
	if err != nil {
		t.Fatalf("GetVM() error = %v", err)
	}
	if got.Name != "vm1" || got.Network != "default" || got.Image != "ubuntu-24.04" {
		t.Fatalf("GetVM() = %#v", got)
	}
}

func TestCreateVMAppliesManifest(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl -n anarchy-system apply -f *":                             "virtualmachine.kubevirt.io/vm1 created",
		"kubectl -n anarchy-system get virtualmachine vm1 -o json":         `{"metadata":{"name":"vm1"},"spec":{"template":{"spec":{"domain":{"cpu":{"cores":2},"resources":{"requests":{"memory":"4Gi"}}},"networks":[{"name":"default"}]},"metadata":{"annotations":{"anarchy.io/image":"ubuntu-24.04"}}}},"status":{"printableStatus":"Provisioning"}}`,
		"kubectl -n anarchy-system get virtualmachineinstance vm1 -o json": `{"status":{"interfaces":[{"ipAddress":""}]}}`,
	}}
	provider := kubevm.NewProvider(runner, "anarchy-system")

	got, err := provider.CreateVM(context.Background(), domainvm.CreateVMRequest{Name: "vm1", Image: "ubuntu-24.04", CPU: 2, Memory: "4Gi", Network: "default", SubnetRef: "tenant-a", NetworkAttachments: []domainvm.NetworkAttachment{{Name: "nic0", Network: "default", SubnetRef: "tenant-a", Primary: true}}})
	if err != nil {
		t.Fatalf("CreateVM() error = %v", err)
	}
	if got.Name != "vm1" {
		t.Fatalf("CreateVM() = %#v", got)
	}
	foundApply := false
	for _, call := range runner.calls {
		if strings.Contains(call, "kubectl -n anarchy-system apply -f ") {
			foundApply = true
			parts := strings.Split(call, " ")
			manifestPath := parts[len(parts)-1]
			if filepath.Ext(manifestPath) != ".yaml" {
				t.Fatalf("expected yaml manifest path, got %q", manifestPath)
			}
		}
	}
	if !foundApply {
		t.Fatal("expected kubectl apply call")
	}
	for _, want := range []string{
		"dataVolumeTemplates:",
		"cdi.kubevirt.io/storage.bind.immediate.requested: \"true\"",
		"sourceRef:",
		"kind: DataSource",
		"name: ubuntu-24.04",
		"namespace: anarchy-system",
		"interfaces:",
		"masquerade: {}",
		"networks:",
		"name: tenant-a",
		"dataVolume:",
		"name: vm1-rootdisk",
	} {
		if !strings.Contains(runner.appliedManifest, want) {
			t.Fatalf("manifest = %q, want to contain %q", runner.appliedManifest, want)
		}
	}
	if strings.Contains(runner.appliedManifest, "containerDisk:") {
		t.Fatalf("manifest should not use containerDisk anymore: %q", runner.appliedManifest)
	}
}

func TestStartStopRestartDeleteIssueCommands(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl -n anarchy-system patch virtualmachine vm1 --type merge -p {\"spec\":{\"running\":true}}":  "ok",
		"kubectl -n anarchy-system patch virtualmachine vm1 --type merge -p {\"spec\":{\"running\":false}}": "ok",
		"kubectl -n anarchy-system restart vm vm1":                                                          "ok",
		"kubectl -n anarchy-system delete virtualmachine vm1":                                               "ok",
	}}
	provider := kubevm.NewProvider(runner, "anarchy-system")
	if err := provider.StartVM(context.Background(), "vm1"); err != nil {
		t.Fatalf("StartVM() error = %v", err)
	}
	if err := provider.StopVM(context.Background(), "vm1"); err != nil {
		t.Fatalf("StopVM() error = %v", err)
	}
	if err := provider.RestartVM(context.Background(), "vm1"); err != nil {
		t.Fatalf("RestartVM() error = %v", err)
	}
	if err := provider.DeleteVM(context.Background(), "vm1"); err != nil {
		t.Fatalf("DeleteVM() error = %v", err)
	}
}

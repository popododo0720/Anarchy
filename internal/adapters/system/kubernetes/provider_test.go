package kubernetes_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	kubesystem "github.com/popododo0720/anarchy/internal/adapters/system/kubernetes"
	domainsystem "github.com/popododo0720/anarchy/internal/domain/system"
)

type fakeRunner struct {
	responses map[string]string
	errors    map[string]error
}

func (f fakeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	key := name + " " + strings.Join(args, " ")
	if err, ok := f.errors[key]; ok {
		return "", err
	}
	if out, ok := f.responses[key]; ok {
		return out, nil
	}
	return "", errors.New("unexpected command: " + key)
}

func TestGetHealthBuildsSummaryFromKubectl(t *testing.T) {
	runner := fakeRunner{responses: map[string]string{
		"kubectl version -o json":                              `{"serverVersion":{"gitVersion":"v1.32.0"}}`,
		"kubectl get nodes -o json":                            `{"items":[{"metadata":{"name":"node1","labels":{"kubevirt.io/schedulable":"true"}},"spec":{"unschedulable":false},"status":{"conditions":[{"type":"Ready","status":"True"}]}}]}`,
		"kubectl get kubevirt -A -o json":                      `{"items":[{"spec":{},"status":{"phase":"Deployed"}}]}`,
		"kubectl get deployment -n cdi cdi-deployment -o json": `{"status":{"readyReplicas":1}}`,
	}}
	provider := kubesystem.NewProvider(runner)

	got, err := provider.GetHealth(context.Background())
	if err != nil {
		t.Fatalf("GetHealth() error = %v", err)
	}
	if got.Status != domainsystem.StatusHealthy {
		t.Fatalf("status = %v, want %v", got.Status, domainsystem.StatusHealthy)
	}
	if got.ReadyNodes != 1 || got.TotalNodes != 1 {
		t.Fatalf("nodes = %d/%d, want 1/1", got.ReadyNodes, got.TotalNodes)
	}
	if !got.KubeVirtInstalled || !got.KubeVirtReady || !got.CDIInstalled || !got.CDIReady {
		t.Fatalf("unexpected component readiness: %#v", got)
	}
}

func TestGetHealthReturnsDegradedWhenKubeVirtMissing(t *testing.T) {
	runner := fakeRunner{responses: map[string]string{
		"kubectl version -o json":         `{"serverVersion":{"gitVersion":"v1.32.0"}}`,
		"kubectl get nodes -o json":       `{"items":[]}`,
		"kubectl get kubevirt -A -o json": `{"items":[]}`,
	}, errors: map[string]error{
		"kubectl get deployment -n cdi cdi-deployment -o json": errors.New("not found"),
	}}
	provider := kubesystem.NewProvider(runner)

	got, err := provider.GetHealth(context.Background())
	if err != nil {
		t.Fatalf("GetHealth() error = %v", err)
	}
	if got.Status != domainsystem.StatusDegraded {
		t.Fatalf("status = %v, want %v", got.Status, domainsystem.StatusDegraded)
	}
}

func TestGetVersionReturnsClusterVersions(t *testing.T) {
	runner := fakeRunner{responses: map[string]string{
		"kubectl version -o json":         `{"serverVersion":{"gitVersion":"v1.32.0"}}`,
		"kubectl get kubevirt -A -o json": `{"items":[{"status":{"observedKubeVirtVersion":"v1.5.0","phase":"Deployed"}}]}`,
	}}
	provider := kubesystem.NewProvider(runner)

	got, err := provider.GetVersion(context.Background())
	if err != nil {
		t.Fatalf("GetVersion() error = %v", err)
	}
	if got.KubernetesVersion != "v1.32.0" || got.KubeVirtVersion != "v1.5.0" {
		t.Fatalf("version summary = %#v", got)
	}
}

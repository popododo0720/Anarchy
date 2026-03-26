package kubernetes_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	kubenode "github.com/popododo0720/anarchy/internal/adapters/node/kubernetes"
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

func TestListNodesParsesKubectlJSON(t *testing.T) {
	runner := fakeRunner{responses: map[string]string{
		"kubectl get nodes -o json": `{"items":[{"metadata":{"name":"node1","labels":{"node-role.kubernetes.io/control-plane":"","kubevirt.io/schedulable":"true","ovn.kubernetes.io/zone-name":"global"}},"spec":{"unschedulable":false},"status":{"conditions":[{"type":"Ready","status":"True"}]}}]}`,
	}}
	provider := kubenode.NewProvider(runner)

	got, err := provider.ListNodes(context.Background())
	if err != nil {
		t.Fatalf("ListNodes() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "node1" || !got[0].VirtualizationCapable {
		t.Fatalf("ListNodes() = %#v", got)
	}
}

func TestGetNodeReturnsCapabilities(t *testing.T) {
	runner := fakeRunner{responses: map[string]string{
		"kubectl get nodes -o json": `{"items":[{"metadata":{"name":"node1","labels":{"node-role.kubernetes.io/control-plane":"","kubevirt.io/schedulable":"true","ovn.kubernetes.io/zone-name":"global"}},"spec":{"unschedulable":false},"status":{"conditions":[{"type":"Ready","status":"True"}]}}]}`,
	}}
	provider := kubenode.NewProvider(runner)

	got, err := provider.GetNode(context.Background(), "node1")
	if err != nil {
		t.Fatalf("GetNode() error = %v", err)
	}
	if len(got.Capabilities) == 0 {
		t.Fatalf("GetNode().Capabilities = %#v, want non-empty", got.Capabilities)
	}
}

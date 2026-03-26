package kubernetes_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	kubenetwork "github.com/popododo0720/anarchy/internal/adapters/network/kubernetes"
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

func TestListNetworksParsesKubeOvnVPCs(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl get vpcs.kubeovn.io -o json": `{"items":[{"metadata":{"name":"ovn-cluster"},"status":{"default":true,"defaultLogicalSwitch":"ovn-default","subnets":["ovn-default"]}}]}`,
	}}
	provider := kubenetwork.NewProvider(runner)

	got, err := provider.ListNetworks(context.Background())
	if err != nil {
		t.Fatalf("ListNetworks() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "ovn-cluster" || got[0].DefaultSubnet != "ovn-default" {
		t.Fatalf("ListNetworks() = %#v", got)
	}
}

func TestGetNetworkReturnsDetailedFields(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl get vpc.kubeovn.io ovn-cluster -o json": `{"metadata":{"name":"ovn-cluster"},"status":{"default":true,"defaultLogicalSwitch":"ovn-default","subnets":["ovn-default","join"],"router":"ovn-cluster"}}`,
	}}
	provider := kubenetwork.NewProvider(runner)

	got, err := provider.GetNetwork(context.Background(), "ovn-cluster")
	if err != nil {
		t.Fatalf("GetNetwork() error = %v", err)
	}
	if got.Name != "ovn-cluster" || !got.Default || len(got.Subnets) != 2 {
		t.Fatalf("GetNetwork() = %#v", got)
	}
}

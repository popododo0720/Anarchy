package kubernetes_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	kubenetwork "github.com/popododo0720/anarchy/internal/adapters/network/kubernetes"
	domainnetwork "github.com/popododo0720/anarchy/internal/domain/network"
)

type fakeRunner struct {
	responses map[string]string
	errors    map[string]error
	calls     []string
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	key := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, key)
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

func TestCreateNetworkAppliesManifestAndReturnsDetail(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl apply -f *":                          "vpc.kubeovn.io/tenant-c created",
		"kubectl get vpc.kubeovn.io tenant-c -o json": `{"metadata":{"name":"tenant-c"},"status":{"default":false,"defaultLogicalSwitch":"tenant-c-subnet","subnets":["tenant-c-subnet"],"router":"tenant-c"}}`,
	}}
	provider := kubenetwork.NewProvider(runner)

	got, err := provider.CreateNetwork(context.Background(), domainnetwork.CreateNetworkRequest{Name: "tenant-c"})
	if err != nil {
		t.Fatalf("CreateNetwork() error = %v", err)
	}
	if got.Name != "tenant-c" || got.Router != "tenant-c" {
		t.Fatalf("CreateNetwork() = %#v", got)
	}
	foundApply := false
	for _, call := range runner.calls {
		if strings.Contains(call, "kubectl apply -f ") {
			foundApply = true
		}
	}
	if !foundApply {
		t.Fatal("expected kubectl apply call")
	}
}

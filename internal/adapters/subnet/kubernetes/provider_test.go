package kubernetes_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	kubesubnet "github.com/popododo0720/anarchy/internal/adapters/subnet/kubernetes"
	domainsubnet "github.com/popododo0720/anarchy/internal/domain/subnet"
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

func TestListSubnetsParsesKubeOvnSubnets(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl get subnets.kubeovn.io -o json": `{"items":[{"metadata":{"name":"ovn-default"},"spec":{"cidrBlock":"10.16.0.0/16","gateway":"10.16.0.1","protocol":"IPv4","provider":"ovn","vlan":"","vpc":"ovn-cluster"}}]}`,
	}}
	provider := kubesubnet.NewProvider(runner)

	got, err := provider.ListSubnets(context.Background())
	if err != nil {
		t.Fatalf("ListSubnets() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "ovn-default" || got[0].CIDR != "10.16.0.0/16" || got[0].Network != "ovn-cluster" {
		t.Fatalf("ListSubnets() = %#v", got)
	}
}

func TestGetSubnetReturnsDetailedFields(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl get subnet.kubeovn.io ovn-default -o json": `{"metadata":{"name":"ovn-default"},"spec":{"cidrBlock":"10.16.0.0/16","gateway":"10.16.0.1","protocol":"IPv4","provider":"ovn","vlan":"","vpc":"ovn-cluster","namespaces":["anarchy-system","kube-system"]}}`,
	}}
	provider := kubesubnet.NewProvider(runner)

	got, err := provider.GetSubnet(context.Background(), "ovn-default")
	if err != nil {
		t.Fatalf("GetSubnet() error = %v", err)
	}
	if got.Name != "ovn-default" || got.Protocol != "IPv4" || len(got.Namespaces) != 2 {
		t.Fatalf("GetSubnet() = %#v", got)
	}
}

func TestCreateSubnetAppliesManifestAndReturnsDetail(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl apply -f *":                             "subnet.kubeovn.io/tenant-c created",
		"kubectl get subnet.kubeovn.io tenant-c -o json": `{"metadata":{"name":"tenant-c"},"spec":{"cidrBlock":"10.18.0.0/24","gateway":"10.18.0.1","protocol":"IPv4","provider":"tenant-c-net.anarchy-system.ovn","vlan":"","vpc":"ovn-cluster","namespaces":["anarchy-system"]}}`,
	}}
	provider := kubesubnet.NewProvider(runner)

	got, err := provider.CreateSubnet(context.Background(), domainsubnet.CreateSubnetRequest{
		Name:       "tenant-c",
		CIDR:       "10.18.0.0/24",
		Gateway:    "10.18.0.1",
		Protocol:   "IPv4",
		Provider:   "tenant-c-net.anarchy-system.ovn",
		Network:    "ovn-cluster",
		Namespaces: []string{"anarchy-system"},
	})
	if err != nil {
		t.Fatalf("CreateSubnet() error = %v", err)
	}
	if got.Name != "tenant-c" || got.Provider != "tenant-c-net.anarchy-system.ovn" {
		t.Fatalf("CreateSubnet() = %#v", got)
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

package kubernetes_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	kubesubnet "github.com/popododo0720/anarchy/internal/adapters/subnet/kubernetes"
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

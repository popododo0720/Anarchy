package kubernetes_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	kubenad "github.com/popododo0720/anarchy/internal/adapters/nad/kubernetes"
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

func TestListNADsParsesDefinitions(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl get network-attachment-definitions.k8s.cni.cncf.io -A -o json": `{"items":[{"metadata":{"name":"tenant-b-net","namespace":"anarchy-system"},"spec":{"config":"{\"cniVersion\":\"0.3.1\",\"type\":\"kube-ovn\",\"provider\":\"tenant-b.ovn\",\"server_socket\":\"/run/openvswitch/kube-ovn-daemon.sock\"}"}}]}`,
	}}
	provider := kubenad.NewProvider(runner)

	got, err := provider.ListNADs(context.Background())
	if err != nil {
		t.Fatalf("ListNADs() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "tenant-b-net" || got[0].Namespace != "anarchy-system" || got[0].Provider != "tenant-b.ovn" {
		t.Fatalf("ListNADs() = %#v", got)
	}
}

func TestGetNADReturnsDetail(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl get network-attachment-definition.k8s.cni.cncf.io tenant-b-net -n anarchy-system -o json": `{"metadata":{"name":"tenant-b-net","namespace":"anarchy-system"},"spec":{"config":"{\"cniVersion\":\"0.3.1\",\"type\":\"kube-ovn\",\"provider\":\"tenant-b.ovn\",\"server_socket\":\"/run/openvswitch/kube-ovn-daemon.sock\"}"}}`,
	}}
	provider := kubenad.NewProvider(runner)

	got, err := provider.GetNAD(context.Background(), "anarchy-system", "tenant-b-net")
	if err != nil {
		t.Fatalf("GetNAD() error = %v", err)
	}
	if got.Name != "tenant-b-net" || got.Type != "kube-ovn" || got.Provider != "tenant-b.ovn" {
		t.Fatalf("GetNAD() = %#v", got)
	}
}

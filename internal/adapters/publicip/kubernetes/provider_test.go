package kubernetes_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	kubepublicip "github.com/popododo0720/anarchy/internal/adapters/publicip/kubernetes"
	domainpublicip "github.com/popododo0720/anarchy/internal/domain/publicip"
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

func TestListPublicIPsParsesKubeOVNOvnEIPs(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl get ovneips.kubeovn.io -o json": `{"items":[{"metadata":{"name":"fip-01","annotations":{"anarchy.io/attachment-target":"vm1:nic0"}},"spec":{"v4Ip":"203.0.113.10"}}]}`,
	}}
	provider := kubepublicip.NewProvider(runner)

	got, err := provider.ListPublicIPs(context.Background())
	if err != nil {
		t.Fatalf("ListPublicIPs() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListPublicIPs() len = %d, want 1", len(got))
	}
	if got[0] != (domainpublicip.PublicIPSummary{Name: "fip-01", Address: "203.0.113.10", Attached: true, AttachmentTarget: "vm1:nic0"}) {
		t.Fatalf("ListPublicIPs() = %#v", got)
	}
}

func TestGetPublicIPReturnsStructuredDetail(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl get ovneip.kubeovn.io fip-01 -o json": `{"metadata":{"name":"fip-01"},"status":{"v4Ip":"203.0.113.10"}}`,
	}}
	provider := kubepublicip.NewProvider(runner)

	got, err := provider.GetPublicIP(context.Background(), "fip-01")
	if err != nil {
		t.Fatalf("GetPublicIP() error = %v", err)
	}
	if got != (domainpublicip.PublicIPDetail{Name: "fip-01", Address: "203.0.113.10", Attached: false, AttachmentTarget: "", Type: "floating"}) {
		t.Fatalf("GetPublicIP() = %#v", got)
	}
}

func TestAttachPublicIPPatchesOvnEIPAnnotations(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl patch ovneip.kubeovn.io fip-01 --type merge -p *": "ovneip.kubeovn.io/fip-01 patched",
		"kubectl get ovneip.kubeovn.io fip-01 -o json":             `{"metadata":{"name":"fip-01","annotations":{"anarchy.io/attachment-target":"vm1:nic1"}},"status":{"v4Ip":"203.0.113.10"}}`,
	}}
	provider := kubepublicip.NewProvider(runner)

	got, err := provider.AttachPublicIP(context.Background(), domainpublicip.AttachPublicIPRequest{Name: "fip-01", AttachmentTarget: "vm1:nic1"})
	if err != nil {
		t.Fatalf("AttachPublicIP() error = %v", err)
	}
	if got.AttachmentTarget != "vm1:nic1" || !got.Attached || got.Address != "203.0.113.10" {
		t.Fatalf("AttachPublicIP() = %#v", got)
	}
	if len(runner.calls) == 0 || !strings.Contains(runner.calls[0], `"anarchy.io/attachment-target":"vm1:nic1"`) || !strings.Contains(runner.calls[0], `"anarchy.io/attachment-vm":"vm1"`) || !strings.Contains(runner.calls[0], `"anarchy.io/attachment-nic":"nic1"`) {
		t.Fatalf("patch call = %q", runner.calls)
	}
}

func TestDetachPublicIPClearsOvnEIPAnnotations(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl patch ovneip.kubeovn.io fip-01 --type merge -p *": "ovneip.kubeovn.io/fip-01 patched",
		"kubectl get ovneip.kubeovn.io fip-01 -o json":             `{"metadata":{"name":"fip-01"},"status":{"v4Ip":"203.0.113.10"}}`,
	}}
	provider := kubepublicip.NewProvider(runner)

	got, err := provider.DetachPublicIP(context.Background(), "fip-01")
	if err != nil {
		t.Fatalf("DetachPublicIP() error = %v", err)
	}
	if got.Attached || got.AttachmentTarget != "" {
		t.Fatalf("DetachPublicIP() = %#v", got)
	}
	if len(runner.calls) == 0 || !strings.Contains(runner.calls[0], `"anarchy.io/attachment-target":null`) {
		t.Fatalf("patch call = %q", runner.calls)
	}
}

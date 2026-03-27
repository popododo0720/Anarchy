package kubernetes_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	kubepublicip "github.com/popododo0720/anarchy/internal/adapters/publicip/kubernetes"
	domainpublicip "github.com/popododo0720/anarchy/internal/domain/publicip"
)

type fakeRunner struct {
	responses         map[string]string
	errors            map[string]error
	sequenceResponses map[string][]string
	sequenceErrors    map[string][]error
	calls             []string
	manifests         []string
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	key := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, key)
	if len(args) >= 3 && name == "kubectl" && args[0] == "apply" && args[1] == "-f" {
		content, err := os.ReadFile(args[2])
		if err != nil {
			return "", err
		}
		f.manifests = append(f.manifests, string(content))
	}
	if err, ok := f.errors[key]; ok {
		return "", err
	}
	if seq, ok := f.sequenceErrors[key]; ok && len(seq) > 0 {
		err := seq[0]
		f.sequenceErrors[key] = seq[1:]
		if err != nil {
			return "", err
		}
	}
	if seq, ok := f.sequenceResponses[key]; ok && len(seq) > 0 {
		out := seq[0]
		f.sequenceResponses[key] = seq[1:]
		return out, nil
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
		"kubectl get ovnfips.kubeovn.io -o json": `{"items":[{"metadata":{"name":"fip-01"},"spec":{"ovnEip":"fip-01","ipName":"10.0.0.15"}}]}`,
	}}
	provider := kubepublicip.NewProvider(runner)

	got, err := provider.ListPublicIPs(context.Background())
	if err != nil {
		t.Fatalf("ListPublicIPs() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListPublicIPs() len = %d, want 1", len(got))
	}
	if got[0] != (domainpublicip.PublicIPSummary{Name: "fip-01", Address: "203.0.113.10", Attached: true, Realized: true, Status: "realized", AttachmentTarget: "vm1:nic0", TargetIPAddress: "10.0.0.15"}) {
		t.Fatalf("ListPublicIPs() = %#v", got)
	}
}

func TestListPublicIPsFallsBackWhenOvnFipCRDIsUnavailable(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string]string{
			"kubectl get ovneips.kubeovn.io -o json": `{"items":[{"metadata":{"name":"fip-01","annotations":{"anarchy.io/attachment-target":"vm1:nic0"}},"spec":{"v4Ip":"203.0.113.10"}}]}`,
		},
		errors: map[string]error{
			"kubectl get ovnfips.kubeovn.io -o json": errors.New("the server doesn't have a resource type \"ovnfips\""),
		},
	}
	provider := kubepublicip.NewProvider(runner)

	got, err := provider.ListPublicIPs(context.Background())
	if err != nil {
		t.Fatalf("ListPublicIPs() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListPublicIPs() len = %d, want 1", len(got))
	}
	if got[0] != (domainpublicip.PublicIPSummary{Name: "fip-01", Address: "203.0.113.10", Attached: true, Realized: false, Status: "pending", AttachmentTarget: "vm1:nic0"}) {
		t.Fatalf("ListPublicIPs() = %#v", got)
	}
}

func TestGetPublicIPReturnsStructuredDetail(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl get ovneip.kubeovn.io fip-01 -o json": `{"metadata":{"name":"fip-01"},"status":{"v4Ip":"203.0.113.10"}}`,
		"kubectl get ovnfip.kubeovn.io fip-01 -o json": `{"metadata":{"name":"fip-01"},"spec":{"ovnEip":"fip-01","ipName":"10.0.0.25"}}`,
	}}
	provider := kubepublicip.NewProvider(runner)

	got, err := provider.GetPublicIP(context.Background(), "fip-01")
	if err != nil {
		t.Fatalf("GetPublicIP() error = %v", err)
	}
	if got != (domainpublicip.PublicIPDetail{Name: "fip-01", Address: "203.0.113.10", Attached: true, Realized: true, Status: "realized", AttachmentTarget: "", TargetIPAddress: "10.0.0.25", Type: "floating"}) {
		t.Fatalf("GetPublicIP() = %#v", got)
	}
}

func TestGetPublicIPFallsBackWhenOvnFipCRDIsUnavailable(t *testing.T) {
	runner := &fakeRunner{
		responses: map[string]string{
			"kubectl get ovneip.kubeovn.io fip-01 -o json": `{"metadata":{"name":"fip-01","annotations":{"anarchy.io/attachment-target":"vm1:nic0"}},"status":{"v4Ip":"203.0.113.10"}}`,
		},
		errors: map[string]error{
			"kubectl get ovnfip.kubeovn.io fip-01 -o json": errors.New("the server doesn't have a resource type \"ovnfip\""),
		},
	}
	provider := kubepublicip.NewProvider(runner)

	got, err := provider.GetPublicIP(context.Background(), "fip-01")
	if err != nil {
		t.Fatalf("GetPublicIP() error = %v", err)
	}
	if got != (domainpublicip.PublicIPDetail{Name: "fip-01", Address: "203.0.113.10", Attached: true, Realized: false, Status: "pending", AttachmentTarget: "vm1:nic0", Type: "floating"}) {
		t.Fatalf("GetPublicIP() = %#v", got)
	}
}

func TestAttachPublicIPPatchesOvnEIPAnnotations(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl apply -f *": "ovnfip.kubeovn.io/fip-01 created",
		"kubectl patch ovneip.kubeovn.io fip-01 --type merge -p *": "ovneip.kubeovn.io/fip-01 patched",
	}, sequenceResponses: map[string][]string{
		"kubectl get ovneip.kubeovn.io fip-01 -o json": {
			`{"metadata":{"name":"fip-01","annotations":{"anarchy.io/attachment-target":"vm1:nic0"}},"status":{"v4Ip":"203.0.113.10"}}`,
			`{"metadata":{"name":"fip-01","annotations":{"anarchy.io/attachment-target":"vm1:nic1"}},"status":{"v4Ip":"203.0.113.10"}}`,
		},
		"kubectl get ovnfip.kubeovn.io fip-01 -o json": {
			`{"metadata":{"name":"fip-01"},"spec":{"ovnEip":"fip-01","ipName":"10.0.0.20"}}`,
			`{"metadata":{"name":"fip-01"},"spec":{"ovnEip":"fip-01","ipName":"10.0.0.25"}}`,
		},
	}}
	provider := kubepublicip.NewProvider(runner)

	got, err := provider.AttachPublicIP(context.Background(), domainpublicip.AttachPublicIPRequest{Name: "fip-01", AttachmentTarget: "vm1:nic1", TargetIPAddress: "10.0.0.25"})
	if err != nil {
		t.Fatalf("AttachPublicIP() error = %v", err)
	}
	if got.AttachmentTarget != "vm1:nic1" || !got.Attached || got.Address != "203.0.113.10" {
		t.Fatalf("AttachPublicIP() = %#v", got)
	}
	joinedCalls := strings.Join(runner.calls, "\n")
	if !strings.Contains(joinedCalls, `"anarchy.io/attachment-target":"vm1:nic1"`) || !strings.Contains(joinedCalls, `"anarchy.io/attachment-vm":"vm1"`) || !strings.Contains(joinedCalls, `"anarchy.io/attachment-nic":"nic1"`) {
		t.Fatalf("patch call = %q", runner.calls)
	}
	if len(runner.manifests) != 1 {
		t.Fatalf("manifests = %#v, want 1 manifest", runner.manifests)
	}
	for _, want := range []string{"kind: OvnFip", "name: fip-01", "ovnEip: fip-01", "ipName: 10.0.0.25"} {
		if !strings.Contains(runner.manifests[0], want) {
			t.Fatalf("manifest = %q, want to contain %q", runner.manifests[0], want)
		}
	}
}

func TestAttachPublicIPIsNoOpWhenAlreadyRealizedForSameTarget(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl get ovneip.kubeovn.io fip-01 -o json": `{"metadata":{"name":"fip-01","annotations":{"anarchy.io/attachment-target":"vm1:nic1"}},"status":{"v4Ip":"203.0.113.10"}}`,
		"kubectl get ovnfip.kubeovn.io fip-01 -o json": `{"metadata":{"name":"fip-01"},"spec":{"ovnEip":"fip-01","ipName":"10.0.0.25"}}`,
	}}
	provider := kubepublicip.NewProvider(runner)

	got, err := provider.AttachPublicIP(context.Background(), domainpublicip.AttachPublicIPRequest{Name: "fip-01", AttachmentTarget: "vm1:nic1", TargetIPAddress: "10.0.0.25"})
	if err != nil {
		t.Fatalf("AttachPublicIP() error = %v", err)
	}
	if got.Status != "realized" || !got.Realized {
		t.Fatalf("AttachPublicIP() = %#v", got)
	}
	joinedCalls := strings.Join(runner.calls, "\n")
	if strings.Contains(joinedCalls, "kubectl apply -f ") || strings.Contains(joinedCalls, "kubectl patch ovneip.kubeovn.io") {
		t.Fatalf("calls = %#v, want no apply/patch for idempotent attach", runner.calls)
	}
}

func TestDetachPublicIPClearsOvnEIPAnnotations(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl delete ovnfip.kubeovn.io fip-01":                  "ovnfip.kubeovn.io \"fip-01\" deleted",
		"kubectl patch ovneip.kubeovn.io fip-01 --type merge -p *": "ovneip.kubeovn.io/fip-01 patched",
	}, sequenceResponses: map[string][]string{
		"kubectl get ovneip.kubeovn.io fip-01 -o json": {
			`{"metadata":{"name":"fip-01","annotations":{"anarchy.io/attachment-target":"vm1:nic1"}},"status":{"v4Ip":"203.0.113.10"}}`,
			`{"metadata":{"name":"fip-01"},"status":{"v4Ip":"203.0.113.10"}}`,
		},
	}, sequenceErrors: map[string][]error{
		"kubectl get ovnfip.kubeovn.io fip-01 -o json": {nil, errors.New("NotFound")},
	}}
	provider := kubepublicip.NewProvider(runner)

	got, err := provider.DetachPublicIP(context.Background(), "fip-01")
	if err != nil {
		t.Fatalf("DetachPublicIP() error = %v", err)
	}
	if got.Attached || got.AttachmentTarget != "" {
		t.Fatalf("DetachPublicIP() = %#v", got)
	}
	joinedCalls := strings.Join(runner.calls, "\n")
	if !strings.Contains(joinedCalls, `"anarchy.io/attachment-target":null`) {
		t.Fatalf("patch call = %q", runner.calls)
	}
	if !strings.Contains(joinedCalls, "kubectl delete ovnfip.kubeovn.io fip-01") {
		t.Fatalf("calls = %#v, want ovnfip delete", runner.calls)
	}
}

func TestDetachPublicIPIgnoresMissingOvnFip(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl patch ovneip.kubeovn.io fip-01 --type merge -p *": "ovneip.kubeovn.io/fip-01 patched",
	}, sequenceResponses: map[string][]string{
		"kubectl get ovneip.kubeovn.io fip-01 -o json": {
			`{"metadata":{"name":"fip-01","annotations":{"anarchy.io/attachment-target":"vm1:nic1"}},"status":{"v4Ip":"203.0.113.10"}}`,
			`{"metadata":{"name":"fip-01"},"status":{"v4Ip":"203.0.113.10"}}`,
		},
	}, errors: map[string]error{
		"kubectl delete ovnfip.kubeovn.io fip-01": errors.New("NotFound"),
	}, sequenceErrors: map[string][]error{
		"kubectl get ovnfip.kubeovn.io fip-01 -o json": {errors.New("NotFound"), errors.New("NotFound")},
	}}
	provider := kubepublicip.NewProvider(runner)

	got, err := provider.DetachPublicIP(context.Background(), "fip-01")
	if err != nil {
		t.Fatalf("DetachPublicIP() error = %v", err)
	}
	if got.Status != "detached" || got.Attached {
		t.Fatalf("DetachPublicIP() = %#v", got)
	}
	joinedCalls := strings.Join(runner.calls, "\n")
	if !strings.Contains(joinedCalls, "kubectl patch ovneip.kubeovn.io fip-01") {
		t.Fatalf("calls = %#v, want patch after missing delete", runner.calls)
	}
}

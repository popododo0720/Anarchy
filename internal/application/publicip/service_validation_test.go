package publicip_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	apppublicip "github.com/popododo0720/anarchy/internal/application/publicip"
	domainpublicip "github.com/popododo0720/anarchy/internal/domain/publicip"
	domainvm "github.com/popododo0720/anarchy/internal/domain/vm"
)

type fakeVMProvider struct {
	vmDetail domainvm.VMDetail
	err      error
}

func (f fakeVMProvider) CreateVM(context.Context, domainvm.CreateVMRequest) (domainvm.VMDetail, error) {
	panic("unexpected CreateVM call")
}

func (f fakeVMProvider) ListVMs(context.Context) ([]domainvm.VMSummary, error) {
	panic("unexpected ListVMs call")
}

func (f fakeVMProvider) GetVM(context.Context, string) (domainvm.VMDetail, error) {
	if f.err != nil {
		return domainvm.VMDetail{}, f.err
	}
	return f.vmDetail, nil
}

func (f fakeVMProvider) StartVM(context.Context, string) error   { panic("unexpected StartVM call") }
func (f fakeVMProvider) StopVM(context.Context, string) error    { panic("unexpected StopVM call") }
func (f fakeVMProvider) RestartVM(context.Context, string) error { panic("unexpected RestartVM call") }
func (f fakeVMProvider) DeleteVM(context.Context, string) error  { panic("unexpected DeleteVM call") }

func TestServiceAttachPublicIPRejectsInvalidAttachmentTarget(t *testing.T) {
	svc := apppublicip.NewService(fakeProvider{}, nil)

	_, err := svc.AttachPublicIP(context.Background(), domainpublicip.AttachPublicIPRequest{Name: "fip-01", AttachmentTarget: "vm1"})
	if err == nil {
		t.Fatal("AttachPublicIP() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "attachment target") {
		t.Fatalf("AttachPublicIP() error = %v, want attachment target validation error", err)
	}
}

func TestServiceAttachPublicIPRejectsUnknownNIC(t *testing.T) {
	svc := apppublicip.NewService(fakeProvider{}, fakeVMProvider{vmDetail: domainvm.VMDetail{
		Name:               "vm1",
		NetworkAttachments: []domainvm.NetworkAttachment{{Name: "nic0"}},
	}})

	_, err := svc.AttachPublicIP(context.Background(), domainpublicip.AttachPublicIPRequest{Name: "fip-01", AttachmentTarget: "vm1:nic1"})
	if err == nil {
		t.Fatal("AttachPublicIP() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "nic") {
		t.Fatalf("AttachPublicIP() error = %v, want nic validation error", err)
	}
}

func TestServiceAttachPublicIPUsesNormalizedRequest(t *testing.T) {
	provider := &capturingPublicIPProvider{}
	svc := apppublicip.NewService(provider, fakeVMProvider{vmDetail: domainvm.VMDetail{
		Name:               "vm1",
		NetworkAttachments: []domainvm.NetworkAttachment{{Name: "nic1"}},
	}})

	item, err := svc.AttachPublicIP(context.Background(), domainpublicip.AttachPublicIPRequest{Name: " fip-01 ", AttachmentTarget: " vm1:nic1 "})
	if err != nil {
		t.Fatalf("AttachPublicIP() error = %v", err)
	}
	if provider.lastReq.Name != "fip-01" || provider.lastReq.AttachmentTarget != "vm1:nic1" {
		t.Fatalf("provider request = %#v", provider.lastReq)
	}
	if item.Name != "fip-01" || item.AttachmentTarget != "vm1:nic1" {
		t.Fatalf("AttachPublicIP() = %#v", item)
	}
}

func TestServiceDetachPublicIPRejectsBlankName(t *testing.T) {
	svc := apppublicip.NewService(fakeProvider{}, nil)

	_, err := svc.DetachPublicIP(context.Background(), "   ")
	if err == nil {
		t.Fatal("DetachPublicIP() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "public ip name") {
		t.Fatalf("DetachPublicIP() error = %v", err)
	}
}

func TestServiceAttachPublicIPPropagatesVMLookupError(t *testing.T) {
	expectedErr := errors.New("vm lookup failed")
	svc := apppublicip.NewService(fakeProvider{}, fakeVMProvider{err: expectedErr})

	_, err := svc.AttachPublicIP(context.Background(), domainpublicip.AttachPublicIPRequest{Name: "fip-01", AttachmentTarget: "vm1:nic1"})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("AttachPublicIP() error = %v, want %v", err, expectedErr)
	}
}

type capturingPublicIPProvider struct {
	lastReq domainpublicip.AttachPublicIPRequest
}

func (p *capturingPublicIPProvider) ListPublicIPs(context.Context) ([]domainpublicip.PublicIPSummary, error) {
	panic("unexpected ListPublicIPs call")
}

func (p *capturingPublicIPProvider) GetPublicIP(context.Context, string) (domainpublicip.PublicIPDetail, error) {
	panic("unexpected GetPublicIP call")
}

func (p *capturingPublicIPProvider) AttachPublicIP(_ context.Context, req domainpublicip.AttachPublicIPRequest) (domainpublicip.PublicIPDetail, error) {
	p.lastReq = req
	return domainpublicip.PublicIPDetail{Name: req.Name, Address: "203.0.113.10", Attached: true, AttachmentTarget: req.AttachmentTarget, Type: "floating"}, nil
}

func (p *capturingPublicIPProvider) DetachPublicIP(context.Context, string) (domainpublicip.PublicIPDetail, error) {
	panic("unexpected DetachPublicIP call")
}

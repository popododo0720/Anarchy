package publicip_test

import (
	"context"
	"testing"

	apppublicip "github.com/popododo0720/anarchy/internal/application/publicip"
	domainpublicip "github.com/popododo0720/anarchy/internal/domain/publicip"
)

type fakeProvider struct{}

func (fakeProvider) ListPublicIPs(context.Context) ([]domainpublicip.PublicIPSummary, error) {
	return []domainpublicip.PublicIPSummary{{Name: "fip-01", Address: "203.0.113.10", Attached: true, AttachmentTarget: "vm1:nic0"}}, nil
}

func (fakeProvider) GetPublicIP(context.Context, string) (domainpublicip.PublicIPDetail, error) {
	return domainpublicip.PublicIPDetail{Name: "fip-01", Address: "203.0.113.10", Attached: true, AttachmentTarget: "vm1:nic0", Type: "floating"}, nil
}

func (fakeProvider) AttachPublicIP(context.Context, domainpublicip.AttachPublicIPRequest) (domainpublicip.PublicIPDetail, error) {
	return domainpublicip.PublicIPDetail{Name: "fip-01", Address: "203.0.113.10", Attached: true, AttachmentTarget: "vm1:nic1", Type: "floating"}, nil
}

func (fakeProvider) DetachPublicIP(context.Context, string) (domainpublicip.PublicIPDetail, error) {
	return domainpublicip.PublicIPDetail{Name: "fip-01", Address: "203.0.113.10", Attached: false, AttachmentTarget: "", Type: "floating"}, nil
}

func TestServiceDelegatesToProvider(t *testing.T) {
	svc := apppublicip.NewService(fakeProvider{})
	items, err := svc.ListPublicIPs(context.Background())
	if err != nil {
		t.Fatalf("ListPublicIPs() error = %v", err)
	}
	if len(items) != 1 || items[0].Name != "fip-01" {
		t.Fatalf("ListPublicIPs() = %#v", items)
	}
	detail, err := svc.GetPublicIP(context.Background(), "fip-01")
	if err != nil {
		t.Fatalf("GetPublicIP() error = %v", err)
	}
	if detail.Type != "floating" {
		t.Fatalf("GetPublicIP() = %#v", detail)
	}
	attached, err := svc.AttachPublicIP(context.Background(), domainpublicip.AttachPublicIPRequest{Name: "fip-01", AttachmentTarget: "vm1:nic1"})
	if err != nil {
		t.Fatalf("AttachPublicIP() error = %v", err)
	}
	if attached.AttachmentTarget != "vm1:nic1" {
		t.Fatalf("AttachPublicIP() = %#v", attached)
	}
	detached, err := svc.DetachPublicIP(context.Background(), "fip-01")
	if err != nil {
		t.Fatalf("DetachPublicIP() error = %v", err)
	}
	if detached.Attached {
		t.Fatalf("DetachPublicIP() = %#v", detached)
	}
}

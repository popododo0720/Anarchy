package publicip

import (
	"context"
	"fmt"
	"strings"

	domainpublicip "github.com/popododo0720/anarchy/internal/domain/publicip"
	domainvm "github.com/popododo0720/anarchy/internal/domain/vm"
	portpublicip "github.com/popododo0720/anarchy/internal/ports/publicip"
	portvm "github.com/popododo0720/anarchy/internal/ports/vm"
)

type Service struct {
	provider   portpublicip.Provider
	vmProvider portvm.Provider
}

func NewService(provider portpublicip.Provider, vmProvider portvm.Provider) *Service {
	return &Service{provider: provider, vmProvider: vmProvider}
}

func (s *Service) ListPublicIPs(ctx context.Context) ([]domainpublicip.PublicIPSummary, error) {
	return s.provider.ListPublicIPs(ctx)
}

func (s *Service) GetPublicIP(ctx context.Context, name string) (domainpublicip.PublicIPDetail, error) {
	return s.provider.GetPublicIP(ctx, name)
}

func (s *Service) AttachPublicIP(ctx context.Context, req domainpublicip.AttachPublicIPRequest) (domainpublicip.PublicIPDetail, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return domainpublicip.PublicIPDetail{}, fmt.Errorf("public ip name is required")
	}
	target, err := domainpublicip.ParseAttachmentTarget(req.AttachmentTarget)
	if err != nil {
		return domainpublicip.PublicIPDetail{}, err
	}
	if s.vmProvider != nil {
		vmDetail, err := s.vmProvider.GetVM(ctx, target.VMName)
		if err != nil {
			return domainpublicip.PublicIPDetail{}, err
		}
		if !hasAttachment(vmDetail.NetworkAttachments, target.NICName) {
			return domainpublicip.PublicIPDetail{}, fmt.Errorf("nic %q not found on vm %q", target.NICName, target.VMName)
		}
	}
	req.Name = name
	req.AttachmentTarget = target.VMName + ":" + target.NICName
	return s.provider.AttachPublicIP(ctx, req)
}

func (s *Service) DetachPublicIP(ctx context.Context, name string) (domainpublicip.PublicIPDetail, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return domainpublicip.PublicIPDetail{}, fmt.Errorf("public ip name is required")
	}
	return s.provider.DetachPublicIP(ctx, trimmed)
}

func hasAttachment(items []domainvm.NetworkAttachment, nicName string) bool {
	for _, item := range items {
		if item.Name == nicName {
			return true
		}
	}
	return false
}

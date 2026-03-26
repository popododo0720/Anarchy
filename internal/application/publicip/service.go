package publicip

import (
	"context"

	domainpublicip "github.com/popododo0720/anarchy/internal/domain/publicip"
	portpublicip "github.com/popododo0720/anarchy/internal/ports/publicip"
)

type Service struct {
	provider portpublicip.Provider
}

func NewService(provider portpublicip.Provider) *Service {
	return &Service{provider: provider}
}

func (s *Service) ListPublicIPs(ctx context.Context) ([]domainpublicip.PublicIPSummary, error) {
	return s.provider.ListPublicIPs(ctx)
}

func (s *Service) GetPublicIP(ctx context.Context, name string) (domainpublicip.PublicIPDetail, error) {
	return s.provider.GetPublicIP(ctx, name)
}

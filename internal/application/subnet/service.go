package subnet

import (
	"context"

	domainsubnet "github.com/popododo0720/anarchy/internal/domain/subnet"
	portsubnet "github.com/popododo0720/anarchy/internal/ports/subnet"
)

type Service struct {
	provider portsubnet.Provider
}

func NewService(provider portsubnet.Provider) *Service {
	return &Service{provider: provider}
}

func (s *Service) ListSubnets(ctx context.Context) ([]domainsubnet.SubnetSummary, error) {
	return s.provider.ListSubnets(ctx)
}

func (s *Service) GetSubnet(ctx context.Context, name string) (domainsubnet.SubnetDetail, error) {
	return s.provider.GetSubnet(ctx, name)
}

func (s *Service) CreateSubnet(ctx context.Context, req domainsubnet.CreateSubnetRequest) (domainsubnet.SubnetDetail, error) {
	return s.provider.CreateSubnet(ctx, req)
}

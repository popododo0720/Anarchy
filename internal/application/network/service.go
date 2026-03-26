package network

import (
	"context"

	domainnetwork "github.com/popododo0720/anarchy/internal/domain/network"
	portnetwork "github.com/popododo0720/anarchy/internal/ports/network"
)

type Service struct {
	provider portnetwork.Provider
}

func NewService(provider portnetwork.Provider) *Service {
	return &Service{provider: provider}
}

func (s *Service) ListNetworks(ctx context.Context) ([]domainnetwork.NetworkSummary, error) {
	return s.provider.ListNetworks(ctx)
}

func (s *Service) GetNetwork(ctx context.Context, name string) (domainnetwork.NetworkDetail, error) {
	return s.provider.GetNetwork(ctx, name)
}

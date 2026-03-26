package system

import (
	"context"

	domainsystem "github.com/popododo0720/anarchy/internal/domain/system"
	portsystem "github.com/popododo0720/anarchy/internal/ports/system"
)

type Service struct {
	provider portsystem.Provider
}

func NewService(provider portsystem.Provider) *Service {
	return &Service{provider: provider}
}

func (s *Service) GetHealth(ctx context.Context) (domainsystem.HealthSummary, error) {
	return s.provider.GetHealth(ctx)
}

func (s *Service) GetVersion(ctx context.Context) (domainsystem.VersionSummary, error) {
	return s.provider.GetVersion(ctx)
}

func (s *Service) GetCapabilities(ctx context.Context) (domainsystem.CapabilitiesSummary, error) {
	return s.provider.GetCapabilities(ctx)
}

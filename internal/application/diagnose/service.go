package diagnose

import (
	"context"

	domaindiag "github.com/popododo0720/anarchy/internal/domain/diagnose"
	portdiag "github.com/popododo0720/anarchy/internal/ports/diagnose"
)

type Service struct {
	provider portdiag.Provider
}

func NewService(provider portdiag.Provider) *Service {
	return &Service{provider: provider}
}

func (s *Service) DiagnoseCluster(ctx context.Context) (domaindiag.ClusterReport, error) {
	return s.provider.DiagnoseCluster(ctx)
}

func (s *Service) DiagnoseVM(ctx context.Context, name string) (domaindiag.VMReport, error) {
	return s.provider.DiagnoseVM(ctx, name)
}

func (s *Service) DiagnosePublicIP(ctx context.Context, name string) (domaindiag.PublicIPReport, error) {
	return s.provider.DiagnosePublicIP(ctx, name)
}

package nad

import (
	"context"

	domainnad "github.com/popododo0720/anarchy/internal/domain/nad"
	portnad "github.com/popododo0720/anarchy/internal/ports/nad"
)

type Service struct {
	provider portnad.Provider
}

func NewService(provider portnad.Provider) *Service {
	return &Service{provider: provider}
}

func (s *Service) ListNADs(ctx context.Context) ([]domainnad.NADSummary, error) {
	return s.provider.ListNADs(ctx)
}

func (s *Service) GetNAD(ctx context.Context, namespace, name string) (domainnad.NADDetail, error) {
	return s.provider.GetNAD(ctx, namespace, name)
}

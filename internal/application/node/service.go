package node

import (
	"context"

	domainnode "github.com/popododo0720/anarchy/internal/domain/node"
	portnode "github.com/popododo0720/anarchy/internal/ports/node"
)

type Service struct {
	provider portnode.Provider
}

func NewService(provider portnode.Provider) *Service {
	return &Service{provider: provider}
}

func (s *Service) ListNodes(ctx context.Context) ([]domainnode.NodeSummary, error) {
	return s.provider.ListNodes(ctx)
}

func (s *Service) GetNode(ctx context.Context, name string) (domainnode.NodeDetail, error) {
	return s.provider.GetNode(ctx, name)
}

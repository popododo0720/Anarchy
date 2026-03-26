package node

import (
	"context"

	domainnode "github.com/popododo0720/anarchy/internal/domain/node"
)

type Provider interface {
	ListNodes(ctx context.Context) ([]domainnode.NodeSummary, error)
	GetNode(ctx context.Context, name string) (domainnode.NodeDetail, error)
}

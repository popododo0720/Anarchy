package network

import (
	"context"

	domainnetwork "github.com/popododo0720/anarchy/internal/domain/network"
)

type Provider interface {
	ListNetworks(ctx context.Context) ([]domainnetwork.NetworkSummary, error)
	GetNetwork(ctx context.Context, name string) (domainnetwork.NetworkDetail, error)
	CreateNetwork(ctx context.Context, req domainnetwork.CreateNetworkRequest) (domainnetwork.NetworkDetail, error)
}

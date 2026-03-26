package subnet

import (
	"context"

	domainsubnet "github.com/popododo0720/anarchy/internal/domain/subnet"
)

type Provider interface {
	ListSubnets(ctx context.Context) ([]domainsubnet.SubnetSummary, error)
	GetSubnet(ctx context.Context, name string) (domainsubnet.SubnetDetail, error)
	CreateSubnet(ctx context.Context, req domainsubnet.CreateSubnetRequest) (domainsubnet.SubnetDetail, error)
}

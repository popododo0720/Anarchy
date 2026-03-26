package publicip

import (
	"context"

	domainpublicip "github.com/popododo0720/anarchy/internal/domain/publicip"
)

type Provider interface {
	ListPublicIPs(ctx context.Context) ([]domainpublicip.PublicIPSummary, error)
	GetPublicIP(ctx context.Context, name string) (domainpublicip.PublicIPDetail, error)
}

package nad

import (
	"context"

	domainnad "github.com/popododo0720/anarchy/internal/domain/nad"
)

type Provider interface {
	ListNADs(ctx context.Context) ([]domainnad.NADSummary, error)
	GetNAD(ctx context.Context, namespace, name string) (domainnad.NADDetail, error)
}

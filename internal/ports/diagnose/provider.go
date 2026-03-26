package diagnose

import (
	"context"

	domaindiag "github.com/popododo0720/anarchy/internal/domain/diagnose"
)

type Provider interface {
	DiagnoseCluster(ctx context.Context) (domaindiag.ClusterReport, error)
	DiagnoseVM(ctx context.Context, name string) (domaindiag.VMReport, error)
}

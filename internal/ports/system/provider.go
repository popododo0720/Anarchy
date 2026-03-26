package system

import (
	"context"

	domainsystem "github.com/popododo0720/anarchy/internal/domain/system"
)

type Provider interface {
	GetHealth(ctx context.Context) (domainsystem.HealthSummary, error)
	GetVersion(ctx context.Context) (domainsystem.VersionSummary, error)
	GetCapabilities(ctx context.Context) (domainsystem.CapabilitiesSummary, error)
}

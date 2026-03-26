package static

import (
	"context"

	domainsystem "github.com/popododo0720/anarchy/internal/domain/system"
)

type Provider struct{}

func NewProvider() Provider {
	return Provider{}
}

func (Provider) GetHealth(context.Context) (domainsystem.HealthSummary, error) {
	return domainsystem.HealthSummary{
		Status:              domainsystem.StatusHealthy,
		APIReachable:        true,
		KubernetesReachable: true,
		KubeVirtInstalled:   true,
		KubeVirtReady:       true,
		CDIInstalled:        true,
		CDIReady:            true,
		TotalNodes:          1,
		ReadyNodes:          1,
		Warnings:            nil,
	}, nil
}

func (Provider) GetVersion(context.Context) (domainsystem.VersionSummary, error) {
	return domainsystem.VersionSummary{
		CLIVersion:           "dev",
		APIVersion:           "v1",
		ServerVersion:        "dev",
		SupportedAPIVersions: []string{"v1"},
		KubernetesVersion:    "unknown",
		KubeVirtVersion:      "unknown",
	}, nil
}

func (Provider) GetCapabilities(context.Context) (domainsystem.CapabilitiesSummary, error) {
	return domainsystem.CapabilitiesSummary{
		VMLifecycleSupported:    true,
		ImageInventorySupported: true,
		DiagnosticsSupported:    true,
		PublicIPSupported:       false,
		Capabilities:            []string{"vm-lifecycle", "image-inventory", "diagnostics"},
	}, nil
}

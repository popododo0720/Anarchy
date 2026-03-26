package system_test

import (
	"context"
	"errors"
	"testing"

	appsystem "github.com/popododo0720/anarchy/internal/application/system"
	domainsystem "github.com/popododo0720/anarchy/internal/domain/system"
	portsystem "github.com/popododo0720/anarchy/internal/ports/system"
)

type fakeProvider struct {
	health       domainsystem.HealthSummary
	version      domainsystem.VersionSummary
	capabilities domainsystem.CapabilitiesSummary
	healthErr    error
	versionErr   error
	capErr       error
}

func (f fakeProvider) GetHealth(context.Context) (domainsystem.HealthSummary, error) {
	return f.health, f.healthErr
}

func (f fakeProvider) GetVersion(context.Context) (domainsystem.VersionSummary, error) {
	return f.version, f.versionErr
}

func (f fakeProvider) GetCapabilities(context.Context) (domainsystem.CapabilitiesSummary, error) {
	return f.capabilities, f.capErr
}

var _ portsystem.Provider = fakeProvider{}

func TestServiceGetHealthDelegatesToProvider(t *testing.T) {
	expected := domainsystem.HealthSummary{Status: domainsystem.StatusDegraded, ReadyNodes: 1, TotalNodes: 3}
	service := appsystem.NewService(fakeProvider{health: expected})

	got, err := service.GetHealth(context.Background())
	if err != nil {
		t.Fatalf("GetHealth() error = %v", err)
	}
	if got.Status != expected.Status || got.ReadyNodes != expected.ReadyNodes || got.TotalNodes != expected.TotalNodes {
		t.Fatalf("GetHealth() = %#v, want %#v", got, expected)
	}
}

func TestServiceGetHealthReturnsProviderError(t *testing.T) {
	expectedErr := errors.New("boom")
	service := appsystem.NewService(fakeProvider{healthErr: expectedErr})

	_, err := service.GetHealth(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("GetHealth() error = %v, want %v", err, expectedErr)
	}
}

func TestServiceGetVersionDelegatesToProvider(t *testing.T) {
	expected := domainsystem.VersionSummary{APIVersion: "v1", ServerVersion: "dev"}
	service := appsystem.NewService(fakeProvider{version: expected})

	got, err := service.GetVersion(context.Background())
	if err != nil {
		t.Fatalf("GetVersion() error = %v", err)
	}
	if got.APIVersion != expected.APIVersion || got.ServerVersion != expected.ServerVersion {
		t.Fatalf("GetVersion() = %#v, want %#v", got, expected)
	}
}

func TestServiceGetCapabilitiesDelegatesToProvider(t *testing.T) {
	expected := domainsystem.CapabilitiesSummary{VMLifecycleSupported: true, Capabilities: []string{"vm-lifecycle"}}
	service := appsystem.NewService(fakeProvider{capabilities: expected})

	got, err := service.GetCapabilities(context.Background())
	if err != nil {
		t.Fatalf("GetCapabilities() error = %v", err)
	}
	if got.VMLifecycleSupported != expected.VMLifecycleSupported {
		t.Fatalf("GetCapabilities().VMLifecycleSupported = %v, want %v", got.VMLifecycleSupported, expected.VMLifecycleSupported)
	}
	if len(got.Capabilities) != 1 || got.Capabilities[0] != "vm-lifecycle" {
		t.Fatalf("GetCapabilities().Capabilities = %#v, want %#v", got.Capabilities, expected.Capabilities)
	}
}

package system_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appsystem "github.com/popododo0720/anarchy/internal/application/system"
	domainsystem "github.com/popododo0720/anarchy/internal/domain/system"
	httpsystem "github.com/popododo0720/anarchy/internal/transport/http/system"
)

func TestHealthHandlerReturnsStructuredSummary(t *testing.T) {
	service := appsystem.NewService(fakeHTTPProvider{})
	handler := httpsystem.NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/health", nil)
	res := httptest.NewRecorder()

	handler.RegisterRoutes(http.NewServeMux())
	handler.Health(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if body["status"] != string(domainsystem.StatusHealthy) {
		t.Fatalf("status field = %v, want %q", body["status"], domainsystem.StatusHealthy)
	}
	if body["readyNodes"] != float64(3) {
		t.Fatalf("readyNodes = %v, want 3", body["readyNodes"])
	}
}

func TestVersionHandlerReturnsStructuredSummary(t *testing.T) {
	service := appsystem.NewService(fakeHTTPProvider{})
	handler := httpsystem.NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/version", nil)
	res := httptest.NewRecorder()

	handler.Version(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body["apiVersion"] != "v1" {
		t.Fatalf("apiVersion = %v, want v1", body["apiVersion"])
	}
}

func TestCapabilitiesHandlerReturnsStructuredSummary(t *testing.T) {
	service := appsystem.NewService(fakeHTTPProvider{})
	handler := httpsystem.NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/capabilities", nil)
	res := httptest.NewRecorder()

	handler.Capabilities(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body["vmLifecycleSupported"] != true {
		t.Fatalf("vmLifecycleSupported = %v, want true", body["vmLifecycleSupported"])
	}
}

type fakeHTTPProvider struct{}

func (fakeHTTPProvider) GetHealth(_ context.Context) (domainsystem.HealthSummary, error) {
	return domainsystem.HealthSummary{
		Status:              domainsystem.StatusHealthy,
		APIReachable:        true,
		KubernetesReachable: true,
		KubeVirtInstalled:   true,
		KubeVirtReady:       true,
		CDIInstalled:        true,
		CDIReady:            true,
		TotalNodes:          3,
		ReadyNodes:          3,
	}, nil
}

func (fakeHTTPProvider) GetVersion(_ context.Context) (domainsystem.VersionSummary, error) {
	return domainsystem.VersionSummary{APIVersion: "v1", ServerVersion: "dev"}, nil
}

func (fakeHTTPProvider) GetCapabilities(_ context.Context) (domainsystem.CapabilitiesSummary, error) {
	return domainsystem.CapabilitiesSummary{VMLifecycleSupported: true, Capabilities: []string{"vm-lifecycle"}}, nil
}

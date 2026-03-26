package diagnose_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appdiag "github.com/popododo0720/anarchy/internal/application/diagnose"
	domaindiag "github.com/popododo0720/anarchy/internal/domain/diagnose"
	httpdiag "github.com/popododo0720/anarchy/internal/transport/http/diagnose"
)

type fakeProvider struct{}

func (fakeProvider) DiagnoseCluster(context.Context) (domaindiag.ClusterReport, error) {
	return domaindiag.ClusterReport{Status: "degraded", Findings: []string{"cdi not ready"}}, nil
}

func (fakeProvider) DiagnoseVM(context.Context, string) (domaindiag.VMReport, error) {
	return domaindiag.VMReport{Name: "testvm", Phase: "Provisioning", Findings: []string{"datavolume phase: WaitForFirstConsumer"}}, nil
}

func TestDiagnoseClusterHandlerReturnsStructuredReport(t *testing.T) {
	handler := httpdiag.NewHandler(appdiag.NewService(fakeProvider{}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/diagnose/cluster", nil)
	res := httptest.NewRecorder()

	handler.DiagnoseCluster(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body["status"] != "degraded" {
		t.Fatalf("body = %#v", body)
	}
}

func TestDiagnoseVMHandlerReturnsStructuredReport(t *testing.T) {
	handler := httpdiag.NewHandler(appdiag.NewService(fakeProvider{}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/diagnose/vms/testvm", nil)
	req.SetPathValue("name", "testvm")
	res := httptest.NewRecorder()

	handler.DiagnoseVM(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body["name"] != "testvm" {
		t.Fatalf("body = %#v", body)
	}
}

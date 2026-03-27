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

func (fakeProvider) DiagnosePublicIP(context.Context, string) (domaindiag.PublicIPReport, error) {
	return domaindiag.PublicIPReport{Name: "fip-01", Status: "pending", Reason: "ovnfip_missing", Code: "public_ip_not_realized", Findings: []string{"floating ip rule not realized yet"}}, nil
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

func TestDiagnosePublicIPHandlerReturnsStructuredReport(t *testing.T) {
	handler := httpdiag.NewHandler(appdiag.NewService(fakeProvider{}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/diagnose/public-ips/fip-01", nil)
	req.SetPathValue("name", "fip-01")
	res := httptest.NewRecorder()

	handler.DiagnosePublicIP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body["name"] != "fip-01" || body["status"] != "pending" || body["reason"] != "ovnfip_missing" || body["code"] != "public_ip_not_realized" {
		t.Fatalf("body = %#v", body)
	}
}

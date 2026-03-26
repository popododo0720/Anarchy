package vm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appvm "github.com/popododo0720/anarchy/internal/application/vm"
	domainvm "github.com/popododo0720/anarchy/internal/domain/vm"
	httpvm "github.com/popododo0720/anarchy/internal/transport/http/vm"
)

type fakeHTTPProvider struct{}

func (fakeHTTPProvider) CreateVM(context.Context, domainvm.CreateVMRequest) (domainvm.VMDetail, error) {
	return domainvm.VMDetail{Name: "vm1", Phase: "Running", Image: "ubuntu-24.04", CPU: 2, Memory: "4Gi", Network: "default"}, nil
}
func (fakeHTTPProvider) ListVMs(context.Context) ([]domainvm.VMSummary, error) {
	return []domainvm.VMSummary{{Name: "vm1", Phase: "Running", Image: "ubuntu-24.04", PrivateIP: "10.0.0.10"}}, nil
}
func (fakeHTTPProvider) GetVM(context.Context, string) (domainvm.VMDetail, error) {
	return domainvm.VMDetail{Name: "vm1", Phase: "Running", Image: "ubuntu-24.04", CPU: 2, Memory: "4Gi", Network: "default", PrivateIP: "10.0.0.10"}, nil
}
func (fakeHTTPProvider) StartVM(context.Context, string) error   { return nil }
func (fakeHTTPProvider) StopVM(context.Context, string) error    { return nil }
func (fakeHTTPProvider) RestartVM(context.Context, string) error { return nil }
func (fakeHTTPProvider) DeleteVM(context.Context, string) error  { return nil }

func TestCreateVMHandlerReturnsStructuredDetail(t *testing.T) {
	service := appvm.NewService(fakeHTTPProvider{})
	handler := httpvm.NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/vms", strings.NewReader(`{"name":"vm1","image":"ubuntu-24.04","cpu":2,"memory":"4Gi","network":"default","subnetRef":"tenant-a"}`))
	res := httptest.NewRecorder()

	handler.CreateVM(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusCreated)
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["name"] != "vm1" {
		t.Fatalf("name = %v, want vm1", body["name"])
	}
}

func TestListVMsHandlerReturnsStructuredSummary(t *testing.T) {
	service := appvm.NewService(fakeHTTPProvider{})
	handler := httpvm.NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vms", nil)
	res := httptest.NewRecorder()

	handler.ListVMs(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var body []map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(body) != 1 || body[0]["name"] != "vm1" {
		t.Fatalf("body = %#v, want vm1", body)
	}
}

func TestActionHandlersReturnAccepted(t *testing.T) {
	service := appvm.NewService(fakeHTTPProvider{})
	handler := httpvm.NewHandler(service)
	cases := []struct {
		path string
		fn   func(http.ResponseWriter, *http.Request)
	}{
		{"/api/v1/vms/vm1/start", handler.StartVM},
		{"/api/v1/vms/vm1/stop", handler.StopVM},
		{"/api/v1/vms/vm1/restart", handler.RestartVM},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodPost, tc.path, nil)
		req.SetPathValue("name", "vm1")
		res := httptest.NewRecorder()
		tc.fn(res, req)
		if res.Code != http.StatusAccepted {
			t.Fatalf("%s status = %d, want %d", tc.path, res.Code, http.StatusAccepted)
		}
	}
}

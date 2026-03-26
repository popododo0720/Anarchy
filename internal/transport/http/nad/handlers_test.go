package nad_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appnad "github.com/popododo0720/anarchy/internal/application/nad"
	domainnad "github.com/popododo0720/anarchy/internal/domain/nad"
	httpnad "github.com/popododo0720/anarchy/internal/transport/http/nad"
)

type fakeProvider struct{}

func (fakeProvider) ListNADs(context.Context) ([]domainnad.NADSummary, error) {
	return []domainnad.NADSummary{{Name: "tenant-b-net", Namespace: "anarchy-system", Type: "kube-ovn", Provider: "tenant-b.ovn"}}, nil
}

func (fakeProvider) GetNAD(context.Context, string, string) (domainnad.NADDetail, error) {
	return domainnad.NADDetail{Name: "tenant-b-net", Namespace: "anarchy-system", Type: "kube-ovn", Provider: "tenant-b.ovn", RawConfig: `{"type":"kube-ovn"}`}, nil
}

func (fakeProvider) CreateNAD(context.Context, domainnad.CreateNADRequest) (domainnad.NADDetail, error) {
	return domainnad.NADDetail{Name: "tenant-c-net", Namespace: "anarchy-system", Type: "kube-ovn", Provider: "tenant-c-net.anarchy-system.ovn", RawConfig: `{"type":"kube-ovn"}`}, nil
}

func TestListNADsHandlerReturnsStructuredSummary(t *testing.T) {
	handler := httpnad.NewHandler(appnad.NewService(fakeProvider{}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/nads", nil)
	res := httptest.NewRecorder()

	handler.ListNADs(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var body []map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(body) != 1 || body[0]["name"] != "tenant-b-net" {
		t.Fatalf("body = %#v", body)
	}
}

func TestGetNADHandlerReturnsStructuredDetail(t *testing.T) {
	handler := httpnad.NewHandler(appnad.NewService(fakeProvider{}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/nads/anarchy-system/tenant-b-net", nil)
	req.SetPathValue("namespace", "anarchy-system")
	req.SetPathValue("name", "tenant-b-net")
	res := httptest.NewRecorder()

	handler.GetNAD(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body["provider"] != "tenant-b.ovn" {
		t.Fatalf("body = %#v", body)
	}
}

func TestCreateNADHandlerReturnsStructuredDetail(t *testing.T) {
	handler := httpnad.NewHandler(appnad.NewService(fakeProvider{}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/nads", strings.NewReader(`{"name":"tenant-c-net","namespace":"anarchy-system","type":"kube-ovn","provider":"tenant-c-net.anarchy-system.ovn","serverSocket":"/run/openvswitch/kube-ovn-daemon.sock"}`))
	res := httptest.NewRecorder()

	handler.CreateNAD(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusCreated)
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body["name"] != "tenant-c-net" {
		t.Fatalf("body = %#v", body)
	}
}

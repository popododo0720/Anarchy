package network_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appnetwork "github.com/popododo0720/anarchy/internal/application/network"
	domainnetwork "github.com/popododo0720/anarchy/internal/domain/network"
	httpnetwork "github.com/popododo0720/anarchy/internal/transport/http/network"
)

type fakeProvider struct{}

func (fakeProvider) ListNetworks(context.Context) ([]domainnetwork.NetworkSummary, error) {
	return []domainnetwork.NetworkSummary{{Name: "ovn-cluster", Default: true, DefaultSubnet: "ovn-default"}}, nil
}

func (fakeProvider) GetNetwork(context.Context, string) (domainnetwork.NetworkDetail, error) {
	return domainnetwork.NetworkDetail{Name: "ovn-cluster", Default: true, Router: "ovn-cluster", DefaultSubnet: "ovn-default", Subnets: []string{"ovn-default"}}, nil
}

func (fakeProvider) CreateNetwork(context.Context, domainnetwork.CreateNetworkRequest) (domainnetwork.NetworkDetail, error) {
	return domainnetwork.NetworkDetail{Name: "tenant-c", Default: false, Router: "tenant-c", DefaultSubnet: "tenant-c-subnet", Subnets: []string{"tenant-c-subnet"}}, nil
}

func TestListNetworksHandlerReturnsStructuredSummary(t *testing.T) {
	handler := httpnetwork.NewHandler(appnetwork.NewService(fakeProvider{}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/networks", nil)
	res := httptest.NewRecorder()

	handler.ListNetworks(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var body []map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(body) != 1 || body[0]["name"] != "ovn-cluster" {
		t.Fatalf("body = %#v", body)
	}
}

func TestGetNetworkHandlerReturnsStructuredDetail(t *testing.T) {
	handler := httpnetwork.NewHandler(appnetwork.NewService(fakeProvider{}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/networks/ovn-cluster", nil)
	req.SetPathValue("name", "ovn-cluster")
	res := httptest.NewRecorder()

	handler.GetNetwork(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body["defaultSubnet"] != "ovn-default" {
		t.Fatalf("body = %#v", body)
	}
}

func TestCreateNetworkHandlerReturnsStructuredDetail(t *testing.T) {
	handler := httpnetwork.NewHandler(appnetwork.NewService(fakeProvider{}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/networks", strings.NewReader(`{"name":"tenant-c"}`))
	res := httptest.NewRecorder()

	handler.CreateNetwork(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusCreated)
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body["name"] != "tenant-c" {
		t.Fatalf("body = %#v", body)
	}
}

package subnet_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appsubnet "github.com/popododo0720/anarchy/internal/application/subnet"
	domainsubnet "github.com/popododo0720/anarchy/internal/domain/subnet"
	httpsubnet "github.com/popododo0720/anarchy/internal/transport/http/subnet"
)

type fakeProvider struct{}

func (fakeProvider) ListSubnets(context.Context) ([]domainsubnet.SubnetSummary, error) {
	return []domainsubnet.SubnetSummary{{Name: "ovn-default", CIDR: "10.16.0.0/16", Gateway: "10.16.0.1", Protocol: "IPv4", Network: "ovn-cluster"}}, nil
}

func (fakeProvider) GetSubnet(context.Context, string) (domainsubnet.SubnetDetail, error) {
	return domainsubnet.SubnetDetail{Name: "ovn-default", CIDR: "10.16.0.0/16", Gateway: "10.16.0.1", Protocol: "IPv4", Provider: "ovn", Network: "ovn-cluster", Namespaces: []string{"anarchy-system"}}, nil
}

func (fakeProvider) CreateSubnet(context.Context, domainsubnet.CreateSubnetRequest) (domainsubnet.SubnetDetail, error) {
	return domainsubnet.SubnetDetail{Name: "tenant-c", CIDR: "10.18.0.0/24", Gateway: "10.18.0.1", Protocol: "IPv4", Provider: "tenant-c-net.anarchy-system.ovn", Network: "ovn-cluster", Namespaces: []string{"anarchy-system"}}, nil
}

func TestListSubnetsHandlerReturnsStructuredSummary(t *testing.T) {
	handler := httpsubnet.NewHandler(appsubnet.NewService(fakeProvider{}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/subnets", nil)
	res := httptest.NewRecorder()

	handler.ListSubnets(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var body []map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(body) != 1 || body[0]["name"] != "ovn-default" {
		t.Fatalf("body = %#v", body)
	}
}

func TestGetSubnetHandlerReturnsStructuredDetail(t *testing.T) {
	handler := httpsubnet.NewHandler(appsubnet.NewService(fakeProvider{}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/subnets/ovn-default", nil)
	req.SetPathValue("name", "ovn-default")
	res := httptest.NewRecorder()

	handler.GetSubnet(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body["network"] != "ovn-cluster" {
		t.Fatalf("body = %#v", body)
	}
}

func TestCreateSubnetHandlerReturnsStructuredDetail(t *testing.T) {
	handler := httpsubnet.NewHandler(appsubnet.NewService(fakeProvider{}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subnets", strings.NewReader(`{"name":"tenant-c","cidr":"10.18.0.0/24","gateway":"10.18.0.1","protocol":"IPv4","provider":"tenant-c-net.anarchy-system.ovn","network":"ovn-cluster","namespaces":["anarchy-system"]}`))
	res := httptest.NewRecorder()

	handler.CreateSubnet(res, req)
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

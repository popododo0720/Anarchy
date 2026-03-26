package publicip_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apppublicip "github.com/popododo0720/anarchy/internal/application/publicip"
	domainpublicip "github.com/popododo0720/anarchy/internal/domain/publicip"
	httppublicip "github.com/popododo0720/anarchy/internal/transport/http/publicip"
)

type fakeProvider struct{}

func (fakeProvider) ListPublicIPs(context.Context) ([]domainpublicip.PublicIPSummary, error) {
	return []domainpublicip.PublicIPSummary{{Name: "fip-01", Address: "203.0.113.10", Attached: true, AttachmentTarget: "vm1:nic0"}}, nil
}

func (fakeProvider) GetPublicIP(context.Context, string) (domainpublicip.PublicIPDetail, error) {
	return domainpublicip.PublicIPDetail{Name: "fip-01", Address: "203.0.113.10", Attached: true, AttachmentTarget: "vm1:nic0", Type: "floating"}, nil
}

func TestListPublicIPsHandlerReturnsStructuredSummary(t *testing.T) {
	handler := httppublicip.NewHandler(apppublicip.NewService(fakeProvider{}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public-ips", nil)
	res := httptest.NewRecorder()

	handler.ListPublicIPs(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var body []map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(body) != 1 || body[0]["name"] != "fip-01" {
		t.Fatalf("body = %#v", body)
	}
}

func TestGetPublicIPHandlerReturnsStructuredDetail(t *testing.T) {
	handler := httppublicip.NewHandler(apppublicip.NewService(fakeProvider{}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public-ips/fip-01", nil)
	req.SetPathValue("name", "fip-01")
	res := httptest.NewRecorder()

	handler.GetPublicIP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body["type"] != "floating" {
		t.Fatalf("body = %#v", body)
	}
}

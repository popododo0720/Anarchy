package image_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appimage "github.com/popododo0720/anarchy/internal/application/image"
	domainimage "github.com/popododo0720/anarchy/internal/domain/image"
	httpimage "github.com/popododo0720/anarchy/internal/transport/http/image"
)

type fakeHTTPProvider struct{}

func (fakeHTTPProvider) ListImages(context.Context) ([]domainimage.ImageSummary, error) {
	return []domainimage.ImageSummary{{
		Name:       "ubuntu-24.04",
		SourceType: "local",
		Ready:      true,
		Size:       "2Gi",
	}}, nil
}

func (fakeHTTPProvider) GetImage(context.Context, string) (domainimage.ImageDetail, error) {
	return domainimage.ImageDetail{
		Name:        "ubuntu-24.04",
		SourceType:  "local",
		Ready:       true,
		Size:        "2Gi",
		Description: "Ubuntu image",
		Tags:        []string{"ubuntu", "24.04"},
	}, nil
}

func TestListImagesHandlerReturnsStructuredSummary(t *testing.T) {
	service := appimage.NewService(fakeHTTPProvider{})
	handler := httpimage.NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images", nil)
	res := httptest.NewRecorder()

	handler.ListImages(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}

	var body []map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(body) != 1 || body[0]["name"] != "ubuntu-24.04" {
		t.Fatalf("body = %#v, want ubuntu-24.04 summary", body)
	}
}

func TestGetImageHandlerReturnsStructuredDetail(t *testing.T) {
	service := appimage.NewService(fakeHTTPProvider{})
	handler := httpimage.NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/images/ubuntu-24.04", nil)
	req.SetPathValue("name", "ubuntu-24.04")
	res := httptest.NewRecorder()

	handler.GetImage(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body["name"] != "ubuntu-24.04" {
		t.Fatalf("name = %v, want ubuntu-24.04", body["name"])
	}
}

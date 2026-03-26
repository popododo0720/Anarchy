package node_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appnode "github.com/popododo0720/anarchy/internal/application/node"
	domainnode "github.com/popododo0720/anarchy/internal/domain/node"
	httpnode "github.com/popododo0720/anarchy/internal/transport/http/node"
)

type fakeHTTPProvider struct{}

func (fakeHTTPProvider) ListNodes(context.Context) ([]domainnode.NodeSummary, error) {
	return []domainnode.NodeSummary{{
		Name:                  "node1",
		Ready:                 true,
		Schedulable:           true,
		VirtualizationCapable: true,
		Class:                 "control-plane",
	}}, nil
}

func (fakeHTTPProvider) GetNode(context.Context, string) (domainnode.NodeDetail, error) {
	return domainnode.NodeDetail{
		Name:                  "node1",
		Ready:                 true,
		Schedulable:           true,
		VirtualizationCapable: true,
		Class:                 "control-plane",
		Capabilities:          []string{"kubevirt", "kube-ovn"},
	}, nil
}

func TestListNodesHandlerReturnsStructuredSummary(t *testing.T) {
	service := appnode.NewService(fakeHTTPProvider{})
	handler := httpnode.NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nodes", nil)
	res := httptest.NewRecorder()

	handler.ListNodes(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}

	var body []map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(body) != 1 || body[0]["name"] != "node1" {
		t.Fatalf("body = %#v, want node1 summary", body)
	}
}

func TestGetNodeHandlerReturnsStructuredDetail(t *testing.T) {
	service := appnode.NewService(fakeHTTPProvider{})
	handler := httpnode.NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nodes/node1", nil)
	req.SetPathValue("name", "node1")
	res := httptest.NewRecorder()

	handler.GetNode(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body["name"] != "node1" {
		t.Fatalf("name = %v, want node1", body["name"])
	}
}

package node_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	clinode "github.com/popododo0720/anarchy/internal/cli/node"
)

func TestRunListPrintsReadableSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/nodes" {
			t.Fatalf("path = %s, want /api/v1/nodes", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"name":"node1","class":"control-plane","ready":true,"schedulable":true,"virtualizationCapable":true}]`))
	}))
	defer server.Close()

	var out bytes.Buffer
	if err := clinode.Run([]string{"list"}, server.URL, server.Client(), &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{"Name: node1", "Class: control-plane", "Virtualization capable: true"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Fatalf("output = %q, want to contain %q", out.String(), want)
		}
	}
}

func TestRunShowPrintsReadableDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/nodes/node1" {
			t.Fatalf("path = %s, want /api/v1/nodes/node1", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"name":"node1","class":"control-plane","ready":true,"schedulable":true,"virtualizationCapable":true,"capabilities":["kubevirt","kube-ovn"]}`))
	}))
	defer server.Close()

	var out bytes.Buffer
	if err := clinode.Run([]string{"show", "node1"}, server.URL, server.Client(), &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{"Name: node1", "Capabilities: kubevirt, kube-ovn"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Fatalf("output = %q, want to contain %q", out.String(), want)
		}
	}
}

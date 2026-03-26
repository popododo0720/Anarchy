package nad_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	clinad "github.com/popododo0720/anarchy/internal/cli/nad"
)

func TestRunListPrintsNADSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/nads" {
			t.Fatalf("path = %s, want /api/v1/nads", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"name":"tenant-b-net","namespace":"anarchy-system","type":"kube-ovn","provider":"tenant-b.ovn"}]`))
	}))
	defer server.Close()

	var out bytes.Buffer
	if err := clinad.Run([]string{"list"}, server.URL, server.Client(), &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{"Name: tenant-b-net", "Namespace: anarchy-system", "Provider: tenant-b.ovn"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Fatalf("output = %q, want to contain %q", out.String(), want)
		}
	}
}

func TestRunShowPrintsNADDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/nads/anarchy-system/tenant-b-net" {
			t.Fatalf("path = %s, want /api/v1/nads/anarchy-system/tenant-b-net", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"name":"tenant-b-net","namespace":"anarchy-system","type":"kube-ovn","provider":"tenant-b.ovn","rawConfig":"{\"type\":\"kube-ovn\"}"}`))
	}))
	defer server.Close()

	var out bytes.Buffer
	if err := clinad.Run([]string{"show", "anarchy-system", "tenant-b-net"}, server.URL, server.Client(), &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{"Name: tenant-b-net", "Type: kube-ovn", "Raw config: {\"type\":\"kube-ovn\"}"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Fatalf("output = %q, want to contain %q", out.String(), want)
		}
	}
}

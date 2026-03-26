package system_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	clisystem "github.com/popododo0720/anarchy/internal/cli/system"
)

func TestRunHealthPrintsReadableSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/system/health" {
			t.Fatalf("path = %s, want /api/v1/system/health", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"healthy","kubernetesReachable":true,"kubevirtReady":true,"cdiReady":true,"readyNodes":3,"totalNodes":3,"warnings":[]}`))
	}))
	defer server.Close()

	var out bytes.Buffer
	err := clisystem.Run([]string{"health"}, server.URL, server.Client(), &out)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	got := out.String()
	for _, want := range []string{"Status: healthy", "Kubernetes reachable: true", "Ready nodes: 3/3"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Fatalf("output = %q, want to contain %q", got, want)
		}
	}
}

func TestRunVersionPrintsReadableSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/system/version" {
			t.Fatalf("path = %s, want /api/v1/system/version", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"apiVersion":"v1","serverVersion":"dev","supportedApiVersions":["v1"]}`))
	}))
	defer server.Close()

	var out bytes.Buffer
	err := clisystem.Run([]string{"version"}, server.URL, server.Client(), &out)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("API version: v1")) {
		t.Fatalf("output = %q, want API version line", out.String())
	}
}

func TestRunCapabilitiesPrintsReadableSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/system/capabilities" {
			t.Fatalf("path = %s, want /api/v1/system/capabilities", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"vmLifecycleSupported":true,"imageInventorySupported":true,"diagnosticsSupported":true,"publicIpSupported":false,"capabilities":["vm-lifecycle","diagnostics"]}`))
	}))
	defer server.Close()

	var out bytes.Buffer
	err := clisystem.Run([]string{"capabilities"}, server.URL, server.Client(), &out)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("VM lifecycle supported: true")) {
		t.Fatalf("output = %q, want VM lifecycle line", out.String())
	}
}

package diagnose_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	clidiag "github.com/popododo0720/anarchy/internal/cli/diagnose"
)

func TestRunClusterPrintsFindings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/diagnose/cluster" {
			t.Fatalf("path = %s, want /api/v1/diagnose/cluster", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"status":"degraded","findings":["cdi not ready","0/1 nodes ready"]}`))
	}))
	defer server.Close()

	var out bytes.Buffer
	if err := clidiag.Run([]string{"cluster"}, server.URL, server.Client(), &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{"Status: degraded", "- cdi not ready", "- 0/1 nodes ready"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Fatalf("output = %q, want to contain %q", out.String(), want)
		}
	}
}

func TestRunVMPrintsFindings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/diagnose/vms/testvm" {
			t.Fatalf("path = %s, want /api/v1/diagnose/vms/testvm", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"name":"testvm","phase":"Provisioning","findings":["datavolume phase: WaitForFirstConsumer"]}`))
	}))
	defer server.Close()

	var out bytes.Buffer
	if err := clidiag.Run([]string{"vm", "testvm"}, server.URL, server.Client(), &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{"Name: testvm", "Phase: Provisioning", "- datavolume phase: WaitForFirstConsumer"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Fatalf("output = %q, want to contain %q", out.String(), want)
		}
	}
}

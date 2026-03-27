package publicip_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	clipublicip "github.com/popododo0720/anarchy/internal/cli/publicip"
)

func TestRunListPrintsPublicIPSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/public-ips" {
			t.Fatalf("path = %s, want /api/v1/public-ips", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"name":"fip-01","address":"203.0.113.10","attached":true,"attachmentTarget":"vm1:nic0"}]`))
	}))
	defer server.Close()

	var out bytes.Buffer
	if err := clipublicip.Run([]string{"list"}, server.URL, server.Client(), &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{"Name: fip-01", "Address: 203.0.113.10", "Target: vm1:nic0"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Fatalf("output = %q, want %q", out.String(), want)
		}
	}
}

func TestRunShowPrintsPublicIPDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/public-ips/fip-01" {
			t.Fatalf("path = %s, want /api/v1/public-ips/fip-01", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"name":"fip-01","address":"203.0.113.10","attached":true,"attachmentTarget":"vm1:nic0","type":"floating"}`))
	}))
	defer server.Close()

	var out bytes.Buffer
	if err := clipublicip.Run([]string{"show", "fip-01"}, server.URL, server.Client(), &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{"Name: fip-01", "Type: floating", "Target: vm1:nic0"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Fatalf("output = %q, want %q", out.String(), want)
		}
	}
}

func TestRunAttachCallsEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/public-ips/fip-01/attach" {
			t.Fatalf("path = %s, want /api/v1/public-ips/fip-01/attach", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"name":"fip-01","address":"203.0.113.10","attached":true,"attachmentTarget":"vm1:nic1","type":"floating"}`))
	}))
	defer server.Close()

	var out bytes.Buffer
	if err := clipublicip.Run([]string{"attach", "fip-01", "vm1:nic1"}, server.URL, server.Client(), &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("Attached public IP: fip-01 -> vm1:nic1")) {
		t.Fatalf("output = %q", out.String())
	}
}

func TestRunAttachRejectsInvalidTargetBeforeCallingAPI(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	var out bytes.Buffer
	err := clipublicip.Run([]string{"attach", "fip-01", "vm1"}, server.URL, server.Client(), &out)
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}
	if called {
		t.Fatal("API was called for invalid attachment target")
	}
}

func TestRunDetachCallsEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/public-ips/fip-01/detach" {
			t.Fatalf("path = %s, want /api/v1/public-ips/fip-01/detach", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"name":"fip-01","address":"203.0.113.10","attached":false,"attachmentTarget":"","type":"floating"}`))
	}))
	defer server.Close()

	var out bytes.Buffer
	if err := clipublicip.Run([]string{"detach", "fip-01"}, server.URL, server.Client(), &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("Detached public IP: fip-01")) {
		t.Fatalf("output = %q", out.String())
	}
}

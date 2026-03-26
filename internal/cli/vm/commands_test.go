package vm_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	clivm "github.com/popododo0720/anarchy/internal/cli/vm"
)

func TestRunListPrintsReadableSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/vms" {
			t.Fatalf("path = %s, want /api/v1/vms", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"name":"vm1","phase":"Running","image":"ubuntu-24.04","network":"tenant-a","subnetRef":"tenant-a","privateIp":"10.0.0.10","networkAttachments":[{"name":"nic0","network":"tenant-a","subnetRef":"tenant-a","primary":true}]}]`))
	}))
	defer server.Close()

	var out bytes.Buffer
	if err := clivm.Run([]string{"list"}, server.URL, server.Client(), &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{"Name: vm1", "Phase: Running", "Network: tenant-a", "Subnet: tenant-a", "Private IP: 10.0.0.10"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Fatalf("output = %q, want %q", out.String(), want)
		}
	}
}

func TestRunShowPrintsReadableDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/vms/vm1" {
			t.Fatalf("path = %s, want /api/v1/vms/vm1", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"name":"vm1","phase":"Running","image":"ubuntu-24.04","cpu":2,"memory":"4Gi","network":"tenant-a","subnetRef":"tenant-a","privateIp":"10.0.0.10","networkAttachments":[{"name":"nic0","network":"tenant-a","subnetRef":"tenant-a","primary":true}]}`))
	}))
	defer server.Close()

	var out bytes.Buffer
	if err := clivm.Run([]string{"show", "vm1"}, server.URL, server.Client(), &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{"Name: vm1", "Image: ubuntu-24.04", "Network: tenant-a", "Subnet: tenant-a", "Attachments: nic0(tenant-a/tenant-a, primary=true)"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Fatalf("output = %q, want %q", out.String(), want)
		}
	}
}

func TestRunCreateSendsNetworkAttachments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/vms" {
			t.Fatalf("path = %s, want /api/v1/vms", r.URL.Path)
		}
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		for _, want := range []string{"\"network\":\"default\"", "\"subnetRef\":\"tenant-a\"", "\"networkAttachments\":[{"} {
			if !bytes.Contains(buf.Bytes(), []byte(want)) {
				t.Fatalf("body = %s, want %s", buf.String(), want)
			}
		}
		_, _ = w.Write([]byte(`{"name":"vm1","phase":"Provisioning","image":"ubuntu-24.04","cpu":2,"memory":"4Gi","network":"default","privateIp":""}`))
	}))
	defer server.Close()

	var out bytes.Buffer
	if err := clivm.Run([]string{"create", "vm1", "ubuntu-24.04", "2", "4Gi", "default", "tenant-a"}, server.URL, server.Client(), &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestRunActionCallsEndpoint(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Path != "/api/v1/vms/vm1/start" {
			t.Fatalf("path = %s, want /api/v1/vms/vm1/start", r.URL.Path)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	var out bytes.Buffer
	if err := clivm.Run([]string{"start", "vm1"}, server.URL, server.Client(), &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !called {
		t.Fatal("expected endpoint to be called")
	}
}

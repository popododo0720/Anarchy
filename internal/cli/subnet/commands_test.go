package subnet_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	clisubnet "github.com/popododo0720/anarchy/internal/cli/subnet"
)

func TestRunListPrintsSubnetSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/subnets" {
			t.Fatalf("path = %s, want /api/v1/subnets", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"name":"ovn-default","cidr":"10.16.0.0/16","gateway":"10.16.0.1","protocol":"IPv4","network":"ovn-cluster"}]`))
	}))
	defer server.Close()

	var out bytes.Buffer
	if err := clisubnet.Run([]string{"list"}, server.URL, server.Client(), &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{"Name: ovn-default", "CIDR: 10.16.0.0/16", "Network: ovn-cluster"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Fatalf("output = %q, want to contain %q", out.String(), want)
		}
	}
}

func TestRunShowPrintsSubnetDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/subnets/ovn-default" {
			t.Fatalf("path = %s, want /api/v1/subnets/ovn-default", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"name":"ovn-default","cidr":"10.16.0.0/16","gateway":"10.16.0.1","protocol":"IPv4","provider":"ovn","network":"ovn-cluster","namespaces":["anarchy-system"]}`))
	}))
	defer server.Close()

	var out bytes.Buffer
	if err := clisubnet.Run([]string{"show", "ovn-default"}, server.URL, server.Client(), &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{"Name: ovn-default", "Provider: ovn", "Namespaces: anarchy-system"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Fatalf("output = %q, want to contain %q", out.String(), want)
		}
	}
}

func TestRunCreatePostsSubnetRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/subnets" {
			t.Fatalf("path = %s, want /api/v1/subnets", r.URL.Path)
		}
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		for _, want := range []string{"\"name\":\"tenant-c\"", "\"cidr\":\"10.18.0.0/24\"", "\"provider\":\"tenant-c-net.anarchy-system.ovn\""} {
			if !bytes.Contains(buf.Bytes(), []byte(want)) {
				t.Fatalf("body = %s, want %s", buf.String(), want)
			}
		}
		_, _ = w.Write([]byte(`{"name":"tenant-c","cidr":"10.18.0.0/24","gateway":"10.18.0.1","protocol":"IPv4","provider":"tenant-c-net.anarchy-system.ovn","network":"ovn-cluster","namespaces":["anarchy-system"]}`))
	}))
	defer server.Close()

	var out bytes.Buffer
	if err := clisubnet.Run([]string{"create", "tenant-c", "10.18.0.0/24", "10.18.0.1", "IPv4", "tenant-c-net.anarchy-system.ovn", "ovn-cluster", "anarchy-system"}, server.URL, server.Client(), &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("Created subnet: tenant-c")) {
		t.Fatalf("output = %q", out.String())
	}
}

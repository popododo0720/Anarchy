package network_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	clinetwork "github.com/popododo0720/anarchy/internal/cli/network"
)

func TestRunListPrintsNetworkSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/networks" {
			t.Fatalf("path = %s, want /api/v1/networks", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"name":"ovn-cluster","default":true,"defaultSubnet":"ovn-default"}]`))
	}))
	defer server.Close()

	var out bytes.Buffer
	if err := clinetwork.Run([]string{"list"}, server.URL, server.Client(), &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{"Name: ovn-cluster", "Default: true", "Default subnet: ovn-default"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Fatalf("output = %q, want to contain %q", out.String(), want)
		}
	}
}

func TestRunShowPrintsNetworkDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/networks/ovn-cluster" {
			t.Fatalf("path = %s, want /api/v1/networks/ovn-cluster", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"name":"ovn-cluster","default":true,"router":"ovn-cluster","defaultSubnet":"ovn-default","subnets":["ovn-default","join"]}`))
	}))
	defer server.Close()

	var out bytes.Buffer
	if err := clinetwork.Run([]string{"show", "ovn-cluster"}, server.URL, server.Client(), &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{"Name: ovn-cluster", "Router: ovn-cluster", "Subnets: ovn-default, join"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Fatalf("output = %q, want to contain %q", out.String(), want)
		}
	}
}

func TestRunCreatePostsNetworkRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/networks" {
			t.Fatalf("path = %s, want /api/v1/networks", r.URL.Path)
		}
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		if !bytes.Contains(buf.Bytes(), []byte(`"name":"tenant-c"`)) {
			t.Fatalf("body = %s", buf.String())
		}
		_, _ = w.Write([]byte(`{"name":"tenant-c","default":false,"router":"tenant-c","defaultSubnet":"tenant-c-subnet","subnets":["tenant-c-subnet"]}`))
	}))
	defer server.Close()

	var out bytes.Buffer
	if err := clinetwork.Run([]string{"create", "tenant-c"}, server.URL, server.Client(), &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("Created network: tenant-c")) {
		t.Fatalf("output = %q", out.String())
	}
}

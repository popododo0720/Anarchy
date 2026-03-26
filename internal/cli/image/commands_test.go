package image_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	cliimage "github.com/popododo0720/anarchy/internal/cli/image"
)

func TestRunListPrintsReadableSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/images" {
			t.Fatalf("path = %s, want /api/v1/images", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"name":"ubuntu-24.04","sourceType":"local","ready":true,"size":"2Gi"}]`))
	}))
	defer server.Close()

	var out bytes.Buffer
	if err := cliimage.Run([]string{"list"}, server.URL, server.Client(), &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{"Name: ubuntu-24.04", "Source type: local", "Ready: true"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Fatalf("output = %q, want to contain %q", out.String(), want)
		}
	}
}

func TestRunShowPrintsReadableDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/images/ubuntu-24.04" {
			t.Fatalf("path = %s, want /api/v1/images/ubuntu-24.04", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"name":"ubuntu-24.04","sourceType":"local","ready":true,"size":"2Gi","description":"Ubuntu image","tags":["ubuntu","24.04"]}`))
	}))
	defer server.Close()

	var out bytes.Buffer
	if err := cliimage.Run([]string{"show", "ubuntu-24.04"}, server.URL, server.Client(), &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{"Name: ubuntu-24.04", "Description: Ubuntu image", "Tags: ubuntu, 24.04"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Fatalf("output = %q, want to contain %q", out.String(), want)
		}
	}
}

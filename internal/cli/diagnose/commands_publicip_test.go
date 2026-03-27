package diagnose_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	clidiag "github.com/popododo0720/anarchy/internal/cli/diagnose"
)

func TestRunPublicIPPrintsReport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/diagnose/public-ips/fip-01" {
			t.Fatalf("path = %s, want /api/v1/diagnose/public-ips/fip-01", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"name":"fip-01","status":"pending","findings":["floating ip rule not realized yet"]}`))
	}))
	defer server.Close()

	var out bytes.Buffer
	if err := clidiag.Run([]string{"publicip", "fip-01"}, server.URL, server.Client(), &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{"Name: fip-01", "Status: pending", "- floating ip rule not realized yet"} {
		if !bytes.Contains(out.Bytes(), []byte(want)) {
			t.Fatalf("output = %q, want %q", out.String(), want)
		}
	}
}

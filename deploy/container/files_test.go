package container

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDockerfileExistsForHelmDeploymentImage(t *testing.T) {
	path := filepath.Join("..", "..", "Dockerfile")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Dockerfile: %v", err)
	}
	text := string(content)
	for _, want := range []string{"FROM golang:", "apk add --no-cache ca-certificates kubectl", "anarchy-api", "ENTRYPOINT"} {
		if !strings.Contains(text, want) {
			t.Fatalf("Dockerfile missing %q\n%s", want, text)
		}
	}
}

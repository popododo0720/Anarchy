package lab

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func TestLabGitOpsIncludesKustomizationAndValues(t *testing.T) {
	base := "."
	for _, name := range []string{"kustomization.yaml", "values.yaml", "gitrepository.yaml", "namespace.yaml", "helmrelease.yaml"} {
		if _, err := os.Stat(filepath.Join(base, name)); err != nil {
			t.Fatalf("expected %s to exist: %v", name, err)
		}
	}

	kustomization := readFile(t, filepath.Join(base, "kustomization.yaml"))
	for _, want := range []string{"namespace.yaml", "gitrepository.yaml", "helmrelease.yaml", "configMapGenerator:", "anarchy-values", "values.yaml"} {
		if !strings.Contains(kustomization, want) {
			t.Fatalf("kustomization.yaml missing %q\n%s", want, kustomization)
		}
	}
}

func TestLabHelmReleaseUsesValuesFromConfigMap(t *testing.T) {
	helmRelease := readFile(t, "helmrelease.yaml")
	for _, want := range []string{"kind: HelmRelease", "valuesFrom:", "kind: ConfigMap", "name: anarchy-values", "valuesKey: values.yaml"} {
		if !strings.Contains(helmRelease, want) {
			t.Fatalf("helmrelease.yaml missing %q\n%s", want, helmRelease)
		}
	}
}

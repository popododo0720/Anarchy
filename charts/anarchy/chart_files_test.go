package anarchy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readChartFile(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(".", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(content)
}

func TestChartIncludesServiceAccountAndClusterRBACTemplates(t *testing.T) {
	for _, name := range []string{"templates/serviceaccount.yaml", "templates/clusterrole.yaml", "templates/clusterrolebinding.yaml"} {
		if _, err := os.Stat(filepath.Join(".", name)); err != nil {
			t.Fatalf("expected %s to exist: %v", name, err)
		}
	}
	clusterRole := readChartFile(t, "templates/clusterrole.yaml")
	for _, want := range []string{"virtualmachines", "datavolumes", "\"create\"", "\"update\"", "\"patch\"", "\"delete\""} {
		if !strings.Contains(clusterRole, want) {
			t.Fatalf("clusterrole.yaml missing %q\n%s", want, clusterRole)
		}
	}
}

func TestChartValuesAndDeploymentSupportNamespaceConfig(t *testing.T) {
	values := readChartFile(t, "values.yaml")
	deployment := readChartFile(t, "templates/deployment.yaml")
	for _, want := range []string{"serviceAccount:", "config:", "namespace:", "nodeSelector:", "imagePullSecrets:"} {
		if !strings.Contains(values, want) {
			t.Fatalf("values.yaml missing %q\n%s", want, values)
		}
	}
	for _, want := range []string{"serviceAccountName:", "ANARCHY_NAMESPACE", ".Values.config.namespace", "with .Values.nodeSelector", "with .Values.imagePullSecrets", "timeoutSeconds: 5", "failureThreshold: 6"} {
		if !strings.Contains(deployment, want) {
			t.Fatalf("deployment.yaml missing %q\n%s", want, deployment)
		}
	}
}

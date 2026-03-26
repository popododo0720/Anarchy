package exec_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	kexec "github.com/popododo0720/anarchy/internal/adapters/kubernetes/exec"
)

func TestCommandRunnerSetsStableKubectlEnv(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "printenv.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho KUBECTL_KUBERC=$KUBECTL_KUBERC\necho HOME=$HOME\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	runner := kexec.NewCommandRunner()
	out, err := runner.Run(context.Background(), script)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{"KUBECTL_KUBERC=false", "HOME="} {
		if !strings.Contains(out, want) {
			t.Fatalf("output = %q, want to contain %q", out, want)
		}
	}
}

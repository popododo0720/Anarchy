package kubernetes_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	kubeimage "github.com/popododo0720/anarchy/internal/adapters/image/kubernetes"
)

type fakeRunner struct {
	responses map[string]string
	errors    map[string]error
	calls     []string
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	key := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, key)
	if err, ok := f.errors[key]; ok {
		return "", err
	}
	if out, ok := f.responses[key]; ok {
		return out, nil
	}
	return "", errors.New("unexpected command: " + key)
}

func TestListImagesParsesDataSourcesAndPVCSize(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl -n anarchy-system get datasources -o json":      `{"items":[{"metadata":{"name":"ubuntu-24.04","annotations":{"anarchy.io/description":"Ubuntu 24.04 cloud image","anarchy.io/tags":"ubuntu,24.04"}},"spec":{"source":{"pvc":{"name":"ubuntu-24.04","namespace":"anarchy-system"}}},"status":{"conditions":[{"type":"Ready","status":"True"}],"source":{"pvc":{"name":"ubuntu-24.04","namespace":"anarchy-system"}}}}]}`,
		"kubectl -n anarchy-system get pvc ubuntu-24.04 -o json": `{"status":{"capacity":{"storage":"10Gi"}}}`,
	}}
	provider := kubeimage.NewProvider(runner, "anarchy-system")

	got, err := provider.ListImages(context.Background())
	if err != nil {
		t.Fatalf("ListImages() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 image, got %#v", got)
	}
	if got[0].Name != "ubuntu-24.04" || got[0].SourceType != "pvc" || !got[0].Ready || got[0].Size != "10Gi" {
		t.Fatalf("ListImages() = %#v", got)
	}
}

func TestGetImageReturnsStructuredDetail(t *testing.T) {
	runner := &fakeRunner{responses: map[string]string{
		"kubectl -n anarchy-system get datasource ubuntu-24.04 -o json": `{"metadata":{"name":"ubuntu-24.04","annotations":{"anarchy.io/description":"Ubuntu 24.04 cloud image","anarchy.io/tags":"ubuntu,24.04"}},"spec":{"source":{"pvc":{"name":"ubuntu-24.04","namespace":"anarchy-system"}}},"status":{"conditions":[{"type":"Ready","status":"True"}],"source":{"pvc":{"name":"ubuntu-24.04","namespace":"anarchy-system"}}}}`,
		"kubectl -n anarchy-system get pvc ubuntu-24.04 -o json":        `{"status":{"capacity":{"storage":"10Gi"}}}`,
	}}
	provider := kubeimage.NewProvider(runner, "anarchy-system")

	got, err := provider.GetImage(context.Background(), "ubuntu-24.04")
	if err != nil {
		t.Fatalf("GetImage() error = %v", err)
	}
	if got.Name != "ubuntu-24.04" || got.SourceType != "pvc" || got.Size != "10Gi" {
		t.Fatalf("GetImage() = %#v", got)
	}
	if got.Description != "Ubuntu 24.04 cloud image" {
		t.Fatalf("description = %q", got.Description)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "ubuntu" || got.Tags[1] != "24.04" {
		t.Fatalf("tags = %#v", got.Tags)
	}
}

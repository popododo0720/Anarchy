package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	kexec "github.com/popododo0720/anarchy/internal/adapters/kubernetes/exec"
	domainnad "github.com/popododo0720/anarchy/internal/domain/nad"
)

type Provider struct {
	runner kexec.Runner
}

func NewProvider(runner kexec.Runner) Provider {
	return Provider{runner: runner}
}

type nadListResponse struct {
	Items []nadItem `json:"items"`
}

type nadItem struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Spec struct {
		Config string `json:"config"`
	} `json:"spec"`
}

type configPayload struct {
	Type     string `json:"type"`
	Provider string `json:"provider"`
}

func (p Provider) ListNADs(ctx context.Context) ([]domainnad.NADSummary, error) {
	out, err := p.runner.Run(ctx, "kubectl", "get", "network-attachment-definitions.k8s.cni.cncf.io", "-A", "-o", "json")
	if err != nil {
		return nil, err
	}
	var payload nadListResponse
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		return nil, err
	}
	items := make([]domainnad.NADSummary, 0, len(payload.Items))
	for _, item := range payload.Items {
		cfg := parseConfig(item.Spec.Config)
		items = append(items, domainnad.NADSummary{Name: item.Metadata.Name, Namespace: item.Metadata.Namespace, Type: cfg.Type, Provider: cfg.Provider})
	}
	return items, nil
}

func (p Provider) GetNAD(ctx context.Context, namespace, name string) (domainnad.NADDetail, error) {
	out, err := p.runner.Run(ctx, "kubectl", "get", "network-attachment-definition.k8s.cni.cncf.io", name, "-n", namespace, "-o", "json")
	if err != nil {
		return domainnad.NADDetail{}, err
	}
	var item nadItem
	if err := json.Unmarshal([]byte(out), &item); err != nil {
		return domainnad.NADDetail{}, err
	}
	cfg := parseConfig(item.Spec.Config)
	return domainnad.NADDetail{Name: item.Metadata.Name, Namespace: item.Metadata.Namespace, Type: cfg.Type, Provider: cfg.Provider, RawConfig: item.Spec.Config}, nil
}

func (p Provider) CreateNAD(ctx context.Context, req domainnad.CreateNADRequest) (domainnad.NADDetail, error) {
	manifest, err := p.writeManifest(req)
	if err != nil {
		return domainnad.NADDetail{}, err
	}
	defer os.Remove(manifest)
	if _, err := p.runner.Run(ctx, "kubectl", "apply", "-f", manifest); err != nil {
		return domainnad.NADDetail{}, err
	}
	return p.GetNAD(ctx, req.Namespace, req.Name)
}

func (p Provider) writeManifest(req domainnad.CreateNADRequest) (string, error) {
	file, err := os.CreateTemp("", "anarchy-nad-*.yaml")
	if err != nil {
		return "", err
	}
	manifest := fmt.Sprintf(`apiVersion: k8s.cni.cncf.io/v1
kind: NetworkAttachmentDefinition
metadata:
  name: %s
  namespace: %s
spec:
  config: |
    {
      "cniVersion": "0.3.1",
      "type": "%s",
      "provider": "%s",
      "server_socket": "%s"
    }
`, req.Name, req.Namespace, req.Type, req.Provider, req.ServerSocket)
	if _, err := file.WriteString(manifest); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return file.Name(), nil
}

func parseConfig(raw string) configPayload {
	var cfg configPayload
	_ = json.Unmarshal([]byte(raw), &cfg)
	return cfg
}

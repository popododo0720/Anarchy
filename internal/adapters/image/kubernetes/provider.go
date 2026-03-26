package kubernetes

import (
	"context"
	"encoding/json"
	"strings"

	kexec "github.com/popododo0720/anarchy/internal/adapters/kubernetes/exec"
	domainimage "github.com/popododo0720/anarchy/internal/domain/image"
)

type Provider struct {
	runner    kexec.Runner
	namespace string
}

func NewProvider(runner kexec.Runner, namespace string) Provider {
	if namespace == "" {
		namespace = "anarchy-system"
	}
	return Provider{runner: runner, namespace: namespace}
}

type dataSourceListResponse struct {
	Items []dataSource `json:"items"`
}

type dataSource struct {
	Metadata struct {
		Name        string            `json:"name"`
		Annotations map[string]string `json:"annotations"`
	} `json:"metadata"`
	Spec struct {
		Source cdiSource `json:"source"`
	} `json:"spec"`
	Status struct {
		Conditions []struct {
			Type   string `json:"type"`
			Status string `json:"status"`
		} `json:"conditions"`
		Source cdiSource `json:"source"`
	} `json:"status"`
}

type cdiSource struct {
	PVC   *pvcRef        `json:"pvc,omitempty"`
	HTTP  *httpRef       `json:"http,omitempty"`
	Blank map[string]any `json:"blank,omitempty"`
}

type pvcRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type httpRef struct {
	URL string `json:"url"`
}

type pvcResponse struct {
	Status struct {
		Capacity struct {
			Storage string `json:"storage"`
		} `json:"capacity"`
	} `json:"status"`
	Spec struct {
		Resources struct {
			Requests struct {
				Storage string `json:"storage"`
			} `json:"requests"`
		} `json:"resources"`
	} `json:"spec"`
}

func (p Provider) ListImages(ctx context.Context) ([]domainimage.ImageSummary, error) {
	out, err := p.runner.Run(ctx, "kubectl", "-n", p.namespace, "get", "datasources", "-o", "json")
	if err != nil {
		return nil, err
	}
	var payload dataSourceListResponse
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		return nil, err
	}
	items := make([]domainimage.ImageSummary, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, domainimage.ImageSummary{
			Name:       item.Metadata.Name,
			SourceType: sourceType(item),
			Ready:      readyCondition(item.Status.Conditions),
			Size:       p.resolveSize(ctx, item),
		})
	}
	return items, nil
}

func (p Provider) GetImage(ctx context.Context, name string) (domainimage.ImageDetail, error) {
	out, err := p.runner.Run(ctx, "kubectl", "-n", p.namespace, "get", "datasource", name, "-o", "json")
	if err != nil {
		return domainimage.ImageDetail{}, err
	}
	var item dataSource
	if err := json.Unmarshal([]byte(out), &item); err != nil {
		return domainimage.ImageDetail{}, err
	}
	annotations := item.Metadata.Annotations
	return domainimage.ImageDetail{
		Name:        item.Metadata.Name,
		SourceType:  sourceType(item),
		Ready:       readyCondition(item.Status.Conditions),
		Size:        p.resolveSize(ctx, item),
		Description: annotations["anarchy.io/description"],
		Tags:        splitCSV(annotations["anarchy.io/tags"]),
	}, nil
}

func (p Provider) resolveSize(ctx context.Context, item dataSource) string {
	ref := pvcFor(item)
	if ref == nil || ref.Name == "" {
		return ""
	}
	namespace := ref.Namespace
	if namespace == "" {
		namespace = p.namespace
	}
	out, err := p.runner.Run(ctx, "kubectl", "-n", namespace, "get", "pvc", ref.Name, "-o", "json")
	if err != nil {
		return ""
	}
	var pvc pvcResponse
	if err := json.Unmarshal([]byte(out), &pvc); err != nil {
		return ""
	}
	if pvc.Status.Capacity.Storage != "" {
		return pvc.Status.Capacity.Storage
	}
	return pvc.Spec.Resources.Requests.Storage
}

func pvcFor(item dataSource) *pvcRef {
	if item.Status.Source.PVC != nil {
		return item.Status.Source.PVC
	}
	if item.Spec.Source.PVC != nil {
		return item.Spec.Source.PVC
	}
	return nil
}

func sourceType(item dataSource) string {
	source := item.Status.Source
	if source.PVC == nil && source.HTTP == nil && source.Blank == nil {
		source = item.Spec.Source
	}
	switch {
	case source.PVC != nil:
		return "pvc"
	case source.HTTP != nil:
		return "http"
	case source.Blank != nil:
		return "blank"
	default:
		return "unknown"
	}
}

func readyCondition(conditions []struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}) bool {
	for _, condition := range conditions {
		if condition.Type == "Ready" {
			return strings.EqualFold(condition.Status, "true")
		}
	}
	return false
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

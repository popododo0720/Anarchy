package static

import (
	"context"
	"fmt"

	domainnode "github.com/popododo0720/anarchy/internal/domain/node"
)

type Provider struct{}

func NewProvider() Provider {
	return Provider{}
}

func (Provider) ListNodes(context.Context) ([]domainnode.NodeSummary, error) {
	return []domainnode.NodeSummary{
		{
			Name:                  "node1",
			Class:                 "control-plane",
			Ready:                 true,
			Schedulable:           true,
			VirtualizationCapable: true,
		},
	}, nil
}

func (Provider) GetNode(ctx context.Context, name string) (domainnode.NodeDetail, error) {
	nodes, err := Provider{}.ListNodes(ctx)
	if err != nil {
		return domainnode.NodeDetail{}, err
	}
	for _, node := range nodes {
		if node.Name == name {
			return domainnode.NodeDetail{
				Name:                  node.Name,
				Class:                 node.Class,
				Ready:                 node.Ready,
				Schedulable:           node.Schedulable,
				VirtualizationCapable: node.VirtualizationCapable,
				Capabilities:          []string{"kubevirt", "kube-ovn"},
			}, nil
		}
	}
	return domainnode.NodeDetail{}, fmt.Errorf("node not found: %s", name)
}

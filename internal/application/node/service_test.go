package node_test

import (
	"context"
	"errors"
	"testing"

	appnode "github.com/popododo0720/anarchy/internal/application/node"
	domainnode "github.com/popododo0720/anarchy/internal/domain/node"
	portnode "github.com/popododo0720/anarchy/internal/ports/node"
)

type fakeProvider struct {
	nodes   []domainnode.NodeSummary
	node    domainnode.NodeDetail
	listErr error
	getErr  error
}

func (f fakeProvider) ListNodes(context.Context) ([]domainnode.NodeSummary, error) {
	return f.nodes, f.listErr
}

func (f fakeProvider) GetNode(context.Context, string) (domainnode.NodeDetail, error) {
	return f.node, f.getErr
}

var _ portnode.Provider = fakeProvider{}

func TestServiceListNodesDelegatesToProvider(t *testing.T) {
	expected := []domainnode.NodeSummary{{Name: "node1", Ready: true, VirtualizationCapable: true}}
	service := appnode.NewService(fakeProvider{nodes: expected})

	got, err := service.ListNodes(context.Background())
	if err != nil {
		t.Fatalf("ListNodes() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "node1" {
		t.Fatalf("ListNodes() = %#v, want %#v", got, expected)
	}
}

func TestServiceGetNodeDelegatesToProvider(t *testing.T) {
	expected := domainnode.NodeDetail{Name: "node1", Ready: true, VirtualizationCapable: true}
	service := appnode.NewService(fakeProvider{node: expected})

	got, err := service.GetNode(context.Background(), "node1")
	if err != nil {
		t.Fatalf("GetNode() error = %v", err)
	}
	if got.Name != expected.Name || got.VirtualizationCapable != expected.VirtualizationCapable {
		t.Fatalf("GetNode() = %#v, want %#v", got, expected)
	}
}

func TestServiceListNodesReturnsProviderError(t *testing.T) {
	expectedErr := errors.New("boom")
	service := appnode.NewService(fakeProvider{listErr: expectedErr})

	_, err := service.ListNodes(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("ListNodes() error = %v, want %v", err, expectedErr)
	}
}

package static

import (
	"context"
	"fmt"

	domainvm "github.com/popododo0720/anarchy/internal/domain/vm"
)

type Provider struct{}

func NewProvider() Provider { return Provider{} }

func (Provider) CreateVM(_ context.Context, req domainvm.CreateVMRequest) (domainvm.VMDetail, error) {
	return domainvm.VMDetail{Name: req.Name, Phase: "Running", Image: req.Image, CPU: req.CPU, Memory: req.Memory, Network: req.Network, PrivateIP: "10.0.0.10"}, nil
}
func (Provider) ListVMs(context.Context) ([]domainvm.VMSummary, error) {
	return []domainvm.VMSummary{{Name: "vm1", Phase: "Running", Image: "ubuntu-24.04", PrivateIP: "10.0.0.10"}}, nil
}
func (Provider) GetVM(ctx context.Context, name string) (domainvm.VMDetail, error) {
	vms, err := Provider{}.ListVMs(ctx)
	if err != nil {
		return domainvm.VMDetail{}, err
	}
	for _, vm := range vms {
		if vm.Name == name {
			return domainvm.VMDetail{Name: vm.Name, Phase: vm.Phase, Image: vm.Image, CPU: 2, Memory: "4Gi", Network: "default", PrivateIP: vm.PrivateIP}, nil
		}
	}
	return domainvm.VMDetail{}, fmt.Errorf("vm not found: %s", name)
}
func (Provider) StartVM(context.Context, string) error   { return nil }
func (Provider) StopVM(context.Context, string) error    { return nil }
func (Provider) RestartVM(context.Context, string) error { return nil }
func (Provider) DeleteVM(context.Context, string) error  { return nil }

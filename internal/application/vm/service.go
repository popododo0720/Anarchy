package vm

import (
	"context"

	domainvm "github.com/popododo0720/anarchy/internal/domain/vm"
	portvm "github.com/popododo0720/anarchy/internal/ports/vm"
)

type Service struct {
	provider portvm.Provider
}

func NewService(provider portvm.Provider) *Service {
	return &Service{provider: provider}
}

func (s *Service) CreateVM(ctx context.Context, req domainvm.CreateVMRequest) (domainvm.VMDetail, error) {
	return s.provider.CreateVM(ctx, req)
}
func (s *Service) ListVMs(ctx context.Context) ([]domainvm.VMSummary, error) {
	return s.provider.ListVMs(ctx)
}
func (s *Service) GetVM(ctx context.Context, name string) (domainvm.VMDetail, error) {
	return s.provider.GetVM(ctx, name)
}
func (s *Service) StartVM(ctx context.Context, name string) error {
	return s.provider.StartVM(ctx, name)
}
func (s *Service) StopVM(ctx context.Context, name string) error { return s.provider.StopVM(ctx, name) }
func (s *Service) RestartVM(ctx context.Context, name string) error {
	return s.provider.RestartVM(ctx, name)
}
func (s *Service) DeleteVM(ctx context.Context, name string) error {
	return s.provider.DeleteVM(ctx, name)
}

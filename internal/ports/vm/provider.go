package vm

import (
	"context"

	domainvm "github.com/popododo0720/anarchy/internal/domain/vm"
)

type Provider interface {
	CreateVM(ctx context.Context, req domainvm.CreateVMRequest) (domainvm.VMDetail, error)
	ListVMs(ctx context.Context) ([]domainvm.VMSummary, error)
	GetVM(ctx context.Context, name string) (domainvm.VMDetail, error)
	StartVM(ctx context.Context, name string) error
	StopVM(ctx context.Context, name string) error
	RestartVM(ctx context.Context, name string) error
	DeleteVM(ctx context.Context, name string) error
}

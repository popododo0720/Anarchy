package vm_test

import (
	"context"
	"errors"
	"testing"

	appvm "github.com/popododo0720/anarchy/internal/application/vm"
	domainvm "github.com/popododo0720/anarchy/internal/domain/vm"
	portvm "github.com/popododo0720/anarchy/internal/ports/vm"
)

type fakeProvider struct {
	vms       []domainvm.VMSummary
	vm        domainvm.VMDetail
	listErr   error
	getErr    error
	actionErr error
	createErr error
	deleteErr error
}

func (f fakeProvider) CreateVM(context.Context, domainvm.CreateVMRequest) (domainvm.VMDetail, error) {
	return f.vm, f.createErr
}
func (f fakeProvider) ListVMs(context.Context) ([]domainvm.VMSummary, error) { return f.vms, f.listErr }
func (f fakeProvider) GetVM(context.Context, string) (domainvm.VMDetail, error) {
	return f.vm, f.getErr
}
func (f fakeProvider) StartVM(context.Context, string) error   { return f.actionErr }
func (f fakeProvider) StopVM(context.Context, string) error    { return f.actionErr }
func (f fakeProvider) RestartVM(context.Context, string) error { return f.actionErr }
func (f fakeProvider) DeleteVM(context.Context, string) error  { return f.deleteErr }

var _ portvm.Provider = fakeProvider{}

func TestServiceCreateVMDelegatesToProvider(t *testing.T) {
	expected := domainvm.VMDetail{Name: "vm1", Phase: "Running", Image: "ubuntu-24.04"}
	service := appvm.NewService(fakeProvider{vm: expected})
	got, err := service.CreateVM(context.Background(), domainvm.CreateVMRequest{Name: "vm1", Image: "ubuntu-24.04", CPU: 2, Memory: "4Gi", Network: "default"})
	if err != nil {
		t.Fatalf("CreateVM() error = %v", err)
	}
	if got.Name != expected.Name || got.Image != expected.Image {
		t.Fatalf("CreateVM() = %#v, want %#v", got, expected)
	}
}

func TestServiceListVMsDelegatesToProvider(t *testing.T) {
	expected := []domainvm.VMSummary{{Name: "vm1", Phase: "Running"}}
	service := appvm.NewService(fakeProvider{vms: expected})
	got, err := service.ListVMs(context.Background())
	if err != nil {
		t.Fatalf("ListVMs() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "vm1" {
		t.Fatalf("ListVMs() = %#v, want %#v", got, expected)
	}
}

func TestServiceActionReturnsProviderError(t *testing.T) {
	expectedErr := errors.New("boom")
	service := appvm.NewService(fakeProvider{actionErr: expectedErr})
	if err := service.StartVM(context.Background(), "vm1"); !errors.Is(err, expectedErr) {
		t.Fatalf("StartVM() error = %v, want %v", err, expectedErr)
	}
}

package hypervisor

import "context"

// Driver interface for hypervisor implementations.
type Driver interface {
	ListVMs(ctx context.Context) ([]VMInfo, error)
	CreateVM(ctx context.Context, spec VMDeploySpec) error
	StartVM(ctx context.Context, name string) error
	StopVM(ctx context.Context, name string) error
	RebootVM(ctx context.Context, name string) error
	DeleteVM(ctx context.Context, name string) error
	GetVM(ctx context.Context, name string) (*VMInfo, error)
}

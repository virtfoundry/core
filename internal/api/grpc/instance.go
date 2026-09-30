package vfgrpc

import (
	"context"
	"strings"

	iaasv1alpha1 "github.com/virtfoundry/core/api/gen/virtfoundry/iaas/v1alpha1"
	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const softListCap = 500

// InstanceBackend is the thin compute facade used by InstanceService.
type InstanceBackend interface {
	ListVMs(ctx context.Context, tenantID string) ([]*platform.PlatformVM, error)
	GetVM(ctx context.Context, tenantID, name string) (*platform.PlatformVM, error)
}

// InstanceServer implements iaasv1alpha1.InstanceServiceServer over InstanceBackend.
type InstanceServer struct {
	iaasv1alpha1.UnimplementedInstanceServiceServer
	Backend InstanceBackend
}

var _ iaasv1alpha1.InstanceServiceServer = (*InstanceServer)(nil)

func (s *InstanceServer) ListInstances(ctx context.Context, req *iaasv1alpha1.ListInstancesRequest) (*iaasv1alpha1.ListInstancesResponse, error) {
	if err := requireVMsRead(ctx); err != nil {
		return nil, err
	}
	tid := TenantIDFromContext(ctx)
	if tid == "" {
		return nil, status.Error(codes.Unauthenticated, "tenant required")
	}
	if s.Backend == nil {
		return nil, status.Error(codes.Internal, "instance backend not configured")
	}
	vms, err := s.Backend.ListVMs(ctx, tid)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list instances: %v", err)
	}
	capN := softListCap
	if req != nil && req.PageSize > 0 && int(req.PageSize) < capN {
		capN = int(req.PageSize)
	}
	out := make([]*iaasv1alpha1.Instance, 0, min(len(vms), capN))
	for i, vm := range vms {
		if i >= capN {
			break
		}
		out = append(out, toProtoInstance(vm))
	}
	return &iaasv1alpha1.ListInstancesResponse{Instances: out}, nil
}

func (s *InstanceServer) GetInstance(ctx context.Context, req *iaasv1alpha1.GetInstanceRequest) (*iaasv1alpha1.GetInstanceResponse, error) {
	if err := requireVMsRead(ctx); err != nil {
		return nil, err
	}
	tid := TenantIDFromContext(ctx)
	if tid == "" {
		return nil, status.Error(codes.Unauthenticated, "tenant required")
	}
	name := ""
	if req != nil {
		name = strings.TrimSpace(req.Name)
	}
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name required")
	}
	if s.Backend == nil {
		return nil, status.Error(codes.Internal, "instance backend not configured")
	}
	vm, err := s.Backend.GetVM(ctx, tid, name)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "instance not found: %v", err)
	}
	return &iaasv1alpha1.GetInstanceResponse{Instance: toProtoInstance(vm)}, nil
}

// WatchInstances is a spike placeholder: emits one MODIFIED snapshot per
// current instance then ends. Real hub-backed streaming is TODO.
func (s *InstanceServer) WatchInstances(req *iaasv1alpha1.WatchInstancesRequest, stream iaasv1alpha1.InstanceService_WatchInstancesServer) error {
	ctx := stream.Context()
	if err := requireVMsRead(ctx); err != nil {
		return err
	}
	tid := TenantIDFromContext(ctx)
	if tid == "" {
		return status.Error(codes.Unauthenticated, "tenant required")
	}
	if s.Backend == nil {
		return status.Error(codes.Internal, "instance backend not configured")
	}
	vms, err := s.Backend.ListVMs(ctx, tid)
	if err != nil {
		return status.Errorf(codes.Internal, "watch instances: %v", err)
	}
	for _, vm := range vms {
		if err := stream.Send(&iaasv1alpha1.WatchInstancesResponse{
			Instance:  toProtoInstance(vm),
			EventType: "MODIFIED",
		}); err != nil {
			return err
		}
	}
	return nil
}

func requireVMsRead(ctx context.Context) error {
	actor := ActorFromContext(ctx)
	if actor == nil {
		return status.Error(codes.Unauthenticated, "missing actor")
	}
	if actor.Role == platform.RoleRoot {
		return nil
	}
	if auth.HasPermission(actor.Permissions, auth.PermVMsRead) {
		return nil
	}
	return status.Error(codes.PermissionDenied, "vms:read required")
}

func toProtoInstance(vm *platform.PlatformVM) *iaasv1alpha1.Instance {
	if vm == nil {
		return nil
	}
	return &iaasv1alpha1.Instance{
		Id:          vm.ID,
		TenantId:    vm.TenantID,
		Name:        vm.Name,
		DisplayName: vm.DisplayName,
		Namespace:   vm.Namespace,
		State:       vm.State,
		Cpu:         int32(vm.CPU),
		MemoryMi:    vm.MemoryMi,
		Ip:          vm.IP,
		PowerState:  vm.PowerState,
	}
}

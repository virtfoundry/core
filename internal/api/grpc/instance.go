package vfgrpc

import (
	"context"
	"encoding/json"
	"strings"

	iaasv1alpha1 "github.com/virtfoundry/core/api/gen/virtfoundry/iaas/v1alpha1"
	"github.com/virtfoundry/core/internal/api/ws"
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
	// Hub is the same realtime hub that backs /ws/events. Nil → snapshot then EOF
	// (unit tests without a hub).
	Hub *ws.Hub
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

// WatchInstances emits an initial MODIFIED snapshot for each current VM, then
// forwards tenant-scoped vm.* events from the /ws/events hub until the client
// disconnects. Without a Hub, only the snapshot is sent (EOF).
//
// TODO gaps vs full WS parity:
//   - no root all_tenants scope (gRPC always pins to resolved x-tenant-id)
//   - non-vm hub events are ignored
//   - ADDED/MODIFIED enrich via GetVM when possible; hub payload alone is name/state
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

	if s.Hub == nil {
		return nil
	}

	events, cancel := s.Hub.Subscribe(ws.Scope{TenantID: tid})
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			resp, ok := s.mapHubEvent(ctx, tid, ev)
			if !ok {
				continue
			}
			if err := stream.Send(resp); err != nil {
				return err
			}
		}
	}
}

// mapHubEvent converts a hub event into a WatchInstancesResponse. Non-VM
// events are skipped. Returns ok=false when the event should be ignored.
func (s *InstanceServer) mapHubEvent(ctx context.Context, tenantID string, ev ws.Event) (*iaasv1alpha1.WatchInstancesResponse, bool) {
	eventType, ok := watchEventType(ev.Type)
	if !ok {
		return nil, false
	}
	payload := parseVMEventPayload(ev.Payload)
	if payload.Name == "" {
		return nil, false
	}

	inst := &iaasv1alpha1.Instance{
		Id:       payload.ID,
		TenantId: tenantID,
		Name:     payload.Name,
		State:    payload.State,
	}
	if eventType != "DELETED" && s.Backend != nil {
		if vm, err := s.Backend.GetVM(ctx, tenantID, payload.Name); err == nil && vm != nil {
			inst = toProtoInstance(vm)
		}
	}
	return &iaasv1alpha1.WatchInstancesResponse{
		Instance:  inst,
		EventType: eventType,
	}, true
}

func watchEventType(hubType string) (string, bool) {
	switch hubType {
	case "vm.created":
		return "ADDED", true
	case "vm.updated":
		return "MODIFIED", true
	case "vm.deleted":
		return "DELETED", true
	default:
		return "", false
	}
}

type vmEventPayload struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	State string `json:"state,omitempty"`
}

func parseVMEventPayload(payload interface{}) vmEventPayload {
	if payload == nil {
		return vmEventPayload{}
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return vmEventPayload{}
	}
	var out vmEventPayload
	_ = json.Unmarshal(b, &out)
	return out
}

// requireVMsRead mirrors REST AutoPermission for GET /vms (vms:read).
// Root bypasses; everyone else needs PermVMsRead from their role/API key.
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

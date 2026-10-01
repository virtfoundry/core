package vfgrpc_test

import (
	"context"
	"net"
	"testing"
	"time"

	iaasv1alpha1 "github.com/virtfoundry/core/api/gen/virtfoundry/iaas/v1alpha1"
	vfgrpc "github.com/virtfoundry/core/internal/api/grpc"
	"github.com/virtfoundry/core/internal/api/ws"
	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/service/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

type fakeBackend struct {
	vms []*platform.PlatformVM
}

func (f *fakeBackend) ListVMs(ctx context.Context, tenantID string) ([]*platform.PlatformVM, error) {
	out := make([]*platform.PlatformVM, 0, len(f.vms))
	for _, vm := range f.vms {
		if vm.TenantID == tenantID {
			out = append(out, vm)
		}
	}
	return out, nil
}

func (f *fakeBackend) GetVM(ctx context.Context, tenantID, name string) (*platform.PlatformVM, error) {
	for _, vm := range f.vms {
		if vm.TenantID == tenantID && vm.Name == name {
			return vm, nil
		}
	}
	return nil, status.Error(codes.NotFound, "not found")
}

func startTestServer(t *testing.T, authDeps vfgrpc.AuthDeps, backend vfgrpc.InstanceBackend, hub *ws.Hub) iaasv1alpha1.InstanceServiceClient {
	t.Helper()
	lis := bufconn.Listen(bufSize)
	gs := vfgrpc.NewGRPCServer(authDeps, backend, nil, hub)
	go func() {
		_ = gs.Serve(lis)
	}()
	t.Cleanup(func() {
		gs.Stop()
		_ = lis.Close()
	})

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return iaasv1alpha1.NewInstanceServiceClient(conn)
}

func TestListInstances_RequiresAuth(t *testing.T) {
	st := store.NewMemory()
	authSvc := auth.NewService("test-secret-at-least-32-bytes-long!!", 3600)
	ident := identity.New(st)
	client := startTestServer(t, vfgrpc.AuthDeps{Auth: authSvc, Store: st, Identity: ident}, &fakeBackend{}, nil)

	_, err := client.ListInstances(context.Background(), &iaasv1alpha1.ListInstancesRequest{})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("want Unauthenticated, got %v", err)
	}
}

func TestListInstances_OK(t *testing.T) {
	st := store.NewMemory()
	if err := st.SeedIAM(); err != nil {
		t.Fatalf("SeedIAM: %v", err)
	}
	tenantID := store.NewID()
	user := &platform.User{
		ID:       store.NewID(),
		Username: "alice",
		Role:     platform.RoleUser,
		RoleID:   store.SystemRoleIDTenantAdmin,
		TenantID: tenantID,
		State:    "active",
	}
	st.SaveUser(user)

	authSvc := auth.NewService("test-secret-at-least-32-bytes-long!!", 3600)
	token, _, err := authSvc.IssueToken(user)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	ident := identity.New(st)

	backend := &fakeBackend{vms: []*platform.PlatformVM{
		{ID: "1", TenantID: tenantID, Name: "web-1", State: "Running", CPU: 2, MemoryMi: 2048},
		{ID: "2", TenantID: "other", Name: "other-vm", State: "Running"},
	}}
	client := startTestServer(t, vfgrpc.AuthDeps{Auth: authSvc, Store: st, Identity: ident}, backend, nil)

	md := metadata.Pairs("authorization", "Bearer "+token)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := client.ListInstances(ctx, &iaasv1alpha1.ListInstancesRequest{})
	if err != nil {
		t.Fatalf("ListInstances: %v", err)
	}
	if len(resp.Instances) != 1 {
		t.Fatalf("got %d instances, want 1", len(resp.Instances))
	}
	if resp.Instances[0].Name != "web-1" {
		t.Fatalf("got name %q, want web-1", resp.Instances[0].Name)
	}
	if resp.Instances[0].Cpu != 2 || resp.Instances[0].MemoryMi != 2048 {
		t.Fatalf("unexpected sizing: %+v", resp.Instances[0])
	}
}

func TestListInstances_RootRequiresTenantMetadata(t *testing.T) {
	st := store.NewMemory()
	root := &platform.User{
		ID: store.NewID(), Username: "root", Role: platform.RoleRoot,
		RoleID: store.SystemRoleIDRoot, State: "active",
	}
	st.SaveUser(root)
	authSvc := auth.NewService("test-secret-at-least-32-bytes-long!!", 3600)
	token, _, err := authSvc.IssueToken(root)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	ident := identity.New(st)
	client := startTestServer(t, vfgrpc.AuthDeps{Auth: authSvc, Store: st, Identity: ident}, &fakeBackend{}, nil)

	md := metadata.Pairs("authorization", "Bearer "+token)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	_, err = client.ListInstances(ctx, &iaasv1alpha1.ListInstancesRequest{})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("want Unauthenticated without x-tenant-id, got %v", err)
	}

	tenantID := store.NewID()
	md = metadata.Pairs("authorization", "Bearer "+token, "x-tenant-id", tenantID)
	ctx = metadata.NewOutgoingContext(context.Background(), md)
	resp, err := client.ListInstances(ctx, &iaasv1alpha1.ListInstancesRequest{})
	if err != nil {
		t.Fatalf("ListInstances with tenant: %v", err)
	}
	if len(resp.GetInstances()) != 0 {
		t.Fatalf("expected empty list for unused tenant, got %d", len(resp.GetInstances()))
	}
}

func TestGetInstance_RequiresName(t *testing.T) {
	st := store.NewMemory()
	if err := st.SeedIAM(); err != nil {
		t.Fatalf("SeedIAM: %v", err)
	}
	tenantID := store.NewID()
	user := &platform.User{
		ID: store.NewID(), Username: "alice", Role: platform.RoleUser,
		RoleID: store.SystemRoleIDTenantAdmin, TenantID: tenantID, State: "active",
	}
	st.SaveUser(user)
	authSvc := auth.NewService("test-secret-at-least-32-bytes-long!!", 3600)
	token, _, err := authSvc.IssueToken(user)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	client := startTestServer(t, vfgrpc.AuthDeps{Auth: authSvc, Store: st, Identity: identity.New(st)}, &fakeBackend{}, nil)

	md := metadata.Pairs("authorization", "Bearer "+token)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	_, err = client.GetInstance(ctx, &iaasv1alpha1.GetInstanceRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("want InvalidArgument, got %v", err)
	}
}

func TestGetInstance_OK(t *testing.T) {
	st := store.NewMemory()
	if err := st.SeedIAM(); err != nil {
		t.Fatalf("SeedIAM: %v", err)
	}
	tenantID := store.NewID()
	user := &platform.User{
		ID: store.NewID(), Username: "alice", Role: platform.RoleUser,
		RoleID: store.SystemRoleIDTenantAdmin, TenantID: tenantID, State: "active",
	}
	st.SaveUser(user)
	authSvc := auth.NewService("test-secret-at-least-32-bytes-long!!", 3600)
	token, _, err := authSvc.IssueToken(user)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	backend := &fakeBackend{vms: []*platform.PlatformVM{
		{ID: "1", TenantID: tenantID, Name: "web-1", State: "Running", CPU: 2, MemoryMi: 2048},
	}}
	client := startTestServer(t, vfgrpc.AuthDeps{Auth: authSvc, Store: st, Identity: identity.New(st)}, backend, nil)

	md := metadata.Pairs("authorization", "Bearer "+token)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := client.GetInstance(ctx, &iaasv1alpha1.GetInstanceRequest{Name: "web-1"})
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if resp.GetInstance().GetName() != "web-1" {
		t.Fatalf("got %+v", resp.GetInstance())
	}

	_, err = client.GetInstance(ctx, &iaasv1alpha1.GetInstanceRequest{Name: "missing"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("want NotFound, got %v", err)
	}
}

func TestWatchInstances_EmitsSnapshot(t *testing.T) {
	st := store.NewMemory()
	if err := st.SeedIAM(); err != nil {
		t.Fatalf("SeedIAM: %v", err)
	}
	tenantID := store.NewID()
	user := &platform.User{
		ID: store.NewID(), Username: "alice", Role: platform.RoleUser,
		RoleID: store.SystemRoleIDTenantAdmin, TenantID: tenantID, State: "active",
	}
	st.SaveUser(user)
	authSvc := auth.NewService("test-secret-at-least-32-bytes-long!!", 3600)
	token, _, err := authSvc.IssueToken(user)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	backend := &fakeBackend{vms: []*platform.PlatformVM{
		{ID: "1", TenantID: tenantID, Name: "web-1", State: "Running"},
		{ID: "2", TenantID: tenantID, Name: "web-2", State: "Stopped"},
	}}
	// No hub → snapshot then EOF (spike fallback path).
	client := startTestServer(t, vfgrpc.AuthDeps{Auth: authSvc, Store: st, Identity: identity.New(st)}, backend, nil)

	md := metadata.Pairs("authorization", "Bearer "+token)
	ctx, cancel := context.WithTimeout(metadata.NewOutgoingContext(context.Background(), md), 5*time.Second)
	defer cancel()

	stream, err := client.WatchInstances(ctx, &iaasv1alpha1.WatchInstancesRequest{})
	if err != nil {
		t.Fatalf("WatchInstances: %v", err)
	}
	var names []string
	for {
		ev, err := stream.Recv()
		if err != nil {
			break
		}
		if ev.GetEventType() != "MODIFIED" {
			t.Fatalf("want MODIFIED, got %q", ev.GetEventType())
		}
		names = append(names, ev.GetInstance().GetName())
	}
	if len(names) != 2 {
		t.Fatalf("got %v, want 2 events", names)
	}
}

func TestWatchInstances_ForwardsHubEvents(t *testing.T) {
	st := store.NewMemory()
	if err := st.SeedIAM(); err != nil {
		t.Fatalf("SeedIAM: %v", err)
	}
	tenantID := store.NewID()
	user := &platform.User{
		ID: store.NewID(), Username: "alice", Role: platform.RoleUser,
		RoleID: store.SystemRoleIDTenantAdmin, TenantID: tenantID, State: "active",
	}
	st.SaveUser(user)
	authSvc := auth.NewService("test-secret-at-least-32-bytes-long!!", 3600)
	token, _, err := authSvc.IssueToken(user)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	backend := &fakeBackend{vms: []*platform.PlatformVM{
		{ID: "1", TenantID: tenantID, Name: "web-1", State: "Running", CPU: 2, MemoryMi: 1024},
	}}
	hub := ws.NewHub()
	client := startTestServer(t, vfgrpc.AuthDeps{Auth: authSvc, Store: st, Identity: identity.New(st)}, backend, hub)

	md := metadata.Pairs("authorization", "Bearer "+token)
	ctx, cancel := context.WithCancel(metadata.NewOutgoingContext(context.Background(), md))
	defer cancel()

	stream, err := client.WatchInstances(ctx, &iaasv1alpha1.WatchInstancesRequest{})
	if err != nil {
		t.Fatalf("WatchInstances: %v", err)
	}

	snap, err := stream.Recv()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snap.GetEventType() != "MODIFIED" || snap.GetInstance().GetName() != "web-1" {
		t.Fatalf("unexpected snapshot: %+v", snap)
	}

	// Wait until the gRPC Watch has subscribed before broadcasting.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if hub.SubscriberCount() == 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if hub.SubscriberCount() != 1 {
		t.Fatalf("hub subscribers = %d, want 1", hub.SubscriberCount())
	}

	hub.BroadcastTenant(tenantID, "vm.updated", map[string]string{
		"id": "1", "name": "web-1", "state": "Stopped",
	})

	ev, err := stream.Recv()
	if err != nil {
		t.Fatalf("hub event: %v", err)
	}
	if ev.GetEventType() != "MODIFIED" {
		t.Fatalf("want MODIFIED, got %q", ev.GetEventType())
	}
	if ev.GetInstance().GetName() != "web-1" {
		t.Fatalf("got name %q", ev.GetInstance().GetName())
	}
	// Enriched via GetVM (backend still has Running).
	if ev.GetInstance().GetCpu() != 2 {
		t.Fatalf("want enriched cpu=2, got %+v", ev.GetInstance())
	}

	hub.BroadcastTenant(tenantID, "vm.deleted", map[string]string{"name": "web-1"})
	del, err := stream.Recv()
	if err != nil {
		t.Fatalf("deleted event: %v", err)
	}
	if del.GetEventType() != "DELETED" || del.GetInstance().GetName() != "web-1" {
		t.Fatalf("unexpected delete: %+v", del)
	}

	// Non-vm events must not be forwarded.
	hub.BroadcastTenant(tenantID, "network.updated", map[string]string{"name": "net-1"})
	backend.vms = append(backend.vms, &platform.PlatformVM{
		ID: "9", TenantID: tenantID, Name: "web-new", State: "Provisioning", CPU: 1, MemoryMi: 512,
	})
	hub.BroadcastTenant(tenantID, "vm.created", map[string]string{
		"id": "9", "name": "web-new", "state": "Provisioning",
	})

	added, err := stream.Recv()
	if err != nil {
		t.Fatalf("added event: %v", err)
	}
	if added.GetEventType() != "ADDED" || added.GetInstance().GetName() != "web-new" {
		t.Fatalf("unexpected added: %+v", added)
	}

	cancel()
	_, err = stream.Recv()
	if err == nil {
		t.Fatal("expected EOF after cancel")
	}
}

func TestListInstances_RequiresVMsRead(t *testing.T) {
	st := store.NewMemory()
	if err := st.SeedIAM(); err != nil {
		t.Fatalf("SeedIAM: %v", err)
	}
	tenantID := store.NewID()
	roleID := store.NewID()
	st.SaveRole(&platform.RoleRecord{
		ID: roleID, TenantID: tenantID, Name: "no-vms",
	})
	st.SetRolePermissions(roleID, []string{auth.PermNetworksRead})
	user := &platform.User{
		ID: store.NewID(), Username: "net-only", Role: platform.RoleUser,
		RoleID: roleID, TenantID: tenantID, State: "active",
	}
	st.SaveUser(user)
	authSvc := auth.NewService("test-secret-at-least-32-bytes-long!!", 3600)
	token, _, err := authSvc.IssueToken(user)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	client := startTestServer(t, vfgrpc.AuthDeps{Auth: authSvc, Store: st, Identity: identity.New(st)}, &fakeBackend{}, nil)

	md := metadata.Pairs("authorization", "Bearer "+token)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	_, err = client.ListInstances(ctx, &iaasv1alpha1.ListInstancesRequest{})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("want PermissionDenied (AutoPermission parity vms:read), got %v", err)
	}
}

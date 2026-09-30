package vfgrpc_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	iaasv1alpha1 "github.com/virtfoundry/core/api/gen/virtfoundry/iaas/v1alpha1"
	vfgrpc "github.com/virtfoundry/core/internal/api/grpc"
	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/service/identity"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// TestServeDualStack_HTTPAndGRPCSamePort verifies cmux: HTTP and gRPC share one TCP listener.
func TestServeDualStack_HTTPAndGRPCSamePort(t *testing.T) {
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

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := lis.Addr().String()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	})
	hs := &http.Server{Handler: mux}

	backend := &fakeBackend{vms: []*platform.PlatformVM{
		{ID: "1", TenantID: tenantID, Name: "cmux-vm", State: "Running", CPU: 1, MemoryMi: 512},
	}}
	errCh := make(chan error, 1)
	go func() {
		errCh <- vfgrpc.ServeDualStack(vfgrpc.DualStackOptions{
			Listener: lis,
			HTTP:     hs,
			Auth:     vfgrpc.AuthDeps{Auth: authSvc, Store: st, Identity: identity.New(st)},
			Backend:  backend,
			Log:      zap.NewNop(),
		})
	}()
	t.Cleanup(func() {
		_ = hs.Close()
		_ = lis.Close()
		select {
		case <-errCh:
		case <-time.After(2 * time.Second):
		}
	})

	deadline := time.Now().Add(3 * time.Second)
	for {
		resp, err := http.Get(fmt.Sprintf("http://%s/healthz", addr))
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK && string(body) == "ok" {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("HTTP healthz never ready: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	client := iaasv1alpha1.NewInstanceServiceClient(conn)

	md := metadata.Pairs("authorization", "Bearer "+token)
	ctx, cancel := context.WithTimeout(metadata.NewOutgoingContext(context.Background(), md), 5*time.Second)
	defer cancel()

	resp, err := client.ListInstances(ctx, &iaasv1alpha1.ListInstancesRequest{})
	if err != nil {
		t.Fatalf("ListInstances over cmux: %v", err)
	}
	if len(resp.GetInstances()) != 1 || resp.GetInstances()[0].GetName() != "cmux-vm" {
		t.Fatalf("unexpected instances: %+v", resp.GetInstances())
	}
}

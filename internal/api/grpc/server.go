package vfgrpc

import (
	"fmt"
	"net"
	"net/http"

	"github.com/soheilhy/cmux"
	iaasv1alpha1 "github.com/virtfoundry/core/api/gen/virtfoundry/iaas/v1alpha1"
	vksv1alpha1 "github.com/virtfoundry/core/api/gen/virtfoundry/vks/v1alpha1"
	"github.com/virtfoundry/core/internal/api/ws"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// DualStackOptions configures cmux on the shared API listener (:8080).
type DualStackOptions struct {
	Listener net.Listener
	// HTTP is the shared http.Server (timeouts, handler). Served on the cmux HTTP match.
	HTTP    *http.Server
	Auth    AuthDeps
	Backend InstanceBackend
	// Clusters backs the canonical VKS ClusterService. Optional until wired.
	Clusters ClusterBackend
	// Hub backs WatchInstances (same as /ws/events). Optional for tests.
	Hub *ws.Hub
	Log *zap.Logger
}

// ServeDualStack starts HTTP + gRPC behind cmux on the same listener.
// Blocks until cmux.Serve returns. Prefer this over a separate :9090.
func ServeDualStack(opts DualStackOptions) error {
	if opts.Listener == nil {
		return fmt.Errorf("listener required")
	}
	if opts.HTTP == nil || opts.HTTP.Handler == nil {
		return fmt.Errorf("http server with handler required")
	}
	log := opts.Log
	if log == nil {
		log = zap.NewNop()
	}

	mux := cmux.New(opts.Listener)
	grpcL := mux.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	httpL := mux.Match(cmux.Any())

	gs := NewGRPCServer(opts.Auth, opts.Backend, opts.Clusters, opts.Hub)
	hs := opts.HTTP

	errCh := make(chan error, 2)
	go func() {
		log.Info("grpc listening (cmux HTTP/2)")
		if err := gs.Serve(grpcL); err != nil {
			errCh <- fmt.Errorf("grpc: %w", err)
		}
	}()
	go func() {
		log.Info("http listening (cmux any)")
		if err := hs.Serve(httpL); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("http: %w", err)
		}
	}()

	if err := mux.Serve(); err != nil {
		gs.GracefulStop()
		_ = hs.Close()
		return err
	}
	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

// NewGRPCServer builds a gRPC server with auth interceptors, InstanceService, and ClusterService.
func NewGRPCServer(auth AuthDeps, backend InstanceBackend, clusters ClusterBackend, hub *ws.Hub) *grpc.Server {
	gs := grpc.NewServer(
		grpc.UnaryInterceptor(auth.UnaryAuth),
		grpc.StreamInterceptor(auth.StreamAuth),
	)
	iaasv1alpha1.RegisterInstanceServiceServer(gs, &InstanceServer{Backend: backend, Hub: hub})
	if clusters != nil {
		vksv1alpha1.RegisterClusterServiceServer(gs, &ClusterServer{Backend: clusters})
	}
	return gs
}

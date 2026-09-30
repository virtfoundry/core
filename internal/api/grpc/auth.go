// Package vfgrpc hosts the gRPC spike transport (#135).
// REST remains canonical for UI/TF; this package is experimental.
package vfgrpc

import (
	"context"
	"strings"

	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/service/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type ctxKey int

const (
	ctxClaims ctxKey = iota
	ctxActor
	ctxTenant
)

const (
	mdAuthorization = "authorization"
	mdTenantID      = "x-tenant-id"
)

// AuthDeps wires JWT / API-key validation for the unary+stream interceptors.
type AuthDeps struct {
	Auth     *auth.Service
	Store    store.Repository
	Identity *identity.Service
}

// UnaryAuth is a fail-closed interceptor: requires authorization Bearer (or
// metadata that extractBearer accepts) and a resolvable tenant. Never reads
// query-string credentials.
func (d AuthDeps) UnaryAuth(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	ctx, err := d.authenticate(ctx)
	if err != nil {
		return nil, err
	}
	return handler(ctx, req)
}

// StreamAuth mirrors UnaryAuth for server streams (Watch).
func (d AuthDeps) StreamAuth(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	ctx, err := d.authenticate(ss.Context())
	if err != nil {
		return err
	}
	return handler(srv, &authenticatedStream{ServerStream: ss, ctx: ctx})
}

type authenticatedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *authenticatedStream) Context() context.Context { return s.ctx }

func (d AuthDeps) authenticate(ctx context.Context) (context.Context, error) {
	if d.Auth == nil || d.Store == nil || d.Identity == nil {
		return nil, status.Error(codes.Internal, "grpc auth not configured")
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	token := extractBearer(md)
	if token == "" {
		return nil, status.Error(codes.Unauthenticated, "missing authorization bearer")
	}

	var actor *auth.Actor
	var claims *auth.Claims
	if auth.IsAPIKeyToken(token) {
		a, err := d.Identity.AuthenticateAPIKey(token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid api key")
		}
		actor = a
		claims = &auth.Claims{
			UserID: a.UserID, Username: a.Username, Role: a.Role, TenantID: a.TenantID,
		}
	} else {
		c, err := d.Auth.ParseToken(token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}
		u, found := d.Store.GetUser(c.UserID)
		if !found || u.State == "disabled" {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}
		actor = d.Identity.ActorFromUser(u)
		claims = c
	}

	requested := firstMD(md, mdTenantID)
	// Root impersonation via x-tenant-id; non-root must not spoof tenants.
	if requested != "" && actor.Role != platform.RoleRoot {
		return nil, status.Error(codes.PermissionDenied, "x-tenant-id forbidden for non-root")
	}
	tid, err := d.Identity.ResolveTenantID(claims, requested)
	if err != nil || tid == "" {
		return nil, status.Error(codes.Unauthenticated, "tenant required")
	}

	ctx = context.WithValue(ctx, ctxClaims, claims)
	ctx = context.WithValue(ctx, ctxActor, actor)
	ctx = context.WithValue(ctx, ctxTenant, tid)
	return ctx, nil
}

func extractBearer(md metadata.MD) string {
	for _, v := range md.Get(mdAuthorization) {
		if len(v) >= 7 && strings.EqualFold(v[:7], "bearer ") {
			return strings.TrimSpace(v[7:])
		}
	}
	return ""
}

func firstMD(md metadata.MD, key string) string {
	vals := md.Get(key)
	if len(vals) == 0 {
		return ""
	}
	return strings.TrimSpace(vals[0])
}

// ClaimsFromContext returns JWT/API-key claims attached by AuthDeps.
func ClaimsFromContext(ctx context.Context) *auth.Claims {
	if v, ok := ctx.Value(ctxClaims).(*auth.Claims); ok {
		return v
	}
	return nil
}

// TenantIDFromContext returns the resolved tenant id (fail-closed auth path).
func TenantIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxTenant).(string); ok {
		return v
	}
	return ""
}

// ActorFromContext returns the authenticated actor.
func ActorFromContext(ctx context.Context) *auth.Actor {
	if v, ok := ctx.Value(ctxActor).(*auth.Actor); ok {
		return v
	}
	return nil
}

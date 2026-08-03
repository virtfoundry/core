package middleware

import (
	"context"
	"net/http"

	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
)

type ctxKey string

const (
	ContextClaims  ctxKey = "claims"
	ContextTenant  ctxKey = "tenant_id"
	ContextActor   ctxKey = "actor"
)

func JWTAuth(authSvc *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token == "" {
				http.Error(w, `{"error":"missing token"}`, http.StatusUnauthorized)
				return
			}
			claims, err := authSvc.ParseToken(token)
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ContextClaims, claims)
			if claims.TenantID != "" {
				ctx = context.WithValue(ctx, ContextTenant, claims.TenantID)
			}
			if tid := r.Header.Get("X-Tenant-ID"); tid != "" && claims.Role == platform.RoleRoot {
				ctx = context.WithValue(ctx, ContextTenant, tid)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRoot(next http.Handler) http.Handler {
	return RequirePermission(auth.PermTenantsWrite)(next)
}

func GetClaims(ctx context.Context) *auth.Claims {
	if v, ok := ctx.Value(ContextClaims).(*auth.Claims); ok {
		return v
	}
	return nil
}

func GetTenantID(ctx context.Context) string {
	if v, ok := ctx.Value(ContextTenant).(string); ok {
		return v
	}
	if claims := GetClaims(ctx); claims != nil {
		return claims.TenantID
	}
	return ""
}


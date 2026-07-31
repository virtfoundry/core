package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/virtforge-cloud/virtforge/internal/auth"
	"github.com/virtforge-cloud/virtforge/internal/platform"
)

type ctxKey string

const (
	ContextClaims  ctxKey = "claims"
	ContextTenant  ctxKey = "tenant_id"
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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r.Context())
		if claims == nil || claims.Role != platform.RoleRoot {
			http.Error(w, `{"error":"root required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
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

func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return r.URL.Query().Get("token")
}

package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/virtforge-cloud/virtforge/internal/platform"
	"github.com/virtforge-cloud/virtforge/internal/platform/store"
)

const ContextImpersonating ctxKey = "impersonating"

// AuditRootImpersonation logs when root operates inside a tenant via X-Tenant-ID.
func AuditRootImpersonation(st store.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r.Context())
			targetTenant := GetTenantID(r.Context())
			impersonating := claims != nil &&
				claims.Role == platform.RoleRoot &&
				targetTenant != "" &&
				r.Header.Get("X-Tenant-ID") != "" &&
				r.Header.Get("X-Tenant-ID") != claims.TenantID

			ctx := r.Context()
			if impersonating {
				ctx = contextWithImpersonating(ctx, true)
				st.SaveAuditEvent(&platform.AuditEvent{
					ID:             store.NewID(),
					ActorUserID:    claims.UserID,
					ActorRole:      string(claims.Role),
					TargetTenantID: targetTenant,
					Action:         "api.request",
					Method:         r.Method,
					Path:           r.URL.Path,
					ResourceType:   resourceFromPath(r.URL.Path),
					CreatedAt:      store.Now(),
				})
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func IsImpersonating(ctx context.Context) bool {
	v, _ := ctx.Value(ContextImpersonating).(bool)
	return v
}

func contextWithImpersonating(ctx context.Context, v bool) context.Context {
	return context.WithValue(ctx, ContextImpersonating, v)
}

func resourceFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 && parts[0] == "api" && parts[1] == "v1" {
		return parts[2]
	}
	return ""
}

package middleware

import (
	"net/http"

	"github.com/virtfoundry/core/internal/auth"
)

// RequirePermission denies requests when the actor lacks a permission.
func RequirePermission(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor := GetActor(r.Context())
			if actor == nil {
				claims := GetClaims(r.Context())
				if claims != nil {
					actor = &auth.Actor{
						UserID: claims.UserID, Username: claims.Username,
						Role: claims.Role, TenantID: claims.TenantID,
						Permissions: auth.LegacyRolePermissions(claims.Role),
					}
				}
			}
			if actor == nil || !auth.HasPermission(actor.Permissions, perm) {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyPermission allows if any listed permission matches.
func RequireAnyPermission(perms ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor := GetActor(r.Context())
			if actor == nil {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}
			for _, p := range perms {
				if auth.HasPermission(actor.Permissions, p) {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		})
	}
}

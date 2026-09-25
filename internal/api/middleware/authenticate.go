package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/service/identity"
)

// Authenticate accepts JWT or VirtFoundry API keys (vfd_live_...) from request
// headers only. Credentials are never read from the URL.
func Authenticate(authSvc *auth.Service, st store.Repository, ident *identity.Service) func(http.Handler) http.Handler {
	return authenticate(authSvc, st, ident, false)
}

// AuthenticateWS additionally accepts ?token=, which a browser WebSocket needs
// because it cannot set request headers. Mount it only on endpoints that have
// no ticket-based handshake; /ws/console must not use it.
func AuthenticateWS(authSvc *auth.Service, st store.Repository, ident *identity.Service) func(http.Handler) http.Handler {
	return authenticate(authSvc, st, ident, true)
}

func authenticate(authSvc *auth.Service, st store.Repository, ident *identity.Service, allowQueryToken bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token == "" && allowQueryToken {
				token = r.URL.Query().Get("token")
			}
			if token == "" {
				http.Error(w, `{"error":"missing token"}`, http.StatusUnauthorized)
				return
			}

			var actor *auth.Actor
			var claims *auth.Claims

			if auth.IsAPIKeyToken(token) {
				a, err := ident.AuthenticateAPIKey(token)
				if err != nil {
					http.Error(w, `{"error":"invalid api key"}`, http.StatusUnauthorized)
					return
				}
				actor = a
				claims = actorToClaims(a)
			} else {
				c, err := authSvc.ParseToken(token)
				if err != nil {
					http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
					return
				}
				u, ok := st.GetUser(c.UserID)
				if !ok || u.State == "disabled" {
					http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
					return
				}
				actor = ident.ActorFromUser(u)
				claims = c
			}

			ctx := context.WithValue(r.Context(), ContextClaims, claims)
			ctx = context.WithValue(ctx, ContextActor, actor)
			if actor.TenantID != "" {
				ctx = context.WithValue(ctx, ContextTenant, actor.TenantID)
			}
			if tid := r.Header.Get("X-Tenant-ID"); tid != "" && actor.Role == platform.RoleRoot {
				ctx = context.WithValue(ctx, ContextTenant, tid)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetActor(ctx context.Context) *auth.Actor {
	if v, ok := ctx.Value(ContextActor).(*auth.Actor); ok {
		return v
	}
	return nil
}

func actorToClaims(a *auth.Actor) *auth.Claims {
	return &auth.Claims{
		UserID: a.UserID, Username: a.Username, Role: a.Role, TenantID: a.TenantID,
	}
}

// extractBearer reads the caller credential from headers. It deliberately does
// not look at the query string: tokens in URLs end up in access logs,
// reverse-proxy logs and browser history.
func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	if k := r.Header.Get("X-API-Key"); k != "" {
		return k
	}
	return ""
}

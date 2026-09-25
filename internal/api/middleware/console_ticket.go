package middleware

import (
	"context"
	"net/http"

	"github.com/virtfoundry/core/internal/auth"
)

const ContextConsoleTicket ctxKey = "console_ticket"

// ConsoleTicketAuth authenticates a console WebSocket from a single-use ticket
// in the query string. The ticket is redeemed into an actor limited to
// vms:console and pinned to the VM it was issued for.
//
// Requests without a ticket fall through to header authentication, which keeps
// non-browser clients (API key, Authorization header) working.
func ConsoleTicketAuth(tickets *auth.ConsoleTicketStore, fallback func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		withFallback := fallback(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := r.URL.Query().Get("ticket")
			if raw == "" {
				withFallback.ServeHTTP(w, r)
				return
			}

			ticket, err := tickets.Redeem(raw)
			if err != nil {
				http.Error(w, `{"error":"invalid or expired console ticket"}`, http.StatusUnauthorized)
				return
			}

			claims := &auth.Claims{
				UserID:   ticket.UserID,
				Username: ticket.Username,
				Role:     ticket.Role,
				TenantID: ticket.TenantID,
			}
			actor := &auth.Actor{
				UserID:      ticket.UserID,
				Username:    ticket.Username,
				Role:        ticket.Role,
				TenantID:    ticket.TenantID,
				Permissions: []string{auth.PermVMsConsole},
				AuthMethod:  "console_ticket",
			}

			ctx := context.WithValue(r.Context(), ContextClaims, claims)
			ctx = context.WithValue(ctx, ContextActor, actor)
			ctx = context.WithValue(ctx, ContextTenant, ticket.TenantID)
			ctx = context.WithValue(ctx, ContextConsoleTicket, &ticket)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetConsoleTicket(ctx context.Context) *auth.ConsoleTicket {
	if v, ok := ctx.Value(ContextConsoleTicket).(*auth.ConsoleTicket); ok {
		return v
	}
	return nil
}

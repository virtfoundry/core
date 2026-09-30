package handler

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/virtfoundry/core/internal/api/middleware"
	"github.com/virtfoundry/core/internal/api/ws"
	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/service"
)

// EventsHandler upgrades /ws/events and subscribes the caller to its own
// tenant's event stream. It must be mounted behind EventsTicketAuth (or
// header Authenticate for non-browser clients).
type EventsHandler struct {
	hub      *ws.Hub
	svc      *service.PlatformService
	tickets  *auth.ConsoleTicketStore
	upgrader websocket.Upgrader
}

func NewEventsHandler(hub *ws.Hub, svc *service.PlatformService, tickets *auth.ConsoleTicketStore, allowedOrigins []string) *EventsHandler {
	return &EventsHandler{
		hub:     hub,
		svc:     svc,
		tickets: tickets,
		upgrader: websocket.Upgrader{
			CheckOrigin: ws.OriginChecker(allowedOrigins),
		},
	}
}

// IssueEventsTicket mints a short-lived credential for /ws/events so the
// browser never puts its session JWT in a WebSocket URL (core#133).
func (h *EventsHandler) IssueEventsTicket(w http.ResponseWriter, r *http.Request) {
	actor := middleware.EffectiveActor(r.Context())
	if actor == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := middleware.GetTenantID(r.Context())
	if tenantID == "" {
		tenantID = actor.TenantID
	}
	token, expiresAt, err := h.tickets.IssueEvents(auth.EventsTicket{
		UserID:   actor.UserID,
		Username: actor.Username,
		Role:     actor.Role,
		TenantID: tenantID,
	})
	if err != nil {
		http.Error(w, `{"error":"could not issue events ticket"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	respondJSON(w, http.StatusCreated, map[string]string{
		"ticket":     token,
		"expires_at": expiresAt.UTC().Format(time.RFC3339),
	})
}

// resolveScope maps the authenticated actor to the tenants it may observe.
// Non-root actors are pinned to their own tenant regardless of query or header
// input; only root can widen the scope.
func (h *EventsHandler) resolveScope(r *http.Request) (ws.Scope, error) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		return ws.Scope{}, auth.ErrUnauthorized
	}

	if claims.Role == platform.RoleRoot && r.URL.Query().Get("all_tenants") == "true" {
		return ws.Scope{AllTenants: true}, nil
	}

	requestedTenant := middleware.GetTenantID(r.Context())
	if requestedTenant == "" {
		requestedTenant = r.URL.Query().Get("tenant_id")
	}
	tenantID, err := h.svc.ResolveTenantID(claims, requestedTenant)
	if err != nil {
		return ws.Scope{}, err
	}
	return ws.Scope{TenantID: tenantID}, nil
}

// Events streams tenant-scoped platform events over WebSocket.
func (h *EventsHandler) Events(w http.ResponseWriter, r *http.Request) {
	scope, err := h.resolveScope(r)
	if err != nil {
		status := http.StatusBadRequest
		if err == auth.ErrUnauthorized {
			status = http.StatusUnauthorized
		}
		http.Error(w, sanitizeClientError(err.Error()), status)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := h.hub.Register(conn, scope)
	if client == nil {
		conn.Close()
		return
	}
	go client.WritePump()
	client.ReadPump()
}

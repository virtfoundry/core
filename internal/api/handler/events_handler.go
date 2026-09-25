package handler

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/virtfoundry/core/internal/api/middleware"
	"github.com/virtfoundry/core/internal/api/ws"
	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/service"
)

// EventsHandler upgrades /ws/events and subscribes the caller to its own
// tenant's event stream. It must be mounted behind middleware.Authenticate.
type EventsHandler struct {
	hub      *ws.Hub
	svc      *service.PlatformService
	upgrader websocket.Upgrader
}

func NewEventsHandler(hub *ws.Hub, svc *service.PlatformService, allowedOrigins []string) *EventsHandler {
	return &EventsHandler{
		hub: hub,
		svc: svc,
		upgrader: websocket.Upgrader{
			CheckOrigin: ws.OriginChecker(allowedOrigins),
		},
	}
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

package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/virtfoundry/core/internal/api/middleware"
	"github.com/virtfoundry/core/internal/api/ws"
	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/infra/hypervisor"
	"github.com/virtfoundry/core/internal/pkg/logger"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/service"
	"go.uber.org/zap"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
)

type ConsoleHandler struct {
	driver         *hypervisor.KubeVirtDriver
	store          store.Repository
	svc            *service.PlatformService
	tickets        *auth.ConsoleTicketStore
	allowedOrigins []string
}

func NewConsoleHandler(driver *hypervisor.KubeVirtDriver, st store.Repository, svc *service.PlatformService, tickets *auth.ConsoleTicketStore, allowedOrigins []string) *ConsoleHandler {
	return &ConsoleHandler{driver: driver, store: st, svc: svc, tickets: tickets, allowedOrigins: allowedOrigins}
}

// resolveVMAccess authorizes the caller for an interactive console and returns
// the VM name plus the namespace it lives in.
//
// A redeemed console ticket is authoritative: it already carries the tenant and
// the VM it was minted for, so query parameters are ignored. Otherwise the
// caller must hold vms:console and the VM is looked up inside the caller's own
// tenant, which is what keeps a viewer out and a cross-tenant name unresolvable.
func (h *ConsoleHandler) resolveVMAccess(r *http.Request) (string, string, error) {
	if ticket := middleware.GetConsoleTicket(r.Context()); ticket != nil {
		return h.vmNamespace(ticket.TenantID, ticket.VMName)
	}

	actor := middleware.EffectiveActor(r.Context())
	if actor == nil {
		return "", "", auth.ErrUnauthorized
	}
	if !auth.HasPermission(actor.Permissions, auth.PermVMsConsole) {
		return "", "", auth.ErrForbidden
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		name = r.URL.Query().Get("virtualmachineid")
	}
	if name == "" {
		return "", "", fmt.Errorf("name required")
	}

	tenantID, err := h.resolveTenant(r)
	if err != nil {
		return "", "", err
	}
	return h.vmNamespace(tenantID, name)
}

// resolveTenant pins the caller to its own tenant; only root may select another.
func (h *ConsoleHandler) resolveTenant(r *http.Request) (string, error) {
	claims := middleware.GetClaims(r.Context())
	requestedTenant := middleware.GetTenantID(r.Context())
	if requestedTenant == "" {
		requestedTenant = r.URL.Query().Get("tenant_id")
	}
	return h.svc.ResolveTenantID(claims, requestedTenant)
}

func (h *ConsoleHandler) vmNamespace(tenantID, name string) (string, string, error) {
	if _, ok := h.store.GetVMByName(tenantID, name); !ok {
		return "", "", fmt.Errorf("vm not found")
	}
	tenant, ok := h.svc.GetTenant(tenantID)
	if !ok {
		return "", "", fmt.Errorf("tenant not found")
	}
	return name, tenant.Namespace, nil
}

// IssueConsoleTicket mints a single-use, short-lived credential for one VM so
// the browser never has to put its JWT in a WebSocket URL. Mounted behind
// Authenticate + RequirePermission(vms:console).
func (h *ConsoleHandler) IssueConsoleTicket(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if name == "" {
		writeConsoleError(w, fmt.Errorf("name required"))
		return
	}

	actor := middleware.EffectiveActor(r.Context())
	if actor == nil {
		writeConsoleError(w, auth.ErrUnauthorized)
		return
	}
	if !auth.HasPermission(actor.Permissions, auth.PermVMsConsole) {
		writeConsoleError(w, auth.ErrForbidden)
		return
	}

	tenantID, err := h.resolveTenant(r)
	if err != nil {
		writeConsoleError(w, err)
		return
	}
	if _, _, err := h.vmNamespace(tenantID, name); err != nil {
		writeConsoleError(w, err)
		return
	}

	token, expiresAt, err := h.tickets.Issue(auth.ConsoleTicket{
		UserID:   actor.UserID,
		Username: actor.Username,
		Role:     actor.Role,
		TenantID: tenantID,
		VMName:   name,
	})
	if err != nil {
		logger.Error("issue console ticket", zap.Error(err), zap.String("vm", name))
		http.Error(w, `{"error":"could not issue console ticket"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	respondJSON(w, http.StatusCreated, map[string]string{
		"ticket":     token,
		"expires_at": expiresAt.UTC().Format(time.RFC3339),
	})
}

func consoleErrorStatus(err error) int {
	switch {
	case errors.Is(err, auth.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, auth.ErrForbidden):
		return http.StatusForbidden
	case err.Error() == "vm not found":
		return http.StatusNotFound
	default:
		return http.StatusBadRequest
	}
}

func writeConsoleError(w http.ResponseWriter, err error) {
	respondJSON(w, consoleErrorStatus(err), map[string]string{"error": sanitizeClientError(err.Error())})
}

// VNCConsole proxies KubeVirt VNC subresource to browser WebSocket (noVNC-compatible).
func (h *ConsoleHandler) VNCConsole(w http.ResponseWriter, r *http.Request) {
	name, namespace, err := h.resolveVMAccess(r)
	if err != nil {
		http.Error(w, sanitizeClientError(err.Error()), consoleErrorStatus(err))
		return
	}

	driver := h.driver.WithNamespace(namespace)
	stream, err := driver.VirtClient().VirtualMachineInstance(namespace).VNC(name, false)
	if err != nil {
		logger.Error("vnc subresource", zap.Error(err), zap.String("vm", name), zap.String("namespace", namespace))
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	upgrader := kvcorev1.NewUpgrader()
	// KubeVirt's NewUpgrader hardcodes CheckOrigin: true; pin it to the same
	// allowlist used by /ws/events and CORS now that console auth is ticket-based.
	upgrader.CheckOrigin = ws.OriginChecker(h.allowedOrigins)
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer wsConn.Close()

	kvConn := stream.AsConn()
	defer kvConn.Close()

	errCh := make(chan error, 2)

	// Browser → KubeVirt (read full WS binary frames before forwarding)
	go func() {
		_, err := kvcorev1.CopyFrom(kvConn, wsConn)
		if err != nil && err != io.EOF {
			errCh <- err
		} else {
			errCh <- io.EOF
		}
	}()

	// KubeVirt → Browser
	go func() {
		_, err := kvcorev1.CopyTo(wsConn, kvConn)
		if err != nil && err != io.EOF {
			errCh <- err
		} else {
			errCh <- io.EOF
		}
	}()

	<-errCh
}

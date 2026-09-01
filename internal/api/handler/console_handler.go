package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/virtfoundry/core/internal/api/middleware"
	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/infra/hypervisor"
	"github.com/virtfoundry/core/internal/pkg/logger"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/service"
	"go.uber.org/zap"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
)

type ConsoleHandler struct {
	driver *hypervisor.KubeVirtDriver
	store  store.Repository
	svc    *service.PlatformService
}

func NewConsoleHandler(driver *hypervisor.KubeVirtDriver, st store.Repository, svc *service.PlatformService) *ConsoleHandler {
	return &ConsoleHandler{driver: driver, store: st, svc: svc}
}

func (h *ConsoleHandler) resolveVMAccess(r *http.Request) (string, string, error) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = r.URL.Query().Get("virtualmachineid")
	}
	if name == "" {
		return "", "", fmt.Errorf("name required")
	}

	claims := middleware.GetClaims(r.Context())
	requestedTenant := middleware.GetTenantID(r.Context())
	if requestedTenant == "" {
		requestedTenant = r.URL.Query().Get("tenant_id")
	}
	tenantID, err := h.svc.ResolveTenantID(claims, requestedTenant)
	if err != nil {
		return "", "", err
	}
	if _, ok := h.store.GetVMByName(tenantID, name); !ok {
		return "", "", fmt.Errorf("vm not found")
	}
	tenant, ok := h.svc.GetTenant(tenantID)
	if !ok {
		return "", "", fmt.Errorf("tenant not found")
	}
	return name, tenant.Namespace, nil
}

// VNCConsole proxies KubeVirt VNC subresource to browser WebSocket (noVNC-compatible).
func (h *ConsoleHandler) VNCConsole(w http.ResponseWriter, r *http.Request) {
	name, namespace, err := h.resolveVMAccess(r)
	if err != nil {
		status := http.StatusBadRequest
		if err == auth.ErrUnauthorized {
			status = http.StatusUnauthorized
		} else if msg := err.Error(); msg == "vm not found" {
			status = http.StatusNotFound
		}
		http.Error(w, sanitizeClientError(err.Error()), status)
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

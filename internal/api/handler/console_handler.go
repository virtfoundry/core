package handler

import (
	"io"
	"net/http"

	"github.com/virtfoundry/core/internal/infra/hypervisor"
	"github.com/virtfoundry/core/internal/pkg/logger"
	"github.com/virtfoundry/core/internal/platform/store"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
	"go.uber.org/zap"
)

type ConsoleHandler struct {
	driver *hypervisor.KubeVirtDriver
	store  store.Repository
}

func NewConsoleHandler(driver *hypervisor.KubeVirtDriver, st store.Repository) *ConsoleHandler {
	return &ConsoleHandler{driver: driver, store: st}
}

func (h *ConsoleHandler) resolveNamespace(vmName, queryNS string) string {
	if queryNS != "" {
		return queryNS
	}
	if h.store != nil {
		for _, t := range h.store.ListTenants() {
			vm, ok := h.store.GetVMByName(t.ID, vmName)
			if ok && vm.Namespace != "" {
				return vm.Namespace
			}
		}
	}
	return h.driver.Namespace()
}

// VNCConsole proxies KubeVirt VNC subresource to browser WebSocket (noVNC-compatible).
func (h *ConsoleHandler) VNCConsole(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = r.URL.Query().Get("virtualmachineid")
	}
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}

	ns := h.resolveNamespace(name, r.URL.Query().Get("namespace"))
	driver := h.driver.WithNamespace(ns)

	stream, err := driver.VirtClient().VirtualMachineInstance(ns).VNC(name, false)
	if err != nil {
		logger.Error("vnc subresource", zap.Error(err), zap.String("vm", name), zap.String("namespace", ns))
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

package handler

import (
	"io"
	"net/http"

	"github.com/virtfoundry/core/internal/infra/hypervisor"
	"github.com/virtfoundry/core/internal/pkg/logger"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
	"go.uber.org/zap"
)

type ConsoleHandler struct {
	driver *hypervisor.KubeVirtDriver
}

func NewConsoleHandler(driver *hypervisor.KubeVirtDriver) *ConsoleHandler {
	return &ConsoleHandler{driver: driver}
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

	driver := h.driver
	if ns := r.URL.Query().Get("namespace"); ns != "" {
		driver = h.driver.WithNamespace(ns)
	}

	stream, err := driver.VirtClient().VirtualMachineInstance(driver.Namespace()).VNC(name, false)
	if err != nil {
		logger.Error("vnc subresource", zap.Error(err), zap.String("vm", name), zap.String("namespace", driver.Namespace()))
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

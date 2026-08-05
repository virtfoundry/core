package storage

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	iaerrors "github.com/virtfoundry/core/internal/pkg/errors"
)

func TestDeleteVolume_AttachedReturnsConflict(t *testing.T) {
	st := store.NewMemory()
	svc := New(st, nil)
	tenantID := store.NewID()
	volID := store.NewID()
	st.SaveVolume(&platform.Volume{
		ID: volID, TenantID: tenantID, VMID: "vm-1",
		Namespace: "tenant-ns", PVCName: "vol-pvc", State: "attached",
	})

	err := svc.DeleteVolume(context.Background(), tenantID, volID)
	var iaErr *iaerrors.IaaSError
	if !errors.As(err, &iaErr) {
		t.Fatalf("expected IaaSError, got %v", err)
	}
	if iaErr.HTTPStatus() != http.StatusConflict {
		t.Fatalf("HTTP status %d, want 409", iaErr.HTTPStatus())
	}
}

func TestDeleteVolume_NotFound(t *testing.T) {
	st := store.NewMemory()
	svc := New(st, nil)

	err := svc.DeleteVolume(context.Background(), store.NewID(), "missing")
	var iaErr *iaerrors.IaaSError
	if !errors.As(err, &iaErr) {
		t.Fatalf("expected IaaSError, got %v", err)
	}
	if iaErr.HTTPStatus() != http.StatusNotFound {
		t.Fatalf("HTTP status %d, want 404", iaErr.HTTPStatus())
	}
}

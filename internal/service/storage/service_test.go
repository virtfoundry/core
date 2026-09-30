package storage

import (
	"context"
	"errors"
	"net/http"
	"testing"

	iaerrors "github.com/virtfoundry/core/internal/pkg/errors"
	"github.com/virtfoundry/core/internal/platform"
	platformk8s "github.com/virtfoundry/core/internal/platform/k8s"
	"github.com/virtfoundry/core/internal/platform/store"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

type recordingSnapshotStore struct {
	*store.Memory
	saveSnapshotCalls int
}

func (r *recordingSnapshotStore) SaveSnapshot(s *platform.Snapshot) {
	r.saveSnapshotCalls++
	r.Memory.SaveSnapshot(s)
}

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

func TestListSnapshots_DoesNotPersistReadiness(t *testing.T) {
	mem := store.NewMemory()
	rec := &recordingSnapshotStore{Memory: mem}
	tenantID := store.NewID()
	ns := "virtfoundry-tenant-acme"
	snap := &platform.Snapshot{
		ID: store.NewID(), TenantID: tenantID, VolumeID: "vol-1",
		Name: "cold-snap", Namespace: ns, State: "creating", CreatedAt: store.Now(),
	}
	mem.SaveSnapshot(snap)

	vsGVR := schema.GroupVersionResource{
		Group: "snapshot.storage.k8s.io", Version: "v1", Resource: "volumesnapshots",
	}
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	dyn := fake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		vsGVR: "VolumeSnapshotList",
	})
	vs := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "snapshot.storage.k8s.io/v1",
		"kind":       "VolumeSnapshot",
		"metadata":   map[string]interface{}{"name": "cold-snap", "namespace": ns},
		"status":     map[string]interface{}{"readyToUse": true},
	}}
	if _, err := dyn.Resource(vsGVR).Namespace(ns).Create(context.Background(), vs, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}

	svc := New(rec, &platformk8s.Manager{Dynamic: dyn})
	out := svc.ListSnapshots(tenantID)
	if len(out) != 1 || out[0].State != "ready" {
		t.Fatalf("expected in-memory ready snapshot, got %#v", out)
	}
	if rec.saveSnapshotCalls != 0 {
		t.Fatalf("ListSnapshots must not SaveSnapshot on GET; got %d saves", rec.saveSnapshotCalls)
	}
}

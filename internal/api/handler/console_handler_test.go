package handler

import (
	"testing"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
)

func TestResolveNamespaceFromStore(t *testing.T) {
	repo := store.NewMemory()
	repo.SaveTenant(&platform.Tenant{ID: "t1", Slug: "default", Namespace: "virtfoundry-tenant-default"})
	repo.SaveVM(&platform.PlatformVM{
		ID: "vm1", TenantID: "t1", Name: "teste", Namespace: "virtfoundry-tenant-default",
	})

	h := NewConsoleHandler(nil, repo)
	if got := h.resolveNamespace("teste", ""); got != "virtfoundry-tenant-default" {
		t.Fatalf("got namespace %q", got)
	}
}

func TestResolveNamespacePrefersQuery(t *testing.T) {
	h := NewConsoleHandler(nil, nil)
	if got := h.resolveNamespace("teste", "custom-ns"); got != "custom-ns" {
		t.Fatalf("got namespace %q", got)
	}
}

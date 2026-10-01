package store

import (
	"testing"

	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform/store/mapping"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func TestGetRolePermissions_CachesHit(t *testing.T) {
	dyn := newTestDynamicClient()
	assignUIDsOnCreate(dyn)
	cs := kubefake.NewSimpleClientset()
	repo := &Kubernetes{dyn: dyn, clientset: cs}
	if err := repo.SeedIAM(); err != nil {
		t.Fatal(err)
	}

	dyn.ClearActions()
	perms, ok := repo.GetRolePermissions(SystemRoleIDRoot)
	if !ok || len(perms) != 1 || perms[0] != auth.PermAll {
		t.Fatalf("first GetRolePermissions: ok=%v perms=%#v", ok, perms)
	}
	firstGets := countResourceVerbs(dyn.Actions(), mapping.RoleGVR.Resource, "get")
	if firstGets < 1 {
		t.Fatalf("expected Role Get on cold cache, got %d", firstGets)
	}

	dyn.ClearActions()
	perms2, ok := repo.GetRolePermissions(SystemRoleIDRoot)
	if !ok || len(perms2) != 1 || perms2[0] != auth.PermAll {
		t.Fatalf("cached GetRolePermissions: ok=%v perms=%#v", ok, perms2)
	}
	if n := countResourceVerbs(dyn.Actions(), mapping.RoleGVR.Resource, "get"); n != 0 {
		t.Fatalf("warm GetRolePermissions must not Get Role; got %d", n)
	}
	if n := countResourceVerbs(dyn.Actions(), mapping.RoleGVR.Resource, "list"); n != 0 {
		t.Fatalf("warm GetRolePermissions must not List Roles; got %d", n)
	}
}

func TestGetRolePermissions_InvalidateOnSet(t *testing.T) {
	dyn := newTestDynamicClient()
	assignUIDsOnCreate(dyn)
	repo := &Kubernetes{dyn: dyn, clientset: kubefake.NewSimpleClientset()}
	if err := repo.SeedIAM(); err != nil {
		t.Fatal(err)
	}

	_, ok := repo.GetRolePermissions(SystemRoleIDTenantViewer)
	if !ok {
		t.Fatal("expected seeded viewer perms")
	}
	repo.SetRolePermissions(SystemRoleIDTenantViewer, []string{auth.PermVMsRead})

	dyn.ClearActions()
	perms, ok := repo.GetRolePermissions(SystemRoleIDTenantViewer)
	if !ok {
		t.Fatal("expected perms after Set")
	}
	if len(perms) != 1 || perms[0] != auth.PermVMsRead {
		t.Fatalf("perms = %#v after invalidate", perms)
	}
	if n := countResourceVerbs(dyn.Actions(), mapping.RoleGVR.Resource, "get"); n < 1 {
		t.Fatalf("expected Role Get after invalidate, got %d", n)
	}
}

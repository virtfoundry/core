package store

import (
	"context"
	"testing"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store/mapping"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kubefake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func countResourceVerbs(actions []k8stesting.Action, resource, verb string) int {
	n := 0
	for _, a := range actions {
		if a.GetVerb() != verb {
			continue
		}
		if a.GetResource().Resource == resource {
			n++
		}
	}
	return n
}

// assignUIDsOnCreate makes the fake dynamic client stamp metadata.uid on Create
// (cluster-scoped CRs often get empty UIDs otherwise).
func assignUIDsOnCreate(dyn interface {
	PrependReactor(verb, resource string, reaction k8stesting.ReactionFunc)
}) {
	dyn.PrependReactor("create", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		obj := action.(k8stesting.CreateAction).GetObject()
		if meta, ok := obj.(metav1.Object); ok && meta.GetUID() == "" {
			meta.SetUID(types.UID("uid-" + meta.GetName()))
		}
		return false, nil, nil
	})
}

func TestListSnapshots_VolumeIDUsesDiskGetNotListVolumes(t *testing.T) {
	dyn := newTestDynamicClient()
	assignUIDsOnCreate(dyn)
	cs := kubefake.NewSimpleClientset()
	repo := &Kubernetes{dyn: dyn, clientset: cs}

	tenant := &platform.Tenant{Name: "Acme", Slug: "acme", State: "active"}
	repo.SaveTenant(tenant)
	ns := tenant.Namespace

	disk := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": mapping.Group + "/" + mapping.Version,
		"kind":       "Disk",
		"metadata":   map[string]interface{}{"name": "boot", "namespace": ns},
		"spec":       map[string]interface{}{"name": "boot", "sizeGi": int64(10)},
	}}
	createdDisk, err := dyn.Resource(mapping.DiskGVR).Namespace(ns).Create(context.Background(), disk, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// Decoy Instance so a ListVolumes→vmIDFromInstanceRef path would List instances.
	_, err = dyn.Resource(mapping.InstanceGVR).Namespace(ns).Create(context.Background(), &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": mapping.Group + "/" + mapping.Version,
		"kind":       "Instance",
		"metadata":   map[string]interface{}{"name": "vm-decoy", "namespace": ns},
		"spec":       map[string]interface{}{"name": "vm-decoy"},
	}}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = dyn.Resource(mapping.DiskSnapshotGVR).Namespace(ns).Create(context.Background(), &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": mapping.Group + "/" + mapping.Version,
		"kind":       "DiskSnapshot",
		"metadata":   map[string]interface{}{"name": "snap1", "namespace": ns},
		"spec": map[string]interface{}{
			"name":    "snap1",
			"diskRef": map[string]interface{}{"name": "boot"},
		},
	}}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}

	dyn.ClearActions()
	snaps := repo.ListSnapshots(tenant.ID)
	if len(snaps) != 1 {
		t.Fatalf("snapshots = %d, want 1", len(snaps))
	}
	wantVolID := string(createdDisk.GetUID())
	if wantVolID == "" {
		t.Fatal("expected Disk UID from fake client")
	}
	if snaps[0].VolumeID != wantVolID {
		t.Fatalf("VolumeID = %q, want %q", snaps[0].VolumeID, wantVolID)
	}

	actions := dyn.Actions()
	if n := countResourceVerbs(actions, mapping.DiskGVR.Resource, "get"); n < 1 {
		t.Fatalf("expected Disk Get for volumeIDFromDiskRef, got %d get actions", n)
	}
	if n := countResourceVerbs(actions, mapping.DiskGVR.Resource, "list"); n != 0 {
		t.Fatalf("ListSnapshots must not ListVolumes (Disk list); got %d", n)
	}
	if n := countResourceVerbs(actions, mapping.InstanceGVR.Resource, "list"); n != 0 {
		t.Fatalf("volumeIDFromDiskRef must not ListVMs (Instance list); got %d", n)
	}
}

func TestListAPIKeys_NoUserListPerKey(t *testing.T) {
	dyn := newTestDynamicClient()
	assignUIDsOnCreate(dyn)
	cs := kubefake.NewSimpleClientset()
	repo := &Kubernetes{dyn: dyn, clientset: cs}
	if err := repo.SeedIAM(); err != nil {
		t.Fatal(err)
	}

	user := &platform.User{
		Username: "ops", PasswordHash: "hash", Role: platform.RoleUser,
		RoleID: SystemRoleIDTenantViewer, State: "active", CreatedAt: Now(),
	}
	repo.SaveUser(user)
	if user.ID == "" {
		t.Fatal("expected user ID after SaveUser")
	}

	for _, name := range []string{"key-a", "key-b", "key-c"} {
		repo.SaveAPIKey(&platform.APIKey{
			ID: NewID(), UserID: user.ID, Name: name, Prefix: name[:4],
			SecretHash: "shh-" + name, CreatedAt: Now(),
		})
	}

	dyn.ClearActions()
	cs.ClearActions()

	keys := repo.ListAPIKeys(user.ID, user.Username)
	if len(keys) != 3 {
		t.Fatalf("keys = %d, want 3", len(keys))
	}
	for _, k := range keys {
		if k.UserID != user.ID {
			t.Fatalf("key UserID = %q, want %q", k.UserID, user.ID)
		}
		if k.SecretHash != "" {
			t.Fatalf("list hydration must skip Secret, got hash %q", k.SecretHash)
		}
	}

	if n := countResourceVerbs(dyn.Actions(), mapping.UserGVR.Resource, "list"); n != 0 {
		t.Fatalf("ListAPIKeys must not ListUsers; got %d User list actions", n)
	}
	if n := countResourceVerbs(dyn.Actions(), mapping.UserGVR.Resource, "get"); n != 1 {
		t.Fatalf("expected exactly 1 User Get (by username), got %d", n)
	}
	secretGets := 0
	for _, a := range cs.Actions() {
		if a.GetVerb() == "get" && a.GetResource().Resource == "secrets" {
			secretGets++
		}
	}
	if secretGets != 0 {
		t.Fatalf("list hydration must not Get Secrets; got %d", secretGets)
	}
}

func TestAPIKeyFromCR_ResolvesUserByGetNotList(t *testing.T) {
	dyn := newTestDynamicClient()
	assignUIDsOnCreate(dyn)
	cs := kubefake.NewSimpleClientset()
	repo := &Kubernetes{dyn: dyn, clientset: cs}
	if err := repo.SeedIAM(); err != nil {
		t.Fatal(err)
	}

	user := &platform.User{
		Username: "alice", PasswordHash: "hash", Role: platform.RoleUser,
		RoleID: SystemRoleIDTenantViewer, State: "active", CreatedAt: Now(),
	}
	repo.SaveUser(user)
	if user.ID == "" {
		t.Fatal("expected user ID after SaveUser")
	}
	key := &platform.APIKey{
		ID: NewID(), UserID: user.ID, Name: "tok", Prefix: "tok1",
		SecretHash: "secret-hash", CreatedAt: Now(),
	}
	repo.SaveAPIKey(key)

	dyn.ClearActions()
	cs.ClearActions()
	got, ok := repo.GetAPIKey(key.ID)
	if !ok {
		t.Fatal("expected API key")
	}
	if got.UserID != user.ID {
		t.Fatalf("UserID = %q, want %q", got.UserID, user.ID)
	}
	if n := countResourceVerbs(dyn.Actions(), mapping.UserGVR.Resource, "list"); n != 0 {
		t.Fatalf("apiKeyFromCR must not ListUsers; got %d", n)
	}
	if n := countResourceVerbs(dyn.Actions(), mapping.UserGVR.Resource, "get"); n < 1 {
		t.Fatalf("expected User Get by userRef.name, got %d", n)
	}
	secretGets := 0
	for _, a := range cs.Actions() {
		if a.GetVerb() == "get" && a.GetResource().Resource == "secrets" {
			secretGets++
		}
	}
	if secretGets < 1 {
		t.Fatalf("GetAPIKey must Get Secret (withSecret); got %d", secretGets)
	}
}

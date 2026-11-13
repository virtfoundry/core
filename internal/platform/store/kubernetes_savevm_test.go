package store

import (
	"errors"
	"testing"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store/mapping"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestSaveVM_ReturnsErrors(t *testing.T) {
	dyn := newTestDynamicClient()
	assignUIDsOnCreate(dyn)
	repo := &Kubernetes{dyn: dyn, clientset: kubefake.NewSimpleClientset()}

	if err := repo.SaveVM(&platform.PlatformVM{ID: "x", TenantID: "missing", Name: "vm"}); err == nil {
		t.Fatal("unknown tenant must return an error")
	}

	tenant := &platform.Tenant{ID: NewID(), Name: "t", Slug: "t", Namespace: "vf-t", State: "active", CreatedAt: Now()}
	repo.SaveTenant(tenant)
	dyn.PrependReactor("create", mapping.InstanceGVR.Resource, func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("denied")
	})
	if err := repo.SaveVM(&platform.PlatformVM{ID: "y", TenantID: tenant.ID, Name: "vm"}); err == nil {
		t.Fatal("failed Instance create must return an error")
	}
}

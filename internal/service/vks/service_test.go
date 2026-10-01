package vks

import (
	"context"
	"testing"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/platform/store/mapping"
	iaerrors "github.com/virtfoundry/core/internal/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestValidateCreateRejectsBadWorkers(t *testing.T) {
	err := validateCreate(CreateInput{
		Name: "demo", KubernetesVersion: "v1.36.5",
		Workers: WorkersSpec{Count: 0, TemplateRef: LocalObjectRef{Name: "t"}, OfferingRef: LocalObjectRef{Name: "o"}, NetworkRef: LocalObjectRef{Name: "n"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var ia *iaerrors.IaaSError
	if !asIaaS(err, &ia) || ia.HTTPStatus() != 400 {
		t.Fatalf("want 400 bad request, got %v", err)
	}
}

func TestValidateCreateRejectsMissingTemplate(t *testing.T) {
	err := validateCreate(CreateInput{
		Name: "demo", KubernetesVersion: "v1.36.5",
		Workers: WorkersSpec{Count: 1, OfferingRef: LocalObjectRef{Name: "o"}, NetworkRef: LocalObjectRef{Name: "n"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateListGetDelete(t *testing.T) {
	mem := store.NewMemory()
	mem.SaveTenant(&platform.Tenant{ID: "t1", Slug: "acme", Namespace: "virtfoundry-tenant-acme", State: "active"})

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	gvrToList := map[schema.GroupVersionResource]string{
		mapping.VKSClusterGVR: "VKSClusterList",
	}
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToList)
	cs := k8sfake.NewSimpleClientset()
	svc := New(mem, dyn, cs)

	created, err := svc.Create(context.Background(), "t1", CreateInput{
		Name: "Demo_1", KubernetesVersion: "v1.36.5",
		Workers: WorkersSpec{
			Count: 1,
			TemplateRef: LocalObjectRef{Name: "ubuntu-node-1-36-5"},
			OfferingRef: LocalObjectRef{Name: "medium"},
			NetworkRef:  LocalObjectRef{Name: "default"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Name != "demo-1" {
		t.Fatalf("sanitized name: got %q", created.Name)
	}

	list, err := svc.List(context.Background(), "t1")
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v len=%d", err, len(list))
	}
	got, err := svc.Get(context.Background(), "t1", "demo-1")
	if err != nil || got.KubernetesVersion != "v1.36.5" {
		t.Fatalf("get: %+v %v", got, err)
	}
	if err := svc.Delete(context.Background(), "t1", "demo-1"); err != nil {
		t.Fatal(err)
	}
}

func TestGetKubeconfigRequiresReady(t *testing.T) {
	mem := store.NewMemory()
	mem.SaveTenant(&platform.Tenant{ID: "t1", Slug: "acme", Namespace: "virtfoundry-tenant-acme", State: "active"})

	obj := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "virtfoundry.io/v1alpha1",
		"kind":       "VKSCluster",
		"metadata":   map[string]interface{}{"name": "demo", "namespace": "virtfoundry-tenant-acme"},
		"spec": map[string]interface{}{
			"kubernetesVersion": "v1.36.5",
			"workers": map[string]interface{}{
				"count":       int64(1),
				"templateRef": map[string]interface{}{"name": "t"},
				"offeringRef": map[string]interface{}{"name": "o"},
				"networkRef":  map[string]interface{}{"name": "n"},
			},
		},
		"status": map[string]interface{}{"phase": "Provisioning"},
	}}
	scheme := runtime.NewScheme()
	gvrToList := map[schema.GroupVersionResource]string{mapping.VKSClusterGVR: "VKSClusterList"}
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToList, obj)
	cs := k8sfake.NewSimpleClientset()
	svc := New(mem, dyn, cs)

	_, err := svc.GetKubeconfig(context.Background(), "t1", "demo")
	if err == nil {
		t.Fatal("expected not ready")
	}
}

func TestGetKubeconfigReturnsAdminConf(t *testing.T) {
	mem := store.NewMemory()
	mem.SaveTenant(&platform.Tenant{ID: "t1", Slug: "acme", Namespace: "virtfoundry-tenant-acme", State: "active"})

	obj := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "virtfoundry.io/v1alpha1",
		"kind":       "VKSCluster",
		"metadata":   map[string]interface{}{"name": "demo", "namespace": "virtfoundry-tenant-acme"},
		"spec": map[string]interface{}{
			"kubernetesVersion": "v1.36.5",
			"workers": map[string]interface{}{
				"count":       int64(1),
				"templateRef": map[string]interface{}{"name": "t"},
				"offeringRef": map[string]interface{}{"name": "o"},
				"networkRef":  map[string]interface{}{"name": "n"},
			},
		},
		"status": map[string]interface{}{
			"phase":               "Ready",
			"kubeconfigSecretRef": "demo-admin-kubeconfig",
		},
	}}
	scheme := runtime.NewScheme()
	gvrToList := map[schema.GroupVersionResource]string{mapping.VKSClusterGVR: "VKSClusterList"}
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToList, obj)
	cs := k8sfake.NewSimpleClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "demo-admin-kubeconfig", Namespace: "virtfoundry-tenant-acme"},
		Data:       map[string][]byte{"admin.conf": []byte("apiVersion: v1\nkind: Config\n")},
	})
	svc := New(mem, dyn, cs)
	raw, err := svc.GetKubeconfig(context.Background(), "t1", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "apiVersion: v1\nkind: Config\n" {
		t.Fatalf("unexpected kubeconfig: %q", raw)
	}
}

func asIaaS(err error, target **iaerrors.IaaSError) bool {
	if err == nil {
		return false
	}
	e, ok := err.(*iaerrors.IaaSError)
	if !ok {
		return false
	}
	*target = e
	return true
}

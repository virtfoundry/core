package vks

import (
	"context"
	"fmt"
	"regexp"
	"time"

	iaerrors "github.com/virtfoundry/core/internal/pkg/errors"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/platform/store/mapping"
	"github.com/virtfoundry/core/internal/service/shared"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var dns1123 = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// Service performs CRD-first VKSCluster operations. It never talks to Kamaji/CAPI.
type Service struct {
	store     store.Repository
	dyn       dynamic.Interface
	clientset kubernetes.Interface
}

func New(st store.Repository, dyn dynamic.Interface, cs kubernetes.Interface) *Service {
	return &Service{store: st, dyn: dyn, clientset: cs}
}

func (s *Service) List(ctx context.Context, tenantID string) ([]Cluster, error) {
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return nil, err
	}
	list, err := s.dyn.Resource(mapping.VKSClusterGVR).Namespace(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list vksclusters: %w", err)
	}
	out := make([]Cluster, 0, len(list.Items))
	for i := range list.Items {
		out = append(out, fromUnstructured(tenantID, &list.Items[i]))
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, tenantID, name string) (*Cluster, error) {
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return nil, err
	}
	obj, err := s.dyn.Resource(mapping.VKSClusterGVR).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, iaerrors.NewNotFoundError("vkscluster", name)
	}
	if err != nil {
		return nil, fmt.Errorf("get vkscluster: %w", err)
	}
	c := fromUnstructured(tenantID, obj)
	return &c, nil
}

func (s *Service) Create(ctx context.Context, tenantID string, in CreateInput) (*Cluster, error) {
	if err := validateCreate(in); err != nil {
		return nil, err
	}
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return nil, err
	}
	obj := toUnstructured(ns, in)
	created, err := s.dyn.Resource(mapping.VKSClusterGVR).Namespace(ns).Create(ctx, obj, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return nil, iaerrors.NewResourceInUseError("vkscluster", "already exists: "+in.Name)
	}
	if err != nil {
		return nil, fmt.Errorf("create vkscluster: %w", err)
	}
	c := fromUnstructured(tenantID, created)
	return &c, nil
}

func (s *Service) Delete(ctx context.Context, tenantID, name string) error {
	ns, err := shared.TenantNamespace(s.store, tenantID)
	if err != nil {
		return err
	}
	err = s.dyn.Resource(mapping.VKSClusterGVR).Namespace(ns).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return iaerrors.NewNotFoundError("vkscluster", name)
	}
	if err != nil {
		return fmt.Errorf("delete vkscluster: %w", err)
	}
	return nil
}

// GetKubeconfig returns admin.conf from the tenant-ns Secret copied by the vks operator.
func (s *Service) GetKubeconfig(ctx context.Context, tenantID, name string) ([]byte, error) {
	c, err := s.Get(ctx, tenantID, name)
	if err != nil {
		return nil, err
	}
	secretName := c.KubeconfigSecretRef
	if secretName == "" {
		secretName = name + "-admin-kubeconfig"
	}
	if c.Phase != "Ready" && c.Phase != "ControlPlaneReady" {
		return nil, iaerrors.NewResourceInUseError("kubeconfig", "cluster not ready (phase="+c.Phase+")")
	}
	sec, err := s.clientset.CoreV1().Secrets(c.Namespace).Get(ctx, secretName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, iaerrors.NewResourceInUseError("kubeconfig", "secret not available yet")
	}
	if err != nil {
		return nil, fmt.Errorf("get kubeconfig secret: %w", err)
	}
	raw, ok := sec.Data["admin.conf"]
	if !ok || len(raw) == 0 {
		return nil, iaerrors.NewInternalError("admin.conf missing in kubeconfig secret")
	}
	return raw, nil
}

func validateCreate(in CreateInput) error {
	name := shared.SanitizeSlug(in.Name)
	if name == "" || !dns1123.MatchString(name) || len(name) > 63 {
		return iaerrors.NewBadRequestError("name must be a DNS-1123 label")
	}
	if in.KubernetesVersion == "" {
		return iaerrors.NewBadRequestError("kubernetes_version is required")
	}
	if in.Workers.Count < 1 || in.Workers.Count > 3 {
		return iaerrors.NewBadRequestError("workers.count must be between 1 and 3")
	}
	if in.Workers.TemplateRef.Name == "" {
		return iaerrors.NewBadRequestError("workers.template_ref.name is required")
	}
	if in.Workers.OfferingRef.Name == "" {
		return iaerrors.NewBadRequestError("workers.offering_ref.name is required")
	}
	if in.Workers.NetworkRef.Name == "" {
		return iaerrors.NewBadRequestError("workers.network_ref.name is required")
	}
	return nil
}

func toUnstructured(ns string, in CreateInput) *unstructured.Unstructured {
	name := shared.SanitizeSlug(in.Name)
	workers := map[string]interface{}{
		"count":       int64(in.Workers.Count),
		"templateRef": map[string]interface{}{"name": in.Workers.TemplateRef.Name},
		"offeringRef": map[string]interface{}{"name": in.Workers.OfferingRef.Name},
		"networkRef":  map[string]interface{}{"name": in.Workers.NetworkRef.Name},
	}
	if len(in.Workers.SSHKeyRefs) > 0 {
		refs := make([]interface{}, 0, len(in.Workers.SSHKeyRefs))
		for _, r := range in.Workers.SSHKeyRefs {
			if r.Name == "" {
				continue
			}
			refs = append(refs, map[string]interface{}{"name": r.Name})
		}
		if len(refs) > 0 {
			workers["sshKeyRefs"] = refs
		}
	}
	cp := map[string]interface{}{}
	if in.ControlPlane.ServiceType != "" {
		cp["serviceType"] = in.ControlPlane.ServiceType
	}
	if in.ControlPlane.Address != "" {
		cp["address"] = in.ControlPlane.Address
	}
	if in.ControlPlane.Port != 0 {
		cp["port"] = int64(in.ControlPlane.Port)
	}
	spec := map[string]interface{}{
		"kubernetesVersion": in.KubernetesVersion,
		"workers":           workers,
	}
	if len(cp) > 0 {
		spec["controlPlane"] = cp
	}
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": mapping.Group + "/" + mapping.Version,
		"kind":       "VKSCluster",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": ns,
		},
		"spec": spec,
	}}
}

func fromUnstructured(tenantID string, obj *unstructured.Unstructured) Cluster {
	c := Cluster{
		Name:      obj.GetName(),
		TenantID:  tenantID,
		Namespace: obj.GetNamespace(),
	}
	c.KubernetesVersion, _, _ = unstructured.NestedString(obj.Object, "spec", "kubernetesVersion")
	c.ControlPlane.ServiceType, _, _ = unstructured.NestedString(obj.Object, "spec", "controlPlane", "serviceType")
	c.ControlPlane.Address, _, _ = unstructured.NestedString(obj.Object, "spec", "controlPlane", "address")
	if p, ok, _ := unstructured.NestedInt64(obj.Object, "spec", "controlPlane", "port"); ok {
		c.ControlPlane.Port = int32(p)
	}
	if n, ok, _ := unstructured.NestedInt64(obj.Object, "spec", "workers", "count"); ok {
		c.Workers.Count = int32(n)
	}
	c.Workers.TemplateRef.Name, _, _ = unstructured.NestedString(obj.Object, "spec", "workers", "templateRef", "name")
	c.Workers.OfferingRef.Name, _, _ = unstructured.NestedString(obj.Object, "spec", "workers", "offeringRef", "name")
	c.Workers.NetworkRef.Name, _, _ = unstructured.NestedString(obj.Object, "spec", "workers", "networkRef", "name")
	if ssh, ok, _ := unstructured.NestedSlice(obj.Object, "spec", "workers", "sshKeyRefs"); ok {
		for _, item := range ssh {
			m, _ := item.(map[string]interface{})
			if m == nil {
				continue
			}
			n, _ := m["name"].(string)
			if n != "" {
				c.Workers.SSHKeyRefs = append(c.Workers.SSHKeyRefs, LocalObjectRef{Name: n})
			}
		}
	}
	c.Phase, _, _ = unstructured.NestedString(obj.Object, "status", "phase")
	c.ControlPlaneEndpoint, _, _ = unstructured.NestedString(obj.Object, "status", "controlPlaneEndpoint")
	c.KubeconfigSecretRef, _, _ = unstructured.NestedString(obj.Object, "status", "kubeconfigSecretRef")
	c.TCPNamespace, _, _ = unstructured.NestedString(obj.Object, "status", "tcpNamespace")
	c.TCPName, _, _ = unstructured.NestedString(obj.Object, "status", "tcpName")
	if rw, ok, _ := unstructured.NestedInt64(obj.Object, "status", "readyWorkers"); ok {
		c.ReadyWorkers = int32(rw)
	}
	if t := obj.GetCreationTimestamp(); !t.IsZero() {
		c.CreatedAt = t.UTC().Format(time.RFC3339)
	}
	if conds, ok, _ := unstructured.NestedSlice(obj.Object, "status", "conditions"); ok {
		for _, item := range conds {
			m, _ := item.(map[string]interface{})
			if m == nil {
				continue
			}
			cond := Condition{}
			cond.Type, _ = m["type"].(string)
			cond.Status, _ = m["status"].(string)
			cond.Reason, _ = m["reason"].(string)
			cond.Message, _ = m["message"].(string)
			if cond.Type != "" {
				c.Conditions = append(c.Conditions, cond)
			}
		}
	}
	return c
}

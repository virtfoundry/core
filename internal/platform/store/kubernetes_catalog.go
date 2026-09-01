package store

import (
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store/mapping"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func (k *Kubernetes) SaveServiceOffering(o *platform.ServiceOffering) {
	k.saveClusterMapped(mapping.OfferingGVR, func() *unstructured.Unstructured {
		return mapping.OfferingToUnstructured(o)
	}, func(saved *unstructured.Unstructured) {
		*o = *mapping.OfferingFromUnstructured(saved)
	})
}

func (k *Kubernetes) GetServiceOffering(id string) (*platform.ServiceOffering, bool) {
	if obj, ok := k.findClusterByID(mapping.OfferingGVR, id); ok {
		return mapping.OfferingFromUnstructured(obj), true
	}
	return nil, false
}

func (k *Kubernetes) GetServiceOfferingByName(name string) (*platform.ServiceOffering, bool) {
	obj, err := k.dyn.Resource(mapping.OfferingGVR).Get(k.ctx(), mapping.SanitizeCRName(name), metav1.GetOptions{})
	if err != nil {
		return nil, false
	}
	return mapping.OfferingFromUnstructured(obj), true
}

func (k *Kubernetes) ListServiceOfferings(activeOnly bool) []*platform.ServiceOffering {
	list, err := k.dyn.Resource(mapping.OfferingGVR).List(k.ctx(), metav1.ListOptions{})
	if err != nil {
		return nil
	}
	out := make([]*platform.ServiceOffering, 0, len(list.Items))
	for i := range list.Items {
		o := mapping.OfferingFromUnstructured(&list.Items[i])
		if activeOnly && o.State != "Active" {
			continue
		}
		out = append(out, o)
	}
	return out
}

func (k *Kubernetes) DeleteServiceOffering(id string) {
	if obj, ok := k.findClusterByID(mapping.OfferingGVR, id); ok {
		k.deleteCluster(mapping.OfferingGVR, obj.GetName())
	}
}

func (k *Kubernetes) templateNS(t *platform.VMTemplate) string {
	if t.TenantID == "" {
		return mapping.SystemNamespace
	}
	if ns, ok := k.tenantNamespace(t.TenantID); ok {
		return ns
	}
	return mapping.SystemNamespace
}

func (k *Kubernetes) SaveVMTemplate(t *platform.VMTemplate) {
	ns := k.templateNS(t)
	k.saveNamespacedMapped(mapping.TemplateGVR, ns, func() *unstructured.Unstructured {
		return mapping.TemplateToUnstructured(t)
	}, func(saved *unstructured.Unstructured) {
		*t = *mapping.TemplateFromUnstructured(saved, t.TenantID)
	})
}

func (k *Kubernetes) GetVMTemplate(id string) (*platform.VMTemplate, bool) {
	obj, ns, ok := k.findNamespacedByID(mapping.TemplateGVR, id)
	if !ok {
		return nil, false
	}
	return mapping.TemplateFromUnstructured(obj, k.tenantIDForNamespace(ns)), true
}

func (k *Kubernetes) listTemplatesInNS(ns, tenantID string, activeOnly bool) []*platform.VMTemplate {
	list, err := k.dyn.Resource(mapping.TemplateGVR).Namespace(ns).List(k.ctx(), metav1.ListOptions{})
	if err != nil {
		return nil
	}
	out := make([]*platform.VMTemplate, 0, len(list.Items))
	for i := range list.Items {
		t := mapping.TemplateFromUnstructured(&list.Items[i], tenantID)
		if activeOnly && t.State != "Active" {
			continue
		}
		out = append(out, t)
	}
	return out
}

func (k *Kubernetes) ListVMTemplates(activeOnly bool) []*platform.VMTemplate {
	var out []*platform.VMTemplate
	for _, obj := range k.listNamespacedAll(mapping.TemplateGVR) {
		t := mapping.TemplateFromUnstructured(&obj, k.tenantIDForNamespace(obj.GetNamespace()))
		if activeOnly && t.State != "Active" {
			continue
		}
		out = append(out, t)
	}
	return out
}

func (k *Kubernetes) ListVMTemplatesForTenant(tenantID string, activeOnly bool) []*platform.VMTemplate {
	out := k.listTemplatesInNS(mapping.SystemNamespace, "", activeOnly)
	if ns, ok := k.tenantNamespace(tenantID); ok {
		out = append(out, k.listTemplatesInNS(ns, tenantID, activeOnly)...)
	}
	return out
}

func (k *Kubernetes) DeleteVMTemplate(id string) {
	if obj, ns, ok := k.findNamespacedByID(mapping.TemplateGVR, id); ok {
		k.deleteNamespaced(mapping.TemplateGVR, ns, obj.GetName())
	}
}

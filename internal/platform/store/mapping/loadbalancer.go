package mapping

import (
	"github.com/virtfoundry/core/internal/platform"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// LBCRName is the LoadBalancer CR name. The operator exposes it as Service
// "lb-<name>", the name the API used before the CRD existed.
func LBCRName(name string) string { return SanitizeCRName(name) }

// TGCRName is the TargetGroup CR name.
func TGCRName(name string) string { return SanitizeCRName(name) }

// LBToUnstructured writes only name/description: listeners are owned by
// LBListenersSpec so a metadata save never wipes them (MergeUnstructuredSpec
// keeps spec keys that src omits).
func LBToUnstructured(lb *platform.LoadBalancer, tenantSlug string) *unstructured.Unstructured {
	obj := newObject("LoadBalancer", LBCRName(lb.Name), "")
	obj.SetLabels(BaseLabels(tenantSlug))
	SetLegacyID(obj, lb.ID)
	spec := map[string]interface{}{"name": lb.Name}
	if lb.Description != "" {
		spec["description"] = lb.Description
	}
	_ = unstructured.SetNestedMap(obj.Object, spec, "spec")
	return obj
}

// LBFromUnstructured maps phase Ready to Active and Failed to Error; anything
// else is still Creating, as before the operator reconciled it.
func LBFromUnstructured(obj *unstructured.Unstructured, tenantID string) *platform.LoadBalancer {
	name, _, _ := unstructured.NestedString(obj.Object, "spec", "name")
	desc, _, _ := unstructured.NestedString(obj.Object, "spec", "description")
	phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
	svc, _, _ := unstructured.NestedString(obj.Object, "status", "serviceName")
	vip, _, _ := unstructured.NestedString(obj.Object, "status", "vip")
	if svc == "" {
		svc = "lb-" + obj.GetName()
	}
	state := "Creating"
	switch phase {
	case "Ready":
		state = "Active"
	case "Failed":
		state = "Error"
	}
	return &platform.LoadBalancer{
		ID: ResourceID(obj), TenantID: tenantID, Name: name, Description: desc,
		Namespace: obj.GetNamespace(), ServiceName: svc, VIP: vip, State: state,
		CreatedAt: obj.GetCreationTimestamp().Time,
	}
}

func TGToUnstructured(tg *platform.TargetGroup, tenantSlug string) *unstructured.Unstructured {
	obj := newObject("TargetGroup", TGCRName(tg.Name), "")
	obj.SetLabels(BaseLabels(tenantSlug))
	SetLegacyID(obj, tg.ID)
	_ = unstructured.SetNestedMap(obj.Object, map[string]interface{}{
		"name": tg.Name, "protocol": tg.Protocol, "port": int64(tg.Port),
	}, "spec")
	return obj
}

func TGFromUnstructured(obj *unstructured.Unstructured, tenantID string) *platform.TargetGroup {
	name, _, _ := unstructured.NestedString(obj.Object, "spec", "name")
	proto, _, _ := unstructured.NestedString(obj.Object, "spec", "protocol")
	port, _, _ := unstructured.NestedInt64(obj.Object, "spec", "port")
	return &platform.TargetGroup{
		ID: ResourceID(obj), TenantID: tenantID, Name: name, Protocol: proto,
		Port: int(port), State: "Active", CreatedAt: obj.GetCreationTimestamp().Time,
	}
}

// Listeners and targets live inside the parent CR's spec. These helpers read
// and rewrite that list; the caller picks the parent and handles conflicts.

func LBListeners(obj *unstructured.Unstructured) []map[string]interface{} {
	return specList(obj, "listeners")
}

func SetLBListeners(obj *unstructured.Unstructured, l []map[string]interface{}) {
	setSpecList(obj, "listeners", l)
}

func TGTargets(obj *unstructured.Unstructured) []map[string]interface{} {
	return specList(obj, "targets")
}

func SetTGTargets(obj *unstructured.Unstructured, t []map[string]interface{}) {
	setSpecList(obj, "targets", t)
}

func LBListenerEntry(l *platform.LBListener, tgCRName string) map[string]interface{} {
	return map[string]interface{}{
		"id": l.ID, "protocol": l.Protocol, "port": int64(l.Port),
		"targetGroupRef": map[string]interface{}{"name": tgCRName},
	}
}

func TGTargetEntry(t *platform.LBTarget) map[string]interface{} {
	e := map[string]interface{}{"id": t.ID, "ip": t.IP}
	if t.VMID != "" {
		e["vmID"] = t.VMID
	}
	return e
}

func LBListenerFromEntry(e map[string]interface{}, lbID, tenantID, tgID string) *platform.LBListener {
	port, _ := e["port"].(int64)
	return &platform.LBListener{
		ID: stringField(e, "id"), TenantID: tenantID, LoadBalancerID: lbID,
		Protocol: stringField(e, "protocol"), Port: int(port), TargetGroupID: tgID,
	}
}

func TGTargetFromEntry(e map[string]interface{}, tg *platform.TargetGroup) *platform.LBTarget {
	return &platform.LBTarget{
		ID: stringField(e, "id"), TenantID: tg.TenantID, TargetGroupID: tg.ID,
		VMID: stringField(e, "vmID"), IP: stringField(e, "ip"), Port: tg.Port,
		State: "registered", CreatedAt: tg.CreatedAt,
	}
}

// TGRefName returns the target group CR name a listener entry points at.
func TGRefName(e map[string]interface{}) string {
	ref, _ := e["targetGroupRef"].(map[string]interface{})
	return stringField(ref, "name")
}

func specList(obj *unstructured.Unstructured, key string) []map[string]interface{} {
	raw, _, _ := unstructured.NestedSlice(obj.Object, "spec", key)
	out := make([]map[string]interface{}, 0, len(raw))
	for _, r := range raw {
		if m, ok := r.(map[string]interface{}); ok {
			out = append(out, m)
		}
	}
	return out
}

func setSpecList(obj *unstructured.Unstructured, key string, l []map[string]interface{}) {
	raw := make([]interface{}, 0, len(l))
	for _, m := range l {
		raw = append(raw, m)
	}
	setSpecField(obj, key, raw)
}

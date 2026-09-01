package store

import (
	"fmt"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store/mapping"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func (k *Kubernetes) resolveVPCCRName(tenantID, vpcID string) string {
	if vpcID == "" {
		return ""
	}
	if obj, _, ok := k.findNamespacedByID(mapping.VPCGVR, vpcID); ok {
		return obj.GetName()
	}
	if v, ok := k.GetVPC(vpcID); ok {
		return mapping.VPCCRName(v)
	}
	return ""
}

func (k *Kubernetes) resolveNetworkCRName(networkID string) string {
	if networkID == "" {
		return ""
	}
	if obj, _, ok := k.findNamespacedByID(mapping.NetworkGVR, networkID); ok {
		return obj.GetName()
	}
	if n, ok := k.GetNetwork(networkID); ok {
		return mapping.NetworkCRName(n)
	}
	return ""
}

func (k *Kubernetes) networkNamespace(n *platform.Network) string {
	if n.NetworkType == platform.NetworkTypeShared {
		return mapping.SystemNamespace
	}
	if n.NADNamespace != "" {
		return n.NADNamespace
	}
	if n.TenantID != "" {
		if ns, ok := k.tenantNamespace(n.TenantID); ok {
			return ns
		}
	}
	return mapping.SystemNamespace
}

func (k *Kubernetes) SaveVPC(v *platform.VPC) {
	ns, ok := k.tenantNamespace(v.TenantID)
	if !ok {
		return
	}
	slug := k.tenantSlug(v.TenantID)
	k.saveNamespacedMapped(mapping.VPCGVR, ns, func() *unstructured.Unstructured {
		return mapping.VPCToUnstructured(v, slug)
	}, func(saved *unstructured.Unstructured) {
		*v = *mapping.VPCFromUnstructured(saved, v.TenantID)
	})
}

func (k *Kubernetes) GetVPC(id string) (*platform.VPC, bool) {
	obj, ns, ok := k.findNamespacedByID(mapping.VPCGVR, id)
	if !ok {
		return nil, false
	}
	return mapping.VPCFromUnstructured(obj, k.tenantIDForNamespace(ns)), true
}

func (k *Kubernetes) ListVPCs(tenantID string) []*platform.VPC {
	ns, ok := k.tenantNamespace(tenantID)
	if !ok {
		return nil
	}
	list, err := k.dyn.Resource(mapping.VPCGVR).Namespace(ns).List(k.ctx(), metav1.ListOptions{})
	if err != nil {
		return nil
	}
	out := make([]*platform.VPC, 0, len(list.Items))
	for i := range list.Items {
		out = append(out, mapping.VPCFromUnstructured(&list.Items[i], tenantID))
	}
	return out
}

func (k *Kubernetes) DeleteVPC(id string) {
	if obj, ns, ok := k.findNamespacedByID(mapping.VPCGVR, id); ok {
		k.deleteNamespaced(mapping.VPCGVR, ns, obj.GetName())
	}
}

func (k *Kubernetes) SaveSG(sg *platform.SecurityGroup) {
	ns, ok := k.tenantNamespace(sg.TenantID)
	if !ok {
		return
	}
	vpcCR := k.resolveVPCCRName(sg.TenantID, sg.VPCID)
	slug := k.tenantSlug(sg.TenantID)
	k.saveNamespacedMapped(mapping.SecurityGroupGVR, ns, func() *unstructured.Unstructured {
		return mapping.SGToUnstructured(sg, slug, vpcCR)
	}, func(saved *unstructured.Unstructured) {
		*sg = *mapping.SGFromUnstructured(saved, sg.TenantID, sg.VPCID)
	})
}

func (k *Kubernetes) ListSGs(tenantID string) []*platform.SecurityGroup {
	ns, ok := k.tenantNamespace(tenantID)
	if !ok {
		return nil
	}
	list, err := k.dyn.Resource(mapping.SecurityGroupGVR).Namespace(ns).List(k.ctx(), metav1.ListOptions{})
	if err != nil {
		return nil
	}
	out := make([]*platform.SecurityGroup, 0, len(list.Items))
	for i := range list.Items {
		out = append(out, mapping.SGFromUnstructured(&list.Items[i], tenantID, k.vpcIDFromRef(tenantID, &list.Items[i])))
	}
	return out
}

func (k *Kubernetes) GetSG(id string) (*platform.SecurityGroup, bool) {
	obj, ns, ok := k.findNamespacedByID(mapping.SecurityGroupGVR, id)
	if !ok {
		return nil, false
	}
	tenantID := k.tenantIDForNamespace(ns)
	return mapping.SGFromUnstructured(obj, tenantID, k.vpcIDFromRef(tenantID, obj)), true
}

func (k *Kubernetes) DeleteSG(id string) {
	if obj, ns, ok := k.findNamespacedByID(mapping.SecurityGroupGVR, id); ok {
		k.deleteNamespaced(mapping.SecurityGroupGVR, ns, obj.GetName())
	}
}

func (k *Kubernetes) vpcIDFromRef(tenantID string, obj *unstructured.Unstructured) string {
	vpcRef, ok, _ := unstructured.NestedString(obj.Object, "spec", "vpcRef", "name")
	if !ok || vpcRef == "" {
		return ""
	}
	for _, v := range k.ListVPCs(tenantID) {
		if mapping.VPCCRName(v) == vpcRef {
			return v.ID
		}
	}
	return ""
}

func (k *Kubernetes) SaveNetwork(n *platform.Network) {
	ns := k.networkNamespace(n)
	if n.NetworkType != platform.NetworkTypeShared && n.TenantID != "" {
		if tns, ok := k.tenantNamespace(n.TenantID); ok {
			ns = tns
		}
	}
	vpcCR := k.resolveVPCCRName(n.TenantID, n.VPCID)
	slug := k.tenantSlug(n.TenantID)
	k.saveNamespacedMapped(mapping.NetworkGVR, ns, func() *unstructured.Unstructured {
		return mapping.NetworkToUnstructured(n, slug, vpcCR)
	}, func(saved *unstructured.Unstructured) {
		*n = *mapping.NetworkFromUnstructured(saved, n.TenantID, n.VPCID)
	})
}

func (k *Kubernetes) ListNetworks(tenantID string) []*platform.Network {
	var out []*platform.Network
	if tenantID != "" {
		if ns, ok := k.tenantNamespace(tenantID); ok {
			out = append(out, k.listNetworksInNS(ns, tenantID)...)
		}
	}
	for _, n := range k.listNetworksInNS(mapping.SystemNamespace, "") {
		if n.NetworkType == platform.NetworkTypeShared {
			out = append(out, n)
		}
	}
	return out
}

func (k *Kubernetes) listNetworksInNS(ns, tenantID string) []*platform.Network {
	list, err := k.dyn.Resource(mapping.NetworkGVR).Namespace(ns).List(k.ctx(), metav1.ListOptions{})
	if err != nil {
		return nil
	}
	out := make([]*platform.Network, 0, len(list.Items))
	for i := range list.Items {
		out = append(out, mapping.NetworkFromUnstructured(&list.Items[i], tenantID, k.vpcIDFromRef(tenantID, &list.Items[i])))
	}
	return out
}

func (k *Kubernetes) GetSharedNetwork() (*platform.Network, bool) {
	obj, err := k.dyn.Resource(mapping.NetworkGVR).Namespace(mapping.SystemNamespace).Get(k.ctx(), "public", metav1.GetOptions{})
	if err == nil {
		return mapping.NetworkFromUnstructured(obj, "", ""), true
	}
	if n, ok := k.findSharedNetworkByLegacyID(); ok {
		return n, true
	}
	return nil, false
}

func (k *Kubernetes) findSharedNetworkByLegacyID() (*platform.Network, bool) {
	obj, _, ok := k.findNamespacedByID(mapping.NetworkGVR, platform.SharedNetworkID)
	if !ok {
		return nil, false
	}
	return mapping.NetworkFromUnstructured(obj, "", ""), true
}

func (k *Kubernetes) GetNetwork(id string) (*platform.Network, bool) {
	if id == platform.SharedNetworkID {
		return k.GetSharedNetwork()
	}
	obj, ns, ok := k.findNamespacedByID(mapping.NetworkGVR, id)
	if !ok {
		return nil, false
	}
	tenantID := k.tenantIDForNamespace(ns)
	return mapping.NetworkFromUnstructured(obj, tenantID, k.vpcIDFromRef(tenantID, obj)), true
}

func (k *Kubernetes) DeleteNetwork(id string) {
	if obj, ns, ok := k.findNamespacedByID(mapping.NetworkGVR, id); ok {
		k.deleteNamespaced(mapping.NetworkGVR, ns, obj.GetName())
	}
}

func (k *Kubernetes) AllocateIPAddress(networkID string) (*platform.IPAddress, error) {
	net, ok := k.GetNetwork(networkID)
	if !ok {
		return nil, fmt.Errorf("network not found")
	}
	ns := k.networkNamespace(net)
	netCR := mapping.NetworkCRName(net)
	list, err := k.dyn.Resource(mapping.IPAddressGVR).Namespace(ns).List(k.ctx(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var best *platform.IPAddress
	for i := range list.Items {
		nref, _, _ := unstructured.NestedString(list.Items[i].Object, "spec", "networkRef", "name")
		if nref != netCR {
			continue
		}
		ip := mapping.IPFromUnstructured(&list.Items[i], networkID)
		if ip.Status == "available" {
			if best == nil || ip.Address < best.Address {
				best = ip
			}
		}
	}
	if best == nil {
		return nil, fmt.Errorf("no available IP in pool")
	}
	best.Status = "allocated"
	return best, nil
}

func (k *Kubernetes) ReleaseIPAddressByVMNic(vmNicID string) {
	for _, obj := range k.listNamespacedAll(mapping.IPAddressGVR) {
		nicRef, _, _ := unstructured.NestedString(obj.Object, "status", "instanceNicRef")
		if nicRef != vmNicID {
			continue
		}
		_ = unstructured.SetNestedField(obj.Object, "", "status", "instanceNicRef")
		copy := obj
		_, _ = k.dyn.Resource(mapping.IPAddressGVR).Namespace(copy.GetNamespace()).Update(k.ctx(), &copy, metav1.UpdateOptions{})
	}
}

func (k *Kubernetes) ReleaseIPAddressByAddress(networkID, address string) {
	net, ok := k.GetNetwork(networkID)
	if !ok {
		return
	}
	ns := k.networkNamespace(net)
	obj, err := k.dyn.Resource(mapping.IPAddressGVR).Namespace(ns).Get(k.ctx(), mapping.IPCRName(address), metav1.GetOptions{})
	if err != nil {
		return
	}
	_ = unstructured.SetNestedField(obj.Object, "", "status", "instanceNicRef")
	_, _ = k.dyn.Resource(mapping.IPAddressGVR).Namespace(ns).Update(k.ctx(), obj, metav1.UpdateOptions{})
}

func (k *Kubernetes) SeedIPPool(networkID, start, end string) error {
	if start == "" || end == "" {
		return nil
	}
	net, ok := k.GetNetwork(networkID)
	if !ok {
		return fmt.Errorf("network not found")
	}
	ns := k.networkNamespace(net)
	netCR := mapping.NetworkCRName(net)
	for addr := start; ; {
		ip := &platform.IPAddress{ID: NewID(), NetworkID: networkID, Address: addr, Status: "available", CreatedAt: Now()}
		obj := mapping.IPToUnstructured(ip, netCR)
		_, _ = k.upsertNamespaced(mapping.IPAddressGVR, ns, obj)
		if addr == end {
			break
		}
		next, ok := nextIPString(addr)
		if !ok || ipGreater(next, end) {
			break
		}
		addr = next
	}
	return nil
}

func (k *Kubernetes) tenantIDForNamespace(ns string) string {
	if ns == mapping.SystemNamespace {
		return ""
	}
	if id, ok := k.tenantSnapshot().nsToID[ns]; ok {
		return id
	}
	return ""
}

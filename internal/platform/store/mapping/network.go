package mapping

import (
	"fmt"
	"strings"

	"github.com/virtfoundry/core/internal/platform"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func VPCCRName(v *platform.VPC) string {
	if n := SanitizeCRName(v.Name); n != "" {
		return n
	}
	return SanitizeCRName(v.ID)
}

func VPCToUnstructured(v *platform.VPC, tenantSlug string) *unstructured.Unstructured {
	obj := newObject("VPC", VPCCRName(v), "")
	obj.SetLabels(BaseLabels(tenantSlug))
	SetLegacyID(obj, v.ID)
	spec := map[string]interface{}{
		"name": v.Name,
		"cidr": v.CIDR,
	}
	_ = unstructured.SetNestedMap(obj.Object, spec, "spec")
	return obj
}

func VPCFromUnstructured(obj *unstructured.Unstructured, tenantID string) *platform.VPC {
	v := &platform.VPC{
		ID:        ResourceID(obj),
		TenantID:  tenantID,
		CreatedAt: obj.GetCreationTimestamp().Time,
		State:     "active",
	}
	name, _, _ := unstructured.NestedString(obj.Object, "spec", "name")
	cidr, _, _ := unstructured.NestedString(obj.Object, "spec", "cidr")
	v.Name = name
	v.CIDR = cidr
	if ns, ok, _ := unstructured.NestedString(obj.Object, "status", "namespace"); ok {
		v.Namespace = ns
	}
	return v
}

func SGCRName(sg *platform.SecurityGroup) string {
	return SanitizeCRName(sg.Name)
}

func SGToUnstructured(sg *platform.SecurityGroup, tenantSlug, vpcCRName string) *unstructured.Unstructured {
	obj := newObject("SecurityGroup", SGCRName(sg), "")
	obj.SetLabels(BaseLabels(tenantSlug))
	SetLegacyID(obj, sg.ID)
	spec := map[string]interface{}{
		"name":        sg.Name,
		"description": sg.Description,
	}
	if vpcCRName != "" {
		spec["vpcRef"] = localRef(vpcCRName)
	}
	if len(sg.Rules) > 0 {
		rules := make([]interface{}, 0, len(sg.Rules))
		for _, r := range sg.Rules {
			rules = append(rules, map[string]interface{}{
				"direction": r.Direction,
				"protocol":  r.Protocol,
				"portFrom":  int64(r.PortFrom),
				"portTo":    int64(r.PortTo),
				"cidr":      r.CIDR,
			})
		}
		spec["rules"] = rules
	}
	_ = unstructured.SetNestedMap(obj.Object, spec, "spec")
	return obj
}

func SGFromUnstructured(obj *unstructured.Unstructured, tenantID string, vpcID string) *platform.SecurityGroup {
	sg := &platform.SecurityGroup{
		ID:        ResourceID(obj),
		TenantID:  tenantID,
		VPCID:     vpcID,
		CreatedAt: obj.GetCreationTimestamp().Time,
	}
	name, _, _ := unstructured.NestedString(obj.Object, "spec", "name")
	desc, _, _ := unstructured.NestedString(obj.Object, "spec", "description")
	sg.Name = name
	sg.Description = desc
	rawRules, _, _ := unstructured.NestedSlice(obj.Object, "spec", "rules")
	for _, rr := range rawRules {
		rm, ok := rr.(map[string]interface{})
		if !ok {
			continue
		}
		rule := platform.SecurityGroupRule{
			Direction: stringField(rm, "direction"),
			Protocol:  stringField(rm, "protocol"),
			CIDR:      stringField(rm, "cidr"),
		}
		if v, ok := rm["portFrom"].(int64); ok {
			rule.PortFrom = int(v)
		}
		if v, ok := rm["portTo"].(int64); ok {
			rule.PortTo = int(v)
		}
		sg.Rules = append(sg.Rules, rule)
	}
	return sg
}

func NetworkCRName(n *platform.Network) string {
	if n.NetworkType == platform.NetworkTypeShared {
		return "public"
	}
	return SanitizeCRName(n.Name)
}

func NetworkToUnstructured(n *platform.Network, tenantSlug, vpcCRName string) *unstructured.Unstructured {
	obj := newObject("Network", NetworkCRName(n), "")
	obj.SetLabels(BaseLabels(tenantSlug))
	SetLegacyID(obj, n.ID)
	spec := map[string]interface{}{
		"name":        n.Name,
		"networkType": n.NetworkType,
		"cidr":        n.CIDR,
	}
	if n.Gateway != "" {
		spec["gateway"] = n.Gateway
	}
	if vpcCRName != "" {
		spec["vpcRef"] = localRef(vpcCRName)
	}
	if imp := importMeta(n.ExternalUUID, n.ImportSource); imp != nil {
		spec["import"] = imp
	}
	_ = unstructured.SetNestedMap(obj.Object, spec, "spec")
	return obj
}

func NetworkFromUnstructured(obj *unstructured.Unstructured, tenantID string, vpcID string) *platform.Network {
	net := &platform.Network{
		ID:        ResourceID(obj),
		TenantID:  tenantID,
		VPCID:     vpcID,
		State:     "active",
		CreatedAt: obj.GetCreationTimestamp().Time,
	}
	name, _, _ := unstructured.NestedString(obj.Object, "spec", "name")
	netType, _, _ := unstructured.NestedString(obj.Object, "spec", "networkType")
	cidr, _, _ := unstructured.NestedString(obj.Object, "spec", "cidr")
	gw, _, _ := unstructured.NestedString(obj.Object, "spec", "gateway")
	net.Name = name
	net.NetworkType = netType
	net.CIDR = cidr
	net.Gateway = gw
	if nadNS, ok, _ := unstructured.NestedString(obj.Object, "status", "nadNamespace"); ok {
		net.NADNamespace = nadNS
	}
	if nadName, ok, _ := unstructured.NestedString(obj.Object, "status", "nadName"); ok {
		net.NADName = nadName
	}
	if ext, ok, _ := unstructured.NestedString(obj.Object, "spec", "import", "externalUUID"); ok {
		net.ExternalUUID = ext
	}
	if src, ok, _ := unstructured.NestedString(obj.Object, "spec", "import", "source"); ok {
		net.ImportSource = src
	}
	return net
}

func IPCRName(address string) string {
	return strings.ReplaceAll(address, ".", "-")
}

func IPToUnstructured(ip *platform.IPAddress, networkCRName string) *unstructured.Unstructured {
	obj := newObject("IPAddress", IPCRName(ip.Address), "")
	spec := map[string]interface{}{
		"networkRef": localRef(networkCRName),
		"address":    ip.Address,
	}
	SetLegacyID(obj, ip.ID)
	_ = unstructured.SetNestedMap(obj.Object, spec, "spec")
	return obj
}

func IPFromUnstructured(obj *unstructured.Unstructured, networkID string) *platform.IPAddress {
	ip := &platform.IPAddress{
		ID:        ResourceID(obj),
		NetworkID: networkID,
		Status:    "available",
		CreatedAt: obj.GetCreationTimestamp().Time,
	}
	addr, _, _ := unstructured.NestedString(obj.Object, "spec", "address")
	ip.Address = addr
	if nicRef, ok, _ := unstructured.NestedString(obj.Object, "status", "instanceNicRef"); ok && nicRef != "" {
		ip.Status = "allocated"
		ip.VMNicID = nicRef
	}
	return ip
}

func stringField(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func NetworkNamespace(n *platform.Network) string {
	if n.NetworkType == platform.NetworkTypeShared {
		return SystemNamespace
	}
	return n.NADNamespace
}

func NetworkNSForShared() string { return SystemNamespace }

func IPNamespaceForNetwork(net *platform.Network) string {
	if net == nil {
		return SystemNamespace
	}
	if net.NADNamespace != "" {
		return net.NADNamespace
	}
	if net.NetworkType == platform.NetworkTypeShared {
		return SystemNamespace
	}
	return fmt.Sprintf("virtfoundry-tenant-unknown")
}

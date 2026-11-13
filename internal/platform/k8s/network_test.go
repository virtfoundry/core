package k8s

import (
	"context"
	"encoding/json"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	"github.com/virtfoundry/core/internal/platform/branding"
)

var nadGVR = schema.GroupVersionResource{Group: "k8s.cni.cncf.io", Version: "v1", Resource: "network-attachment-definitions"}

func nadConfig(t *testing.T, m *Manager, ns, name string) map[string]any {
	t.Helper()
	u, err := m.Dynamic.Resource(nadGVR).Namespace(ns).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _, _ := nestedString(u.Object, "spec", "config")
	var cfg map[string]any
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func nestedString(o map[string]any, keys ...string) (string, bool, error) {
	var cur any = o
	for _, k := range keys {
		mm, ok := cur.(map[string]any)
		if !ok {
			return "", false, nil
		}
		cur = mm[k]
	}
	s, ok := cur.(string)
	return s, ok, nil
}

// Isolated networks of different tenants must not share a Linux bridge nor an
// IPAM scope, and the NAD must match what the operator renders (same bridge,
// same CNI name) so the two writers do not flap the object.
func TestCreateNetworkAttachmentIsolatesTenants(t *testing.T) {
	scheme := runtime.NewScheme()
	m := &Manager{Dynamic: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{nadGVR: "NetworkAttachmentDefinitionList"})}
	ctx := context.Background()
	for _, ns := range []string{"virtfoundry-tenant-default", "virtfoundry-tenant-smoke"} {
		if err := m.CreateNetworkAttachment(ctx, ns, "default-default", "10.0.0.0/24", nil); err != nil {
			t.Fatal(err)
		}
	}
	a := nadConfig(t, m, "virtfoundry-tenant-default", "default-default")
	b := nadConfig(t, m, "virtfoundry-tenant-smoke", "default-default")
	if a["bridge"] == b["bridge"] || a["bridge"] == branding.BridgeName {
		t.Fatalf("tenants share bridge: %v / %v", a["bridge"], b["bridge"])
	}
	if a["bridge"] != branding.TenantBridge("virtfoundry-tenant-default") {
		t.Fatalf("bridge=%v", a["bridge"])
	}
	if a["name"] == b["name"] || a["name"] != "virtfoundry-tenant-default-default-default" {
		t.Fatalf("cni name=%v / %v", a["name"], b["name"])
	}
}

package compute

import (
	"strings"
	"testing"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/branding"
	"github.com/virtfoundry/core/internal/platform/store"
)

func TestTenantDefaultNetworkID_MissingNetworkMessage(t *testing.T) {
	mem := store.NewMemory()
	tenantID := store.NewID()
	vpcID := store.NewID()
	mem.SaveTenant(&platform.Tenant{
		ID: tenantID, Name: "t", Slug: "t", Namespace: "vf-t", State: "active", CreatedAt: store.Now(),
	})
	mem.SaveVPC(&platform.VPC{
		ID: vpcID, TenantID: tenantID, Name: branding.DefaultVPCName, CIDR: branding.DefaultVPCCIDR,
		Namespace: "vf-t", State: "active", CreatedAt: store.Now(),
	})

	svc := &Service{store: mem}
	_, err := svc.tenantDefaultNetworkID(tenantID)
	if err == nil {
		t.Fatal("expected error when default network missing")
	}
	if !strings.Contains(err.Error(), "default network missing") {
		t.Fatalf("error %q should mention missing default network", err)
	}
	if strings.Contains(err.Error(), "default VPC not provisioned") {
		t.Fatalf("error %q should not claim VPC is missing", err)
	}
}

func TestTenantDefaultNetworkID_MissingVPCMessage(t *testing.T) {
	mem := store.NewMemory()
	tenantID := store.NewID()
	mem.SaveTenant(&platform.Tenant{
		ID: tenantID, Name: "t", Slug: "t", Namespace: "vf-t", State: "active", CreatedAt: store.Now(),
	})

	svc := &Service{store: mem}
	_, err := svc.tenantDefaultNetworkID(tenantID)
	if err == nil {
		t.Fatal("expected error when default VPC missing")
	}
	if !strings.Contains(err.Error(), "default VPC not provisioned") {
		t.Fatalf("error %q should say VPC not provisioned", err)
	}
}

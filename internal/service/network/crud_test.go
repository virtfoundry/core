package network

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/branding"
	"github.com/virtfoundry/core/internal/platform/cidr"
	"github.com/virtfoundry/core/internal/platform/store"
	iaerrors "github.com/virtfoundry/core/internal/pkg/errors"
)

func TestDeleteNetwork_RefusesDefaultOnDefaultVPC(t *testing.T) {
	st := store.NewMemory()
	svc := New(st, nil)
	tenantID := store.NewID()
	vpcID := store.NewID()
	netID := store.NewID()

	st.SaveTenant(&platform.Tenant{
		ID: tenantID, Name: "t", Slug: "t", Namespace: "vf-t", State: "active", CreatedAt: store.Now(),
	})
	st.SaveVPC(&platform.VPC{
		ID: vpcID, TenantID: tenantID, Name: branding.DefaultVPCName, CIDR: branding.DefaultVPCCIDR,
		Namespace: "vf-t", State: "active", CreatedAt: store.Now(),
	})
	st.SaveNetwork(&platform.Network{
		ID: netID, TenantID: tenantID, VPCID: vpcID, Name: "default",
		NetworkType: platform.NetworkTypeIsolated, CIDR: "10.0.0.0/24", State: "active", CreatedAt: store.Now(),
	})

	err := svc.DeleteNetwork(context.Background(), tenantID, netID)
	var iaErr *iaerrors.IaaSError
	if !errors.As(err, &iaErr) {
		t.Fatalf("expected IaaSError, got %v", err)
	}
	if iaErr.HTTPStatus() != http.StatusBadRequest {
		t.Fatalf("HTTP status %d, want 400", iaErr.HTTPStatus())
	}
	if _, ok := st.GetNetwork(netID); !ok {
		t.Fatal("default network must remain after refused delete")
	}
}

func TestDeleteNetwork_AllowsDefaultOnNonDefaultVPC(t *testing.T) {
	// Refuse path only for default VPC; other VPCs' "default" subnet is deletable
	// at the policy layer (k8s may be nil — we only assert the guard is skipped).
	st := store.NewMemory()
	svc := New(st, nil)
	tenantID := store.NewID()
	vpcID := store.NewID()
	netID := store.NewID()

	st.SaveTenant(&platform.Tenant{
		ID: tenantID, Name: "t", Slug: "t", Namespace: "vf-t", State: "active", CreatedAt: store.Now(),
	})
	st.SaveVPC(&platform.VPC{
		ID: vpcID, TenantID: tenantID, Name: "audit", CIDR: "10.1.0.0/16",
		Namespace: "vf-t", State: "active", CreatedAt: store.Now(),
	})
	st.SaveNetwork(&platform.Network{
		ID: netID, TenantID: tenantID, VPCID: vpcID, Name: "default",
		NetworkType: platform.NetworkTypeIsolated, CIDR: "10.1.0.0/24", State: "active", CreatedAt: store.Now(),
	})

	err := svc.refuseProtectedDefaultNetwork(tenantID, &platform.Network{
		ID: netID, TenantID: tenantID, VPCID: vpcID, Name: "default",
	})
	if err != nil {
		t.Fatalf("non-default VPC default subnet must be deletable, got %v", err)
	}
}

func TestDeleteVPC_RefusesDefaultVPC(t *testing.T) {
	st := store.NewMemory()
	svc := New(st, nil)
	tenantID := store.NewID()
	vpcID := store.NewID()

	st.SaveTenant(&platform.Tenant{
		ID: tenantID, Name: "t", Slug: "t", Namespace: "vf-t", State: "active", CreatedAt: store.Now(),
	})
	st.SaveVPC(&platform.VPC{
		ID: vpcID, TenantID: tenantID, Name: branding.DefaultVPCName, CIDR: branding.DefaultVPCCIDR,
		Namespace: "vf-t", State: "active", CreatedAt: store.Now(),
	})

	err := svc.DeleteVPC(context.Background(), tenantID, vpcID)
	var iaErr *iaerrors.IaaSError
	if !errors.As(err, &iaErr) {
		t.Fatalf("expected IaaSError, got %v", err)
	}
	if iaErr.HTTPStatus() != http.StatusBadRequest {
		t.Fatalf("HTTP status %d, want 400", iaErr.HTTPStatus())
	}
	if _, ok := st.GetVPC(vpcID); !ok {
		t.Fatal("default VPC must remain after refused delete")
	}
}

// Isolated networks of all tenants meet on the same host (gateway on the
// per-tenant bridge), so VPC CIDRs must not overlap across tenants.
func TestAllVPCCIDRs_SpansTenants(t *testing.T) {
	st := store.NewMemory()
	svc := New(st, nil)
	for _, c := range []string{"10.0.0.0/16", "10.1.0.0/16"} {
		tid := store.NewID()
		st.SaveTenant(&platform.Tenant{ID: tid, Name: "t", Slug: "t", Namespace: "vf-t", State: "active", CreatedAt: store.Now()})
		st.SaveVPC(&platform.VPC{ID: store.NewID(), TenantID: tid, Name: "v", CIDR: c, Namespace: "vf-t", State: "active", CreatedAt: store.Now()})
	}
	got := svc.allVPCCIDRs()
	if len(got) != 2 {
		t.Fatalf("want CIDRs of both tenants, got %v", got)
	}
	if err := cidr.ValidateVPC("10.0.0.0/16", got); err == nil {
		t.Fatal("overlap with another tenant's VPC must be rejected")
	}
}

func TestCreateVPC_RejectsDuplicateNameInTenant(t *testing.T) {
	st := store.NewMemory()
	svc := New(st, nil)
	tenantID := store.NewID()
	st.SaveTenant(&platform.Tenant{
		ID: tenantID, Name: "t", Slug: "t", Namespace: "vf-t", State: "active", CreatedAt: store.Now(),
	})
	st.SaveVPC(&platform.VPC{
		ID: store.NewID(), TenantID: tenantID, Name: "prod", CIDR: "10.5.0.0/16",
		Namespace: "vf-t", State: "active", CreatedAt: store.Now(),
	})

	_, _, err := svc.CreateVPC(context.Background(), tenantID, "prod", "10.6.0.0/16")
	var iaErr *iaerrors.IaaSError
	if !errors.As(err, &iaErr) || iaErr.HTTPStatus() != http.StatusConflict {
		t.Fatalf("expected 409 conflict, got %v", err)
	}
}

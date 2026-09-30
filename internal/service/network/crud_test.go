package network

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/branding"
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

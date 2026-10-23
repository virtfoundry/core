package network

import (
	"context"
	"errors"
	"net/http"
	"testing"

	iaerrors "github.com/virtfoundry/core/internal/pkg/errors"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/branding"
	"github.com/virtfoundry/core/internal/platform/store"
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

func TestUpdateNetwork_TagsReplaceAndKeepsName(t *testing.T) {
	st := store.NewMemory()
	svc := New(st, nil)
	tenantID := store.NewID()
	net := &platform.Network{ID: "net-1", TenantID: tenantID, Name: "app", CIDR: "10.0.0.0/24"}
	st.SaveNetwork(net)

	tags := []string{"frontend", " web ", ""}
	got, err := svc.UpdateNetwork(context.Background(), tenantID, net.ID, "", &tags)
	if err != nil {
		t.Fatalf("UpdateNetwork: %v", err)
	}
	if got.Name != "app" {
		t.Fatalf("name should be preserved when omitted: %q", got.Name)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "frontend" || got.Tags[1] != "web" {
		t.Fatalf("tags not normalized: %#v", got.Tags)
	}

	renamed, err := svc.UpdateNetwork(context.Background(), tenantID, net.ID, "web-tier", nil)
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if renamed.Name != "web-tier" {
		t.Fatalf("name not updated: %q", renamed.Name)
	}
	if len(renamed.Tags) != 2 {
		t.Fatalf("nil tags must keep existing: %#v", renamed.Tags)
	}
}

func TestUpdateNetwork_NotFound(t *testing.T) {
	svc := New(store.NewMemory(), nil)
	tags := []string{"x"}
	if _, err := svc.UpdateNetwork(context.Background(), store.NewID(), "missing", "", &tags); err == nil {
		t.Fatal("expected error for missing network")
	}
}

func TestUpdateVPC_TagsOnlyKeepsName(t *testing.T) {
	st := store.NewMemory()
	svc := New(st, nil)
	tenantID := store.NewID()
	vpc := &platform.VPC{ID: "vpc-1", TenantID: tenantID, Name: "prod", CIDR: "10.0.0.0/16"}
	st.SaveVPC(vpc)

	tags := []string{"prod", " edge ", ""}
	got, err := svc.UpdateVPC(context.Background(), tenantID, vpc.ID, "", &tags)
	if err != nil {
		t.Fatalf("UpdateVPC: %v", err)
	}
	if got.Name != "prod" {
		t.Fatalf("name should be preserved when omitted: %q", got.Name)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "prod" || got.Tags[1] != "edge" {
		t.Fatalf("tags not normalized: %#v", got.Tags)
	}

	renamed, err := svc.UpdateVPC(context.Background(), tenantID, vpc.ID, "staging", nil)
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if renamed.Name != "staging" {
		t.Fatalf("name not updated: %q", renamed.Name)
	}
	if len(renamed.Tags) != 2 {
		t.Fatalf("nil tags must keep existing: %#v", renamed.Tags)
	}
}

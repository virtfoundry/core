package compute

import (
	"strings"
	"testing"

	iaerrors "github.com/virtfoundry/core/internal/pkg/errors"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
)

func TestNeedsLinuxGuestAuth(t *testing.T) {
	t.Parallel()
	if !needsLinuxGuestAuth("linux", nil) {
		t.Fatal("linux without template should require guest auth")
	}
	if !needsLinuxGuestAuth("", nil) {
		t.Fatal("empty osType (default linux image path) should require guest auth")
	}
	if needsLinuxGuestAuth("windows", nil) {
		t.Fatal("windows must not require linux guest auth")
	}
	iso := &platform.VMTemplate{SourceType: "iso", OSType: "windows"}
	if needsLinuxGuestAuth("linux", iso) {
		t.Fatal("ISO templates must not require linux guest auth")
	}
}

func TestDeployVMRejectsLinuxWithoutGuestAuth(t *testing.T) {
	mem := store.NewMemory()
	_ = store.SeedCatalog(mem, "")
	tenant := &platform.Tenant{
		ID: store.NewID(), Name: "t", Slug: "t", Namespace: "vf-t", State: "active", CreatedAt: store.Now(),
	}
	mem.SaveTenant(tenant)

	tmpls := mem.ListVMTemplates(false)
	var ubuntuID string
	for _, tmpl := range tmpls {
		if tmpl.Name == "ubuntu-2204" {
			ubuntuID = tmpl.ID
			break
		}
	}
	if ubuntuID == "" {
		t.Fatal("expected seeded ubuntu-2204 template")
	}

	svc := New(mem, nil, nil, nil)
	_, err := svc.DeployVM(t.Context(), tenant.ID, DeployVMInput{
		Name:       "no-auth",
		TemplateID: ubuntuID,
	})
	if err == nil {
		t.Fatal("expected error when deploying linux VM without SSH key or password")
	}
	iaErr, ok := err.(*iaerrors.IaaSError)
	if !ok {
		t.Fatalf("expected IaaSError, got %T: %v", err, err)
	}
	if iaErr.HTTPStatus() != 400 {
		t.Fatalf("HTTPStatus = %d, want 400", iaErr.HTTPStatus())
	}
	if !strings.Contains(iaErr.Message, "ssh_key_id") {
		t.Fatalf("error message = %q, want mention of ssh_key_id", iaErr.Message)
	}
}

func TestLooksLikeInsecureDefaultUbuntuUserData(t *testing.T) {
	t.Parallel()
	if !looksLikeInsecureDefaultUbuntuUserData("password: ubuntu\nssh_pwauth: true\n") {
		t.Fatal("expected detect insecure default")
	}
	if looksLikeInsecureDefaultUbuntuUserData("password: secret\nssh_pwauth: true\n") {
		t.Fatal("must not treat non-ubuntu password as insecure default")
	}
}

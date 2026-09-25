package compute

import (
	"context"
	"errors"
	"net/http"
	"testing"

	iaerrors "github.com/virtfoundry/core/internal/pkg/errors"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/importurl"
	"github.com/virtfoundry/core/internal/platform/store"
)

func newISOTestService(t *testing.T, allowedHosts []string) (*Service, string) {
	t.Helper()
	st := store.NewMemory()
	tenantID := store.NewID()
	st.SaveTenant(&platform.Tenant{
		ID: tenantID, Name: "acme", Slug: "acme", Namespace: "vf-acme",
		State: "Active", CreatedAt: store.Now(),
	})
	s := New(st, nil, nil, nil)
	s.ConfigureISOImport(importurl.NewPolicy(allowedHosts))
	return s, tenantID
}

// assertBadRequest checks the caller gets a 4xx, not a 500, for a rejected URL.
func assertBadRequest(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	var iaErr *iaerrors.IaaSError
	if !errors.As(err, &iaErr) {
		t.Fatalf("error %v is not an *IaaSError, so it would map to HTTP 500", err)
	}
	if iaErr.HTTPStatus() != http.StatusBadRequest {
		t.Fatalf("HTTPStatus() = %d, want %d", iaErr.HTTPStatus(), http.StatusBadRequest)
	}
}

func TestCreateVMTemplateRejectsSSRFISOURL(t *testing.T) {
	s, tenantID := newISOTestService(t, []string{"iso.example.com"})

	cases := map[string]string{
		"cloud metadata":       "https://169.254.169.254/latest/meta-data/",
		"in-cluster service":   "https://kubernetes.default.svc/api/v1/secrets",
		"private network":      "https://10.0.0.5/win2022.iso",
		"localhost":            "https://localhost:8080/win2022.iso",
		"plain http":           "http://iso.example.com/win2022.iso",
		"internal suffix":      "https://registry.internal/win2022.iso",
		"host not allowlisted": "https://evil.example.com/win2022.iso",
	}

	for name, isoURL := range cases {
		t.Run(name, func(t *testing.T) {
			tmpl, err := s.CreateVMTemplate(context.Background(), tenantID, CreateVMTemplateInput{
				Name: "win2022-" + store.NewID(), SourceType: "iso", Image: isoURL,
			})
			assertBadRequest(t, err)
			if tmpl != nil {
				t.Fatalf("template was created for rejected URL %q", isoURL)
			}
			if len(s.store.ListVMTemplatesForTenant(tenantID, false)) != 0 {
				t.Fatal("rejected ISO URL must not be persisted")
			}
		})
	}
}

// Accepting a URL on the create path starts a background CDI import, so the
// allowed cases are asserted on the gate CreateVMTemplate calls.
func TestValidateISOImportURLAcceptsAllowlistedHosts(t *testing.T) {
	s, _ := newISOTestService(t, []string{"iso.example.com", "*.cdn.example.com"})

	for _, isoURL := range []string{
		"https://iso.example.com/win2022.iso",
		"https://eu.cdn.example.com/win2022.iso?sig=abc123",
	} {
		if err := s.validateISOImportURL(isoURL); err != nil {
			t.Fatalf("validateISOImportURL(%q) = %v, want nil", isoURL, err)
		}
	}
}

func TestCreateVMTemplateSkipsURLPolicyForContainerDisks(t *testing.T) {
	s, tenantID := newISOTestService(t, []string{"iso.example.com"})

	// Container disks are image references pulled by the kubelet, not URLs
	// fetched by CDI, so the ISO allowlist must not apply to them.
	if _, err := s.CreateVMTemplate(context.Background(), tenantID, CreateVMTemplateInput{
		Name: "fedora-40", SourceType: "container", Image: "quay.io/containerdisks/fedora:40",
	}); err != nil {
		t.Fatalf("CreateVMTemplate(container) = %v, want nil", err)
	}
}

func TestUpdateVMTemplateRejectsSSRFISOURL(t *testing.T) {
	s, tenantID := newISOTestService(t, []string{"iso.example.com"})
	tmpl := &platform.VMTemplate{
		ID: store.NewID(), TenantID: tenantID, Name: "win2022", SourceType: "iso",
		Image: "https://iso.example.com/win2022.iso", OSType: "windows",
		ImportState: "ready", State: "Active", CreatedAt: store.Now(),
	}
	s.store.SaveVMTemplate(tmpl)

	_, err := s.UpdateVMTemplate(tenantID, tmpl.ID, "", "", "https://kubernetes.default.svc/api", "", "", "", "")
	assertBadRequest(t, err)

	stored, ok := s.store.GetVMTemplate(tmpl.ID)
	if !ok {
		t.Fatal("template disappeared")
	}
	if stored.Image != "https://iso.example.com/win2022.iso" {
		t.Fatalf("Image = %q, want the original allowlisted URL", stored.Image)
	}
}

func TestValidateISOImportURLDefaultsToBuiltinAllowlist(t *testing.T) {
	// A Service built without ConfigureISOImport must still fail closed.
	s := &Service{}
	if err := s.validateISOImportURL("https://169.254.169.254/latest/meta-data/"); err == nil {
		t.Fatal("unconfigured service accepted a link-local URL")
	}
	if err := s.validateISOImportURL("https://go.microsoft.com/fwlink/?linkid=2195280"); err != nil {
		t.Fatalf("unconfigured service rejected a default-allowlisted URL: %v", err)
	}
}

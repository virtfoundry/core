package store

import (
	"strings"
	"testing"

	"github.com/virtfoundry/core/internal/config"
)

func TestSeedCatalog_NoDefaultUbuntuPassword(t *testing.T) {
	mem := NewMemory()
	if err := SeedCatalog(mem, ""); err != nil {
		t.Fatal(err)
	}
	for _, tmpl := range mem.ListVMTemplates(false) {
		if tmpl.Name != "ubuntu-2204" {
			continue
		}
		if strings.Contains(tmpl.CloudInitUserData, "password: ubuntu") {
			t.Fatalf("seeded ubuntu-2204 with default password ubuntu:\n%s", tmpl.CloudInitUserData)
		}
		if strings.TrimSpace(tmpl.CloudInitUserData) != "" {
			t.Fatalf("expected empty CloudInitUserData without configured password, got:\n%s", tmpl.CloudInitUserData)
		}
		return
	}
	t.Fatal("ubuntu-2204 not seeded")
}

func TestSeedCatalog_StripsInsecureDefault(t *testing.T) {
	mem := NewMemory()
	if err := SeedCatalog(mem, ""); err != nil {
		t.Fatal(err)
	}
	var id string
	for _, tmpl := range mem.ListVMTemplates(false) {
		if tmpl.Name == "ubuntu-2204" {
			tmpl.CloudInitUserData = "#cloud-config\npassword: ubuntu\nssh_pwauth: true\n"
			mem.SaveVMTemplate(tmpl)
			id = tmpl.ID
			break
		}
	}
	if id == "" {
		t.Fatal("ubuntu-2204 not seeded")
	}
	if err := SeedCatalog(mem, ""); err != nil {
		t.Fatal(err)
	}
	got, ok := mem.GetVMTemplate(id)
	if !ok {
		t.Fatal("template missing after re-seed")
	}
	if strings.Contains(got.CloudInitUserData, "password: ubuntu") {
		t.Fatalf("insecure default not stripped:\n%s", got.CloudInitUserData)
	}
}

func TestSeedCatalog_ExplicitPasswordStillAllowed(t *testing.T) {
	mem := NewMemory()
	if err := SeedCatalog(mem, "lab-only-secret"); err != nil {
		t.Fatal(err)
	}
	for _, tmpl := range mem.ListVMTemplates(false) {
		if tmpl.Name != "ubuntu-2204" {
			continue
		}
		if !strings.Contains(tmpl.CloudInitUserData, "password: lab-only-secret") {
			t.Fatalf("expected explicit password in seed user-data:\n%s", tmpl.CloudInitUserData)
		}
		if strings.Contains(tmpl.CloudInitUserData, "password: "+config.BuiltinDefaultVMPassword) {
			t.Fatal("must not embed builtin default alongside explicit password")
		}
		return
	}
	t.Fatal("ubuntu-2204 not seeded")
}

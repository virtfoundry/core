package tenant

import (
	"context"
	"errors"
	"testing"

	iaerrors "github.com/virtfoundry/core/internal/pkg/errors"
	"github.com/virtfoundry/core/internal/platform/store"
)

func TestCreateTenantRejectsEmptyAdminPassword(t *testing.T) {
	st := store.NewMemory()
	svc := New(st, nil)

	for _, pw := range []string{"", "   "} {
		_, _, err := svc.CreateTenant(context.Background(), "Acme", "acme", pw)
		if err == nil {
			t.Fatalf("password %q: expected error", pw)
		}
		var bad *iaerrors.IaaSError
		if !errors.As(err, &bad) || bad.HTTPStatus() != 400 {
			t.Fatalf("password %q: got %v, want bad request", pw, err)
		}
		if len(st.ListTenants()) != 0 {
			t.Fatal("empty password must not create a tenant")
		}
	}
}

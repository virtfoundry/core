package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
)

func TestAutoPermissionDeniesUnmappedSegment(t *testing.T) {
	reached := false
	h := AutoPermission(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
	}))

	req := httptest.NewRequest("GET", "/api/v1/unknown-thing", nil)
	req = req.WithContext(context.WithValue(req.Context(), ContextActor, &auth.Actor{
		Role: platform.RoleRoot, Permissions: []string{auth.PermAll},
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if reached || rec.Code != http.StatusForbidden {
		t.Fatalf("unmapped segment must be 403: reached=%v status=%d", reached, rec.Code)
	}
}

func TestAutoPermissionAllowsMappedRead(t *testing.T) {
	reached := false
	h := AutoPermission(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/vms", nil)
	req = req.WithContext(context.WithValue(req.Context(), ContextActor, &auth.Actor{
		Role: platform.RoleUser, Permissions: []string{auth.PermVMsRead},
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if !reached || rec.Code != http.StatusOK {
		t.Fatalf("mapped read failed: reached=%v status=%d", reached, rec.Code)
	}
}

func TestAutoPermissionAllowsVKSRead(t *testing.T) {
	reached := false
	h := AutoPermission(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/api/v1/vks/clusters", nil)
	req = req.WithContext(context.WithValue(req.Context(), ContextActor, &auth.Actor{
		Role: platform.RoleUser, Permissions: []string{auth.PermVKSRead},
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if !reached || rec.Code != http.StatusOK {
		t.Fatalf("vks read failed: reached=%v status=%d", reached, rec.Code)
	}
}

func TestAutoPermissionAllowsAPIKeysSelfService(t *testing.T) {
	reached := false
	h := AutoPermission(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/api-keys", nil)
	req = req.WithContext(context.WithValue(req.Context(), ContextActor, &auth.Actor{
		Role: platform.RoleUser, Permissions: []string{},
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if !reached || rec.Code != http.StatusOK {
		t.Fatalf("api-keys self-service failed: reached=%v status=%d", reached, rec.Code)
	}
}

func TestAutoPermissionAllowsAuthMeAndDashboardBypass(t *testing.T) {
	for _, path := range []string{"/api/v1/auth/me", "/api/v1/dashboard/summary", "/api/v1/search", "/api/v1/notifications"} {
		reached := false
		h := AutoPermission(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reached = true
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest("GET", path, nil)
		req = req.WithContext(context.WithValue(req.Context(), ContextActor, &auth.Actor{
			Role: platform.RoleUser, Permissions: []string{},
		}))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if !reached || rec.Code != http.StatusOK {
			t.Fatalf("%s bypass failed: reached=%v status=%d", path, reached, rec.Code)
		}
	}
}

func TestAutoPermissionDeniesWriteWithoutPerm(t *testing.T) {
	reached := false
	h := AutoPermission(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
	}))
	req := httptest.NewRequest("POST", "/api/v1/vms", nil)
	req = req.WithContext(context.WithValue(req.Context(), ContextActor, &auth.Actor{
		Role: platform.RoleUser, Permissions: []string{auth.PermVMsRead},
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if reached || rec.Code != http.StatusForbidden {
		t.Fatalf("write without perm must be 403: reached=%v status=%d", reached, rec.Code)
	}
}

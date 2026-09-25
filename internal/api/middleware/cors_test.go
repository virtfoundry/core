package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSReflectsAllowedOrigin(t *testing.T) {
	h := CORS([]string{"https://ui.virtfoundry.test"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://api.virtfoundry.test/api/v1/health", nil)
	req.Host = "api.virtfoundry.test"
	req.Header.Set("Origin", "https://ui.virtfoundry.test")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://ui.virtfoundry.test" {
		t.Fatalf("Allow-Origin = %q, want configured UI origin", got)
	}
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("Vary = %q, want Origin", got)
	}
}

func TestCORSAllowsSameOriginWithoutConfig(t *testing.T) {
	h := CORS(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://console.virtfoundry.test/api/v1/vms", nil)
	req.Host = "console.virtfoundry.test"
	req.Header.Set("Origin", "https://console.virtfoundry.test")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://console.virtfoundry.test" {
		t.Fatalf("Allow-Origin = %q, want same-origin host", got)
	}
}

func TestCORSDeniesForeignOrigin(t *testing.T) {
	h := CORS(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://console.virtfoundry.test/api/v1/vms", nil)
	req.Host = "console.virtfoundry.test"
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Allow-Origin = %q, want empty (fail closed)", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got == "*" {
		t.Fatal("CORS still emits wildcard")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (browser enforces CORS)", rec.Code)
	}
}

func TestCORSNeverEmitsWildcard(t *testing.T) {
	h := CORS(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://api.test/api/v1/vms", nil)
	req.Host = "api.test"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got == "*" {
		t.Fatal("CORS emitted Access-Control-Allow-Origin: *")
	}
}

func TestCORSPreflightAllowsConfiguredOrigin(t *testing.T) {
	h := CORS([]string{"https://ui.virtfoundry.test"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler must not run on OPTIONS")
	}))

	req := httptest.NewRequest(http.MethodOptions, "http://api.virtfoundry.test/api/v1/vms", nil)
	req.Host = "api.virtfoundry.test"
	req.Header.Set("Origin", "https://ui.virtfoundry.test")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://ui.virtfoundry.test" {
		t.Fatalf("Allow-Origin = %q, want configured UI origin", got)
	}
}

func TestCORSPreflightDeniesForeignOrigin(t *testing.T) {
	h := CORS(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler must not run on OPTIONS")
	}))

	req := httptest.NewRequest(http.MethodOptions, "http://console.virtfoundry.test/api/v1/vms", nil)
	req.Host = "console.virtfoundry.test"
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Allow-Origin = %q, want empty", got)
	}
}

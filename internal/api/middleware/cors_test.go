package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
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

// muxCORSRouter mirrors cmd/server: Use(CORS) plus MethodNotAllowed/NotFound
// handlers wrapped in CORS so OPTIONS on method-restricted routes still hit it.
func muxCORSRouter(allowed []string) http.Handler {
	cors := CORS(allowed)
	r := mux.NewRouter()
	r.Use(cors)
	r.MethodNotAllowedHandler = cors(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}))
	r.NotFoundHandler = cors(http.NotFoundHandler())
	v1 := r.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/auth/login", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods(http.MethodPost)
	v1.HandleFunc("/vms", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods(http.MethodGet)
	return r
}

func TestCORSPreflightMuxPOSTOnlyRouteAllowed(t *testing.T) {
	h := muxCORSRouter([]string{"https://ui.virtfoundry.test"})

	req := httptest.NewRequest(http.MethodOptions, "http://api.virtfoundry.test/api/v1/auth/login", nil)
	req.Host = "api.virtfoundry.test"
	req.Header.Set("Origin", "https://ui.virtfoundry.test")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (got body %q)", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://ui.virtfoundry.test" {
		t.Fatalf("Allow-Origin = %q, want configured UI origin", got)
	}
}

func TestCORSPreflightMuxPOSTOnlyRouteDenied(t *testing.T) {
	h := muxCORSRouter(nil)

	req := httptest.NewRequest(http.MethodOptions, "http://api.virtfoundry.test/api/v1/auth/login", nil)
	req.Host = "api.virtfoundry.test"
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (got body %q)", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Allow-Origin = %q, want empty", got)
	}
}

func TestCORSMuxGETUnchanged(t *testing.T) {
	h := muxCORSRouter([]string{"https://ui.virtfoundry.test"})

	req := httptest.NewRequest(http.MethodGet, "http://api.virtfoundry.test/api/v1/vms", nil)
	req.Host = "api.virtfoundry.test"
	req.Header.Set("Origin", "https://ui.virtfoundry.test")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://ui.virtfoundry.test" {
		t.Fatalf("Allow-Origin = %q, want configured UI origin", got)
	}

	deny := httptest.NewRequest(http.MethodGet, "http://api.virtfoundry.test/api/v1/vms", nil)
	deny.Host = "api.virtfoundry.test"
	deny.Header.Set("Origin", "https://evil.example.com")
	denyRec := httptest.NewRecorder()
	h.ServeHTTP(denyRec, deny)
	if denyRec.Code != http.StatusOK {
		t.Fatalf("denied GET status = %d, want 200", denyRec.Code)
	}
	if got := denyRec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("denied GET Allow-Origin = %q, want empty", got)
	}
}

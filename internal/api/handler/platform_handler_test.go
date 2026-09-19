package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/service"
)

func newLoginTestHandler(t *testing.T, params auth.ThrottleParams) *PlatformHandler {
	t.Helper()
	st := store.NewMemory()
	authSvc := auth.NewService("test-secret", 3600)
	svc := service.NewPlatformService(st, nil, nil, nil)
	th := auth.NewLoginThrottle(params)
	t.Cleanup(th.Stop)
	return NewPlatformHandler(authSvc, st, svc, th)
}

func saveLoginUser(t *testing.T, h *PlatformHandler, username, password, state string) {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	h.store.SaveUser(&platform.User{
		ID:           store.NewID(),
		Username:     username,
		PasswordHash: hash,
		Role:         platform.RoleUser,
		State:        state,
		CreatedAt:    time.Now(),
	})
}

func doLogin(t *testing.T, h *PlatformHandler, username, password, realIP string) *httptest.ResponseRecorder {
	t.Helper()
	body := fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	if realIP != "" {
		req.Header.Set("X-Real-IP", realIP)
	}
	rec := httptest.NewRecorder()
	h.Login(rec, req)
	return rec
}

func TestLoginMissingUserAndWrongPasswordAreIndistinguishable(t *testing.T) {
	h := newLoginTestHandler(t, auth.ThrottleParams{})
	saveLoginUser(t, h, "alice", "correct-password", "active")

	missing := doLogin(t, h, "bob", "any-password", "")
	wrong := doLogin(t, h, "alice", "wrong-password", "")

	if missing.Code != http.StatusUnauthorized || wrong.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d/%d, want 401/401", missing.Code, wrong.Code)
	}
	if missing.Body.String() != wrong.Body.String() {
		t.Fatalf("bodies differ: missing user=%q wrong password=%q",
			missing.Body.String(), wrong.Body.String())
	}
}

func TestLoginDisabledUserIsRejectedWithoutToken(t *testing.T) {
	h := newLoginTestHandler(t, auth.ThrottleParams{})
	saveLoginUser(t, h, "disabled-user", "correct-password", "disabled")
	saveLoginUser(t, h, "active-user", "correct-password", "active")

	disabled := doLogin(t, h, "disabled-user", "correct-password", "")
	wrong := doLogin(t, h, "active-user", "wrong-password", "")

	if disabled.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", disabled.Code)
	}
	if disabled.Body.String() != wrong.Body.String() {
		t.Fatalf("disabled body = %q, want identical to wrong password %q",
			disabled.Body.String(), wrong.Body.String())
	}
	if strings.Contains(disabled.Body.String(), "token") {
		t.Fatalf("disabled login response must not contain a token: %q", disabled.Body.String())
	}
}

func TestLoginSuccess(t *testing.T) {
	h := newLoginTestHandler(t, auth.ThrottleParams{})
	saveLoginUser(t, h, "alice", "correct-password", "active")

	rec := doLogin(t, h, "alice", "correct-password", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if token, _ := body["token"].(string); token == "" {
		t.Fatalf("expected token in response: %v", body)
	}
}

func TestLoginThrottledAfterRepeatedFailures(t *testing.T) {
	params := auth.ThrottleParams{
		UserMaxFailures: 3,
		IPMaxFailures:   50,
		Window:          10 * time.Minute,
		Lockout:         5 * time.Minute,
	}
	h := newLoginTestHandler(t, params)
	saveLoginUser(t, h, "alice", "correct-password", "active")

	for i := 0; i < 3; i++ {
		rec := doLogin(t, h, "alice", "wrong-password", "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status = %d, want 401", i+1, rec.Code)
		}
	}
	blocked := doLogin(t, h, "alice", "wrong-password", "")
	if blocked.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", blocked.Code)
	}
	if blocked.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header on throttled response")
	}
	// A different username is not affected by alice's throttle.
	other := doLogin(t, h, "bob", "any-password", "")
	if other.Code != http.StatusUnauthorized {
		t.Fatalf("other username status = %d, want 401", other.Code)
	}
}

func TestLoginThrottleTracksSeparateClientIPs(t *testing.T) {
	params := auth.ThrottleParams{
		UserMaxFailures: 100,
		IPMaxFailures:   2,
		Window:          10 * time.Minute,
		Lockout:         5 * time.Minute,
	}
	h := newLoginTestHandler(t, params)
	saveLoginUser(t, h, "alice", "correct-password", "active")

	doLogin(t, h, "alice", "wrong-password", "203.0.113.1")
	doLogin(t, h, "alice", "wrong-password", "203.0.113.1")
	blocked := doLogin(t, h, "alice", "wrong-password", "203.0.113.1")
	if blocked.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429 for repeated client IP", blocked.Code)
	}
	// The same username from a fresh IP is not IP-throttled.
	other := doLogin(t, h, "alice", "wrong-password", "203.0.113.2")
	if other.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for fresh client IP", other.Code)
	}
}

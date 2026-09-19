package auth

import (
	"net/http/httptest"
	"testing"
	"time"
)

func testParams() ThrottleParams {
	return ThrottleParams{
		UserMaxFailures: 3,
		IPMaxFailures:   5,
		Window:          10 * time.Minute,
		Lockout:         5 * time.Minute,
	}
}

func TestNewLoginThrottleAppliesDefaults(t *testing.T) {
	th := NewLoginThrottle(ThrottleParams{})
	defer th.Stop()
	if th.userMax != DefaultUserMaxFailures || th.ipMax != DefaultIPMaxFailures {
		t.Fatalf("max failures = %d/%d, want defaults %d/%d",
			th.userMax, th.ipMax, DefaultUserMaxFailures, DefaultIPMaxFailures)
	}
	if th.window != DefaultWindow || th.lockout != DefaultLockout {
		t.Fatalf("window/lockout = %v/%v, want defaults %v/%v",
			th.window, th.lockout, DefaultWindow, DefaultLockout)
	}
}

func TestLoginThrottleAllowsUnderThreshold(t *testing.T) {
	th := NewLoginThrottle(testParams())
	defer th.Stop()
	for i := 0; i < 2; i++ {
		th.Failure("1.1.1.1", "root")
	}
	if _, ok := th.Allow("1.1.1.1", "root"); !ok {
		t.Fatal("expected attempt to be allowed under threshold")
	}
}

func TestLoginThrottleBlocksUsernameAfterMaxFailures(t *testing.T) {
	th := NewLoginThrottle(testParams())
	defer th.Stop()
	for i := 0; i < 3; i++ {
		th.Failure("1.1.1.1", "root")
	}
	retryAfter, ok := th.Allow("9.9.9.9", "root")
	if ok {
		t.Fatal("expected username to be blocked after max failures")
	}
	if retryAfter <= 0 || retryAfter > 5*time.Minute {
		t.Fatalf("retryAfter = %v, want within (0, 5m]", retryAfter)
	}
	// A different username from the same IP is still allowed.
	if _, ok := th.Allow("1.1.1.1", "alice"); !ok {
		t.Fatal("expected different username to remain allowed")
	}
}

func TestLoginThrottleBlocksIPAfterMaxFailures(t *testing.T) {
	th := NewLoginThrottle(testParams())
	defer th.Stop()
	users := []string{"a", "b", "c", "d", "e"}
	for _, u := range users {
		th.Failure("1.1.1.1", u)
	}
	if _, ok := th.Allow("1.1.1.1", "new-user"); ok {
		t.Fatal("expected IP to be blocked after max failures across usernames")
	}
	// A different IP is still allowed.
	if _, ok := th.Allow("2.2.2.2", "new-user"); !ok {
		t.Fatal("expected different IP to remain allowed")
	}
}

func TestLoginThrottleSuccessResetsUsername(t *testing.T) {
	th := NewLoginThrottle(testParams())
	defer th.Stop()
	th.Failure("1.1.1.1", "root")
	th.Failure("1.1.1.1", "root")
	th.Success("1.1.1.1", "root")
	// Counter was cleared: two more failures stay under the threshold.
	th.Failure("1.1.1.1", "root")
	th.Failure("1.1.1.1", "root")
	if _, ok := th.Allow("1.1.1.1", "root"); !ok {
		t.Fatal("expected username counter to reset after success")
	}
}

func TestLoginThrottleLockoutExpires(t *testing.T) {
	th := NewLoginThrottle(testParams())
	defer th.Stop()
	for i := 0; i < 3; i++ {
		th.Failure("1.1.1.1", "root")
	}
	if _, ok := th.Allow("1.1.1.1", "root"); ok {
		t.Fatal("expected username to be blocked")
	}
	// Simulate the lockout having expired.
	th.mu.Lock()
	th.byUser["root"].blockedUntil = time.Now().Add(-time.Second)
	th.mu.Unlock()
	if _, ok := th.Allow("1.1.1.1", "root"); !ok {
		t.Fatal("expected username to be allowed after lockout expiry")
	}
}

func TestLoginThrottleWindowResetsCounter(t *testing.T) {
	th := NewLoginThrottle(testParams())
	defer th.Stop()
	th.Failure("1.1.1.1", "root")
	th.Failure("1.1.1.1", "root")
	// Simulate the last failure being older than the window.
	th.mu.Lock()
	th.byUser["root"].lastFailure = time.Now().Add(-11 * time.Minute)
	th.mu.Unlock()
	th.Failure("1.1.1.1", "root")
	th.mu.Lock()
	count := th.byUser["root"].failures
	th.mu.Unlock()
	if count != 1 {
		t.Fatalf("failures = %d, want 1 after window expiry", count)
	}
}

func TestClientIP(t *testing.T) {
	cases := []struct {
		name       string
		remoteAddr string
		realIP     string
		xff        string
		want       string
	}{
		{"real ip header wins", "10.0.0.1:1234", "203.0.113.7", "", "203.0.113.7"},
		{"xff first entry", "10.0.0.1:1234", "", "198.51.100.3, 10.0.0.1", "198.51.100.3"},
		{"xff single", "10.0.0.1:1234", "", "198.51.100.3", "198.51.100.3"},
		{"remote addr with port", "192.0.2.9:443", "", "", "192.0.2.9"},
		{"remote addr without port", "192.0.2.9", "", "", "192.0.2.9"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
			r.RemoteAddr = tc.remoteAddr
			if tc.realIP != "" {
				r.Header.Set("X-Real-IP", tc.realIP)
			}
			if tc.xff != "" {
				r.Header.Set("X-Forwarded-For", tc.xff)
			}
			if got := ClientIP(r); got != tc.want {
				t.Fatalf("ClientIP = %q, want %q", got, tc.want)
			}
		})
	}
}

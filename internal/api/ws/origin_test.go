package ws

import (
	"net/http/httptest"
	"testing"
)

func TestOriginCheckerRejectsForeignOrigin(t *testing.T) {
	check := OriginChecker(nil)

	req := httptest.NewRequest("GET", "http://console.virtfoundry.test/ws/events", nil)
	req.Host = "console.virtfoundry.test"
	req.Header.Set("Origin", "https://evil.example.com")

	if check(req) {
		t.Fatal("foreign origin accepted; CheckOrigin is allow-all")
	}
}

func TestOriginCheckerAcceptsSameOrigin(t *testing.T) {
	check := OriginChecker(nil)

	req := httptest.NewRequest("GET", "http://console.virtfoundry.test/ws/events", nil)
	req.Host = "console.virtfoundry.test"
	req.Header.Set("Origin", "https://console.virtfoundry.test")

	if !check(req) {
		t.Fatal("same-origin UI request rejected")
	}
}

func TestOriginCheckerAcceptsConfiguredOrigin(t *testing.T) {
	check := OriginChecker([]string{"https://ui.virtfoundry.test"})

	req := httptest.NewRequest("GET", "http://api.virtfoundry.test/ws/events", nil)
	req.Host = "api.virtfoundry.test"
	req.Header.Set("Origin", "https://ui.virtfoundry.test")

	if !check(req) {
		t.Fatal("configured UI origin rejected")
	}
}

func TestOriginCheckerRejectsOriginNotInAllowList(t *testing.T) {
	check := OriginChecker([]string{"https://ui.virtfoundry.test"})

	req := httptest.NewRequest("GET", "http://api.virtfoundry.test/ws/events", nil)
	req.Host = "api.virtfoundry.test"
	req.Header.Set("Origin", "https://ui.virtfoundry.test.evil.example.com")

	if check(req) {
		t.Fatal("origin with allowed host as prefix accepted")
	}
}

func TestOriginCheckerAllowsNonBrowserClient(t *testing.T) {
	check := OriginChecker(nil)

	req := httptest.NewRequest("GET", "http://api.virtfoundry.test/ws/events", nil)
	req.Host = "api.virtfoundry.test"

	if !check(req) {
		t.Fatal("request without Origin rejected; CLI clients cannot connect")
	}
}

func TestScopeAllows(t *testing.T) {
	cases := []struct {
		name  string
		scope Scope
		event string
		want  bool
	}{
		{"same tenant", Scope{TenantID: "tenant-a"}, "tenant-a", true},
		{"other tenant", Scope{TenantID: "tenant-a"}, "tenant-b", false},
		{"root sees all", Scope{AllTenants: true}, "tenant-b", true},
		{"empty scope matches nothing", Scope{}, "tenant-a", false},
		{"empty tenant event", Scope{TenantID: "tenant-a"}, "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.scope.allows(tc.event); got != tc.want {
				t.Fatalf("allows(%q) = %v, want %v", tc.event, got, tc.want)
			}
		})
	}
}

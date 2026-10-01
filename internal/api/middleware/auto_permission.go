package middleware

import (
	"net/http"
	"strings"

	"github.com/virtfoundry/core/internal/auth"
)

var resourcePermMap = map[string]string{
	"tenants":           "tenants",
	"users":             "users",
	"roles":             "users",
	"api-keys":          "users",
	"vpcs":              "vpcs",
	"networks":          "networks",
	"security-groups":   "security_groups",
	"volumes":           "volumes",
	"snapshots":         "volumes",
	"load-balancers":    "networks",
	"target-groups":     "networks",
	"vms":               "vms",
	"vm-templates":      "vms",
	"vm-snapshots":      "vms",
	"ssh-keys":          "ssh_keys",
	"service-offerings": "vms",
	"auth":              "users",
	"vks":               "vks",
}

// routes that aggregate multiple resources; authz is enforced inside handlers
// (permission-filtered payloads), not via a single resourcePermMap entry.
var handlerAuthzBypass = map[string]bool{
	"/api/v1/auth/me":           true,
	"/api/v1/dashboard/summary": true,
	"/api/v1/search":            true,
	"/api/v1/notifications":     true,
}

// AutoPermission enforces <resource>:read|write from URL path and HTTP method.
// Unmapped API segments are denied (fail-closed). /api-keys self-service and
// handler-authz routes are explicit allows.
func AutoPermission(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handlerAuthzBypass[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}
		actor := GetActor(r.Context())
		if actor == nil {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) < 3 {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		segment := parts[2]
		// API key self-service: any authenticated principal may manage own keys;
		// admin cross-user revoke is gated inside the handler/service.
		if segment == "api-keys" {
			next.ServeHTTP(w, r)
			return
		}
		base, ok := resourcePermMap[segment]
		if !ok {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		action := "read"
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			action = "write"
		}
		perm := base + ":" + action
		if !auth.HasPermission(actor.Permissions, perm) {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

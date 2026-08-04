package middleware

import (
	"net/http"
	"strings"

	"github.com/virtfoundry/core/internal/auth"
)

var resourcePermMap = map[string]string{
	"tenants":          "tenants",
	"users":            "users",
	"roles":            "users",
	"api-keys":         "users",
	"vpcs":             "vpcs",
	"networks":         "networks",
	"security-groups":  "security_groups",
	"volumes":          "volumes",
	"snapshots":        "volumes",
	"vms":              "vms",
	"vm-templates":     "vms",
	"vm-snapshots":     "vms",
	"ssh-keys":         "ssh_keys",
	"service-offerings": "vms",
	"auth":             "users",
}

// AutoPermission enforces <resource>:read|write from URL path and HTTP method.
func AutoPermission(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/me" ||
			r.URL.Path == "/api/v1/dashboard/summary" ||
			r.URL.Path == "/api/v1/search" ||
			r.URL.Path == "/api/v1/notifications" {
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
			next.ServeHTTP(w, r)
			return
		}
		segment := parts[2]
		if segment == "api-keys" {
			next.ServeHTTP(w, r)
			return
		}
		base, ok := resourcePermMap[segment]
		if !ok {
			next.ServeHTTP(w, r)
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

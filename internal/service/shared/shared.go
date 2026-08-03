package shared

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/virtfoundry/core/internal/platform/store"
)

// EventBroadcaster pushes realtime events to WebSocket clients.
type EventBroadcaster interface {
	Broadcast(eventType string, payload interface{})
}

// TenantNamespace resolves the K8s namespace for a tenant ID.
func TenantNamespace(st store.Repository, tenantID string) (string, error) {
	t, ok := st.GetTenant(tenantID)
	if !ok {
		return "", fmt.Errorf("tenant not found")
	}
	return t.Namespace, nil
}

var slugRe = regexp.MustCompile(`[^a-z0-9-]+`)

// SanitizeSlug normalizes resource names for K8s and KubeVirt.
func SanitizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "_", "-")
	s = slugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

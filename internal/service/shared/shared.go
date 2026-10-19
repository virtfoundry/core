package shared

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/virtfoundry/core/internal/platform/store"
)

// EventBroadcaster pushes realtime events to WebSocket clients. Every event
// carries the owning tenant so the hub can keep streams tenant-isolated.
type EventBroadcaster interface {
	BroadcastTenant(tenantID, eventType string, payload interface{})
}

// TenantNamespace resolves the K8s namespace for a tenant ID.
func TenantNamespace(st store.Repository, tenantID string) (string, error) {
	t, ok := st.GetTenant(tenantID)
	if !ok {
		return "", fmt.Errorf("tenant not found")
	}
	return t.Namespace, nil
}

// NormalizeTags trims whitespace, drops empty entries and removes duplicates
// while preserving first-occurrence order. Returns nil when nothing remains so
// the field is omitted from JSON, matching the VM tags convention.
func NormalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

var slugRe = regexp.MustCompile(`[^a-z0-9-]+`)

// SanitizeSlug normalizes resource names for K8s and KubeVirt.
func SanitizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "_", "-")
	s = slugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

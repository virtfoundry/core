package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/virtfoundry/core/internal/auth"
	"github.com/virtfoundry/core/internal/platform"
)

// DashboardSummary aggregates tenant overview for the UI dashboard.
type DashboardSummary struct {
	VMs            DashboardResourceCount `json:"vms"`
	Volumes        DashboardResourceCount `json:"volumes"`
	VPCs           DashboardResourceCount `json:"vpcs"`
	Networks       DashboardResourceCount `json:"networks"`
	SecurityGroups DashboardResourceCount `json:"security_groups"`
	Health         string                 `json:"health"`
	RecentActivity []DashboardActivity    `json:"recent_activity"`
}

type DashboardResourceCount struct {
	Total   int `json:"total"`
	Running int `json:"running,omitempty"`
	Error   int `json:"error,omitempty"`
}

type DashboardActivity struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
	State       string `json:"state"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	Path        string `json:"path"`
}

type SearchHit struct {
	Type     string `json:"type"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	Subtitle string `json:"subtitle,omitempty"`
	Path     string `json:"path"`
}

type NotificationItem struct {
	ID        string `json:"id"`
	Level     string `json:"level"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	Path      string `json:"path,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

func (s *PlatformService) DashboardSummary(ctx context.Context, tenantID string) (*DashboardSummary, error) {
	vms, err := s.ListVMs(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	vmCount := countVMStates(vms)
	summary := &DashboardSummary{
		VMs: DashboardResourceCount{
			Total:   len(vms),
			Running: vmCount.running,
			Error:   vmCount.errors,
		},
		Volumes:        DashboardResourceCount{Total: len(s.ListVolumes(tenantID))},
		VPCs:           DashboardResourceCount{Total: len(s.ListVPCs(tenantID))},
		Networks:       DashboardResourceCount{Total: len(s.ListNetworks(tenantID))},
		SecurityGroups: DashboardResourceCount{Total: len(s.ListSecurityGroups(tenantID))},
		Health:         dashboardHealth(vmCount.errors, vmCount.transitional),
		RecentActivity: recentVMActivity(vms, 8),
	}
	return summary, nil
}

func (s *PlatformService) Search(ctx context.Context, tenantID, query string, perms []string) []SearchHit {
	q := strings.TrimSpace(strings.ToLower(query))
	if len(q) < 2 {
		return []SearchHit{}
	}

	var hits []SearchHit
	if auth.HasPermission(perms, auth.PermVMsRead) {
		if vms, err := s.ListVMs(ctx, tenantID); err == nil {
			for _, vm := range vms {
				if vm == nil {
					continue
				}
				if matchesQuery(q, vm.Name, vm.DisplayName, vm.IP) {
					hits = append(hits, SearchHit{
						Type:     "vm",
						ID:       vm.ID,
						Name:     vm.Name,
						Subtitle: vmStateSubtitle(vm),
						Path:     "/vms/" + vm.Name,
					})
				}
			}
		}
	}
	if auth.HasPermission(perms, auth.PermVolumesRead) {
		for _, vol := range s.ListVolumes(tenantID) {
			if vol != nil && matchesQuery(q, vol.Name, vol.PVCName) {
				hits = append(hits, SearchHit{
					Type:     "volume",
					ID:       vol.ID,
					Name:     vol.Name,
					Subtitle: vol.PVCName,
					Path:     "/volumes",
				})
			}
		}
	}
	if auth.HasPermission(perms, auth.PermVPCsRead) {
		for _, vpc := range s.ListVPCs(tenantID) {
			if vpc != nil && matchesQuery(q, vpc.Name, vpc.CIDR) {
				hits = append(hits, SearchHit{
					Type:     "vpc",
					ID:       vpc.ID,
					Name:     vpc.Name,
					Subtitle: vpc.CIDR,
					Path:     "/vpcs",
				})
			}
		}
	}
	if auth.HasPermission(perms, auth.PermNetworksRead) {
		for _, net := range s.ListNetworks(tenantID) {
			if net != nil && matchesQuery(q, net.Name, net.CIDR) {
				hits = append(hits, SearchHit{
					Type:     "network",
					ID:       net.ID,
					Name:     net.Name,
					Subtitle: net.CIDR,
					Path:     "/networks",
				})
			}
		}
	}
	if auth.HasPermission(perms, auth.PermSecurityGroupsRead) {
		for _, sg := range s.ListSecurityGroups(tenantID) {
			if sg != nil && matchesQuery(q, sg.Name, sg.Description) {
				hits = append(hits, SearchHit{
					Type:     "security_group",
					ID:       sg.ID,
					Name:     sg.Name,
					Subtitle: sg.Description,
					Path:     "/security-groups",
				})
			}
		}
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Type == hits[j].Type {
			return hits[i].Name < hits[j].Name
		}
		return hits[i].Type < hits[j].Type
	})
	if len(hits) > 20 {
		hits = hits[:20]
	}
	return hits
}

func (s *PlatformService) Notifications(ctx context.Context, tenantID string) []NotificationItem {
	vms, err := s.ListVMs(ctx, tenantID)
	if err != nil {
		return []NotificationItem{}
	}

	var items []NotificationItem
	for _, vm := range vms {
		if vm == nil {
			continue
		}
		state := strings.ToLower(vm.State)
		switch state {
		case "error":
			msg := vm.ErrorMsg
			if msg == "" {
				msg = "VM reported an error state"
			}
			items = append(items, NotificationItem{
				ID:        "vm-error-" + vm.Name,
				Level:     "error",
				Title:     vm.Name,
				Message:   msg,
				Path:      "/vms/" + vm.Name,
				CreatedAt: formatTime(vm.UpdatedAt),
			})
		case "starting", "stopping":
			items = append(items, NotificationItem{
				ID:        "vm-trans-" + vm.Name + "-" + state,
				Level:     "warning",
				Title:     vm.Name,
				Message:   "VM is " + state,
				Path:      "/vms/" + vm.Name,
				CreatedAt: formatTime(vm.UpdatedAt),
			})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		levelRank := map[string]int{"error": 0, "warning": 1, "info": 2}
		if levelRank[items[i].Level] != levelRank[items[j].Level] {
			return levelRank[items[i].Level] < levelRank[items[j].Level]
		}
		return items[i].Title < items[j].Title
	})
	if len(items) > 15 {
		items = items[:15]
	}
	return items
}

type vmStateTally struct {
	running      int
	errors       int
	transitional int
}

func countVMStates(vms []*platform.PlatformVM) vmStateTally {
	var tally vmStateTally
	for _, vm := range vms {
		if vm == nil {
			continue
		}
		switch strings.ToLower(vm.State) {
		case "running":
			tally.running++
		case "error":
			tally.errors++
		case "starting", "stopping":
			tally.transitional++
		}
	}
	return tally
}

func dashboardHealth(errors, transitional int) string {
	if errors > 0 {
		return "critical"
	}
	if transitional > 0 {
		return "warning"
	}
	return "ok"
}

func recentVMActivity(vms []*platform.PlatformVM, limit int) []DashboardActivity {
	sorted := make([]*platform.PlatformVM, 0, len(vms))
	for _, vm := range vms {
		if vm != nil {
			sorted = append(sorted, vm)
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].UpdatedAt.After(sorted[j].UpdatedAt)
	})
	if len(sorted) > limit {
		sorted = sorted[:limit]
	}
	out := make([]DashboardActivity, 0, len(sorted))
	for _, vm := range sorted {
		out = append(out, DashboardActivity{
			Type:        "vm",
			Name:        vm.Name,
			DisplayName: vm.DisplayName,
			State:       vm.State,
			UpdatedAt:   formatTime(vm.UpdatedAt),
			Path:        "/vms/" + vm.Name,
		})
	}
	return out
}

func matchesQuery(q string, fields ...string) bool {
	for _, f := range fields {
		if f != "" && strings.Contains(strings.ToLower(f), q) {
			return true
		}
	}
	return false
}

func vmStateSubtitle(vm *platform.PlatformVM) string {
	if vm.IP != "" {
		return vm.State + " · " + vm.IP
	}
	return vm.State
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

package platform

import "time"

type TargetGroup struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Protocol  string    `json:"protocol"`
	Port      int       `json:"port"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
}

type LoadBalancer struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Namespace   string    `json:"namespace"`
	ServiceName string    `json:"service_name"`
	VIP         string    `json:"vip,omitempty"`
	State       string    `json:"state"`
	CreatedAt   time.Time `json:"created_at"`
}

type LBListener struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	LoadBalancerID string    `json:"load_balancer_id"`
	Protocol       string    `json:"protocol"`
	Port           int       `json:"port"`
	TargetGroupID  string    `json:"target_group_id"`
	CreatedAt      time.Time `json:"created_at"`
}

type LBTarget struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	TargetGroupID string    `json:"target_group_id"`
	VMID          string    `json:"vm_id"`
	IP            string    `json:"ip"`
	Port          int       `json:"port"`
	State         string    `json:"state"`
	CreatedAt     time.Time `json:"created_at"`
}

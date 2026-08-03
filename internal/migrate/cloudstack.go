package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/service"
)

// CloudStackAccount represents an account/domain from CloudStack MySQL.
type CloudStackAccount struct {
	ID          string
	AccountName string
	DomainID    string
	State       string
}

// CloudStackVM represents a VM instance row from CloudStack.
type CloudStackVM struct {
	ID          string
	Name        string
	DisplayName string
	AccountID   string
	State       string
	CPU         int
	MemoryMi    int64
	Template    string
}

// CloudStackNetwork represents a guest network.
type CloudStackNetwork struct {
	ID        string
	Name      string
	AccountID string
	CIDR      string
	Type      string
}

// CloudStackVolume represents a data volume.
type CloudStackVolume struct {
	ID        string
	Name      string
	AccountID string
	VMID      string
	SizeGi    int
	State     string
}

// Report summarizes a migration run.
type Report struct {
	TenantsCreated  int
	VPCsCreated     int
	NetworksCreated int
	VMsImported     int
	VMsDeployed     int
	VolumesImported int
	Errors          []string
}

// Options configures CloudStack import.
type Options struct {
	DSN         string
	DryRun      bool
	DeployVMs   bool
	TemplateMap map[string]string // CloudStack template name → container disk image
}

// CloudStackReader reads from Apache CloudStack or compatible MySQL schema.
type CloudStackReader struct {
	db *sql.DB
}

func OpenCloudStack(dsn string) (*CloudStackReader, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("cloudstack db ping: %w", err)
	}
	return &CloudStackReader{db: db}, nil
}

func (r *CloudStackReader) Close() error {
	return r.db.Close()
}

func (r *CloudStackReader) ListAccounts(ctx context.Context) ([]CloudStackAccount, error) {
	queries := []string{
		`SELECT CAST(id AS CHAR), account_name, CAST(domain_id AS CHAR), state
		 FROM account WHERE removed IS NULL AND type IN (0, 2) ORDER BY account_name`,
		`SELECT id, account_name, domain_id, state
		 FROM account WHERE removed IS NULL ORDER BY account_name`,
	}
	for _, q := range queries {
		rows, err := r.db.QueryContext(ctx, q)
		if err != nil {
			continue
		}
		defer rows.Close()
		var out []CloudStackAccount
		for rows.Next() {
			var a CloudStackAccount
			if err := rows.Scan(&a.ID, &a.AccountName, &a.DomainID, &a.State); err != nil {
				return nil, err
			}
			out = append(out, a)
		}
		if len(out) > 0 {
			return out, rows.Err()
		}
	}
	return nil, fmt.Errorf("could not read accounts table (unsupported CloudStack schema?)")
}

func (r *CloudStackReader) ListVMs(ctx context.Context) ([]CloudStackVM, error) {
	queries := []string{
		`SELECT CAST(v.id AS CHAR), v.name, COALESCE(v.display_name, v.name),
		        CAST(v.account_id AS CHAR), v.state, COALESCE(v.cpu, 1), COALESCE(v.ram / 1024 / 1024, 1024),
		        COALESCE(t.name, '')
		 FROM vm_instance v
		 LEFT JOIN vm_template t ON v.vm_template_id = t.id
		 WHERE v.removed IS NULL AND v.type = 'User'
		 ORDER BY v.name`,
		`SELECT id, name, COALESCE(display_name, name), account_id, state,
		        COALESCE(cpu_count, 1), COALESCE(memory / 1024 / 1024, 1024), ''
		 FROM vm_instance WHERE removed IS NULL ORDER BY name`,
	}
	for _, q := range queries {
		rows, err := r.db.QueryContext(ctx, q)
		if err != nil {
			continue
		}
		defer rows.Close()
		var out []CloudStackVM
		for rows.Next() {
			var vm CloudStackVM
			if err := rows.Scan(&vm.ID, &vm.Name, &vm.DisplayName, &vm.AccountID, &vm.State, &vm.CPU, &vm.MemoryMi, &vm.Template); err != nil {
				return nil, err
			}
			out = append(out, vm)
		}
		if len(out) > 0 {
			return out, rows.Err()
		}
	}
	return nil, fmt.Errorf("could not read vm_instance table")
}

func (r *CloudStackReader) ListNetworks(ctx context.Context) ([]CloudStackNetwork, error) {
	queries := []string{
		`SELECT CAST(id AS CHAR), name, CAST(account_id AS CHAR), COALESCE(cidr, ''), COALESCE(traffic_type, 'Guest')
		 FROM networks WHERE removed IS NULL ORDER BY name`,
		`SELECT id, name, account_id, COALESCE(cidr, ''), COALESCE(type, 'Guest')
		 FROM networks WHERE removed IS NULL ORDER BY name`,
	}
	for _, q := range queries {
		rows, err := r.db.QueryContext(ctx, q)
		if err != nil {
			continue
		}
		defer rows.Close()
		var out []CloudStackNetwork
		for rows.Next() {
			var n CloudStackNetwork
			if err := rows.Scan(&n.ID, &n.Name, &n.AccountID, &n.CIDR, &n.Type); err != nil {
				return nil, err
			}
			out = append(out, n)
		}
		if len(out) > 0 {
			return out, rows.Err()
		}
	}
	return nil, nil // networks optional
}

// ImportCloudStack imports CloudStack metadata into VirtFoundry via PlatformService.
func ImportCloudStack(ctx context.Context, reader *CloudStackReader, svc *service.PlatformService, opts Options) (*Report, error) {
	report := &Report{}

	accounts, err := reader.ListAccounts(ctx)
	if err != nil {
		return report, err
	}
	vms, err := reader.ListVMs(ctx)
	if err != nil {
		return report, err
	}
	networks, _ := reader.ListNetworks(ctx)

	if opts.DryRun {
		for _, acc := range accounts {
			slug := sanitizeSlug(acc.AccountName)
			if slug == "" || slug == "admin" || slug == "system" {
				continue
			}
			report.TenantsCreated++
			report.VPCsCreated++
		}
		report.NetworksCreated = len(networks)
		report.VMsImported = len(vms)
		if opts.DeployVMs {
			for _, vm := range vms {
				if vm.State == "Running" || vm.State == "Stopped" || vm.State == "Starting" {
					report.VMsDeployed++
				}
			}
		}
		return report, nil
	}

	if svc == nil {
		return report, fmt.Errorf("platform service required when not dry-run")
	}

	accountTenants := map[string]string{}

	for _, acc := range accounts {
		slug := sanitizeSlug(acc.AccountName)
		if slug == "" || slug == "admin" || slug == "system" {
			continue
		}
		tenant, _, err := svc.CreateTenant(ctx, acc.AccountName, slug, "migrate-"+slug)
		if err != nil {
			if strings.Contains(err.Error(), "already exists") {
				if t, ok := findTenantBySlug(svc, slug); ok {
					accountTenants[acc.ID] = t.ID
				}
				continue
			}
			report.Errors = append(report.Errors, fmt.Sprintf("tenant %s: %v", acc.AccountName, err))
			continue
		}
		accountTenants[acc.ID] = tenant.ID
		report.TenantsCreated++

		if _, err := svc.CreateVPC(ctx, tenant.ID, "migrated-vpc", "10.200.0.0/16"); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("vpc %s: %v", acc.AccountName, err))
		} else {
			report.VPCsCreated++
		}
	}

	for _, net := range networks {
		tid, ok := accountTenants[net.AccountID]
		if !ok {
			continue
		}
		cidr := net.CIDR
		if cidr == "" {
			cidr = "10.200.1.0/24"
		}
		vpcs := svc.ListVPCs(tid)
		vpcID := ""
		if len(vpcs) > 0 {
			vpcID = vpcs[0].ID
		}
		if _, err := svc.CreateNetwork(ctx, tid, vpcID, net.Name, cidr, 0); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("network %s: %v", net.Name, err))
		} else {
			report.NetworksCreated++
		}
	}

	for _, vm := range vms {
		tid, ok := accountTenants[vm.AccountID]
		if !ok {
			continue
		}
		report.VMsImported++
		if !opts.DeployVMs {
			continue
		}
		if vm.State != "Running" && vm.State != "Stopped" && vm.State != "Starting" {
			continue
		}
		image := resolveTemplate(vm.Template, opts.TemplateMap)
		name := sanitizeSlug(vm.Name)
		if name == "" {
			name = sanitizeSlug(vm.DisplayName)
		}
		_, err := svc.DeployVM(ctx, tid, service.PlatformDeployVMInput{
			Name: name, Image: image, CPU: vm.CPU, MemoryMi: vm.MemoryMi, Start: vm.State == "Running",
		})
		if err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("vm %s: %v", vm.Name, err))
			continue
		}
		report.VMsDeployed++
	}

	return report, nil
}

func findTenantBySlug(svc *service.PlatformService, slug string) (*platform.Tenant, bool) {
	for _, t := range svc.ListTenants() {
		if t.Slug == slug {
			return t, true
		}
	}
	return nil, false
}

func resolveTemplate(template string, m map[string]string) string {
	if img, ok := m[template]; ok {
		return img
	}
	return "quay.io/kubevirt/cirros-container-disk-demo"
}

func sanitizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "_", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else if r == ' ' || r == '.' {
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

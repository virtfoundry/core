package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/virtforge-cloud/virtforge/internal/config"
	"github.com/virtforge-cloud/virtforge/internal/platform/branding"
	"github.com/virtforge-cloud/virtforge/internal/infra/hypervisor"
	"github.com/virtforge-cloud/virtforge/internal/migrate"
	platformk8s "github.com/virtforge-cloud/virtforge/internal/platform/k8s"
	"github.com/virtforge-cloud/virtforge/internal/platform/store"
	"github.com/virtforge-cloud/virtforge/internal/service"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "cloudstack":
		runCloudStack(os.Args[2:])
	default:
		printUsage()
		os.Exit(1)
	}
}

func runCloudStack(args []string) {
	fs := flag.NewFlagSet("cloudstack", flag.ExitOnError)
	dsn := fs.String("dsn", "", "CloudStack MySQL DSN (user:pass@tcp(host:3306)/cloud)")
	dryRun := fs.Bool("dry-run", false, "Report only, do not create resources")
	deployVMs := fs.Bool("deploy-vms", false, "Deploy VMs in KubeVirt (metadata-only by default)")
	templateMap := fs.String("template-map", "", "Comma-separated template=image pairs")
	rootPassword := fs.String("root-password", branding.DefaultRootPassword, "Bootstrap root password if needed")
	_ = fs.Parse(args)

	if *dsn == "" {
		fmt.Fprintln(os.Stderr, "error: --dsn required")
		os.Exit(1)
	}

	templates := parseTemplateMap(*templateMap)

	reader, err := migrate.OpenCloudStack(*dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect: %v\n", err)
		os.Exit(1)
	}
	defer reader.Close()

	ctx := context.Background()

	if *dryRun {
		report, err := migrate.ImportCloudStack(ctx, reader, nil, migrate.Options{
			DSN: *dsn, DryRun: true, DeployVMs: *deployVMs, TemplateMap: templates,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "dry-run error: %v\n", err)
			os.Exit(1)
		}
		printReport(report)
		return
	}

	cfg := loadConfig()
	k8sMgr, err := platformk8s.NewManager(platformk8s.Options{Kubeconfig: cfg.KubeVirt.Kubeconfig})
	if err != nil {
		fmt.Fprintf(os.Stderr, "k8s: %v\n", err)
		os.Exit(1)
	}
	kvDriver, err := hypervisor.NewKubeVirtDriver(hypervisor.KubeVirtConfig{
		Kubeconfig: cfg.KubeVirt.Kubeconfig,
		Namespace:  cfg.KubeVirt.Namespace,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "kubevirt: %v\n", err)
		os.Exit(1)
	}

	memStore := store.NewMemory()
	svc := service.NewPlatformService(memStore, k8sMgr, kvDriver, nil)
	if !memStore.HasRootUser() {
		if _, err := svc.BootstrapRoot("root", *rootPassword); err != nil {
			fmt.Fprintf(os.Stderr, "bootstrap: %v\n", err)
			os.Exit(1)
		}
	}

	report, err := migrate.ImportCloudStack(ctx, reader, svc, migrate.Options{
		DSN: *dsn, DryRun: false, DeployVMs: *deployVMs, TemplateMap: templates,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "import: %v\n", err)
		os.Exit(1)
	}
	printReport(report)
}

func printReport(r *migrate.Report) {
	fmt.Println("=== VirtForge CloudStack Migration ===")
	fmt.Printf("Tenants:  %d\n", r.TenantsCreated)
	fmt.Printf("VPCs:     %d\n", r.VPCsCreated)
	fmt.Printf("Networks: %d\n", r.NetworksCreated)
	fmt.Printf("VMs meta: %d\n", r.VMsImported)
	fmt.Printf("VMs deployed: %d\n", r.VMsDeployed)
	if len(r.Errors) > 0 {
		fmt.Println("\nWarnings:")
		for _, e := range r.Errors {
			fmt.Printf("  - %s\n", e)
		}
	}
}

func parseTemplateMap(s string) map[string]string {
	out := map[string]string{}
	for _, pair := range strings.Split(s, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			out[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return out
}

func loadConfig() *config.Config {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "config.yaml"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return config.DefaultConfig()
	}
	return cfg
}

func printUsage() {
	fmt.Println(`VirtForge migration tool

Usage:
  go run ./cmd/migrate cloudstack --dsn "user:pass@tcp(host:3306)/cloud" [options]

Options:
  --dry-run          Report counts without creating resources
  --deploy-vms       Recreate VMs in KubeVirt (requires kubeconfig)
  --template-map     e.g. "CentOS 7=quay.io/kubevirt/cirros-container-disk-demo"
  --root-password    Root bootstrap password (default: virtforge)`)
}

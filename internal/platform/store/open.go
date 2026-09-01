package store

import (
	"os"
	"strings"

	"github.com/virtfoundry/core/internal/config"
)

// Open returns the configured platform store backend.
func Open(cfg config.DatabaseConfig) (Repository, error) {
	driver := os.Getenv("VIRTFOUNDRY_STORE")
	if driver == "" {
		driver = cfg.Driver
	}
	if strings.EqualFold(driver, "kubernetes") {
		kubeconfig := cfg.Kubeconfig
		if kubeconfig == "" {
			kubeconfig = os.Getenv("KUBECONFIG")
		}
		repo, err := NewKubernetes(KubernetesOptions{Kubeconfig: kubeconfig})
		if err != nil {
			return nil, err
		}
		_ = SeedCatalog(repo)
		_ = repo.SeedIAM()
		return repo, nil
	}

	dsn := cfg.DSN
	if dsn == "" {
		dsn = os.Getenv("DATABASE_DSN")
	}
	if dsn != "" {
		return NewMySQL(dsn)
	}
	mem := NewMemory()
	_ = SeedCatalog(mem)
	_ = mem.SeedIAM()
	return mem, nil
}

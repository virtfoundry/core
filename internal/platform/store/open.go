package store

import (
	"os"
	"strings"

	"github.com/virtfoundry/core/internal/config"
)

// Open returns the configured platform store backend (kubernetes or memory).
func Open(cfg config.Config) (Repository, error) {
	driver := os.Getenv("VIRTFOUNDRY_STORE")
	if driver == "" {
		driver = cfg.Database.Driver
	}
	// Password for seed Linux templates is opt-in only. Never fall back to a
	// published default such as "ubuntu" (issue #97).
	defaultPassword := cfg.VM.DefaultPassword
	if strings.EqualFold(driver, "kubernetes") {
		kubeconfig := cfg.Database.Kubeconfig
		if kubeconfig == "" {
			kubeconfig = os.Getenv("KUBECONFIG")
		}
		repo, err := NewKubernetes(KubernetesOptions{Kubeconfig: kubeconfig})
		if err != nil {
			return nil, err
		}
		_ = SeedCatalog(repo, defaultPassword)
		_ = repo.SeedIAM()
		return repo, nil
	}

	mem := NewMemory()
	_ = SeedCatalog(mem, defaultPassword)
	_ = mem.SeedIAM()
	return mem, nil
}
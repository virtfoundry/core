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
	// Password is safe to be a known default ("ubuntu") because the seed
	// CloudInitNoCloud payload sets chpasswd.expire: True, forcing a password
	// change on first login via PAM.
	defaultPassword := cfg.VM.DefaultPassword
	if defaultPassword == "" {
		defaultPassword = config.BuiltinDefaultVMPassword
	}
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
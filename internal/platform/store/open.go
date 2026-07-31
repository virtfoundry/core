package store

import (
	"os"

	"github.com/virtforge-cloud/virtforge/internal/config"
)

// Open returns MySQL when database.dsn is set, otherwise in-memory store.
func Open(cfg config.DatabaseConfig) (Repository, error) {
	dsn := cfg.DSN
	if dsn == "" {
		dsn = os.Getenv("DATABASE_DSN")
	}
	if dsn != "" {
		return NewMySQL(dsn)
	}
	mem := NewMemory()
	_ = SeedCatalog(mem)
	return mem, nil
}

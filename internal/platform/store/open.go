package store

import (
	"os"

	"github.com/virtfoundry/core/internal/config"
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
	_ = mem.SeedIAM()
	return mem, nil
}

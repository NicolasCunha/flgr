// Command server is flgr-server's entrypoint: it wires configuration, the
// database, and the HTTP router together, per docs/architecture/adr/0001
// through 0012. It is intentionally thin — the pieces it wires are each
// tested on their own (internal/config, internal/database, internal/api).
package main

import (
	"log"

	"github.com/NicolasCunha/flgr/flgr-server/internal/api"
	"github.com/NicolasCunha/flgr/flgr-server/internal/config"
	"github.com/NicolasCunha/flgr/flgr-server/internal/database"
)

// main itself is not unit tested, per the exception in
// docs/architecture/adr/0004-testing-and-coverage-standards.md: every
// call here either terminates the process (log.Fatalf) or blocks
// (router.Run), and every piece it wires (config.Load, database.Open,
// database.Migrate, api.NewRouter) is already tested on its own.
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("closing database: %v", err)
		}
	}()

	if err := database.Migrate(db, "migrations"); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	router := api.NewRouter()

	log.Printf("flgr-server listening on :%s", cfg.HTTPPort)
	if err := router.Run(":" + cfg.HTTPPort); err != nil {
		log.Fatalf("server: %v", err)
	}
}

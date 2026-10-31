package migrate

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func Up(dsn, path string, log *slog.Logger) error {
	migration, err := migrate.New("file://"+path, dsn)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	defer migration.Close()

	err = migration.Up()
	switch {
		case errors.Is(err, migrate.ErrNoChange):
			log.Info("migrations: nothing to apply")
			return nil
		case err != nil:
			return fmt.Errorf("migrate up: %w", err)
	}

	version, dirty, _ := migration.Version()
	log.Info("migrations applied", "version", version, "dirty", dirty)
	
	return nil
}
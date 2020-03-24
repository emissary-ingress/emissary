//go:generate go-bindata -pkg migrations -ignore bindata -prefix ./migrations/ -o ./migrations/bindata.go ./migrations

package datasource

import (
	"database/sql"

	migrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"

	"github.com/datawire/apro/cmd/apictl-key/datasource/migrations"
)

// targetVersion defines the current and desired migration version.
// This ensures the app is compatible with the version of the database.
const targetVersion = 2

// validateSchema migrates the Postgres schema to the current version.
func validateSchema(db *sql.DB) error {
	sourceInstance, err := bindata.WithInstance(bindata.Resource(migrations.AssetNames(), migrations.Asset))
	if err != nil {
		return err
	}

	targetInstance, err := postgres.WithInstance(db, new(postgres.Config))
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("go-bindata", sourceInstance, "postgres", targetInstance)
	if err != nil {
		return err
	}

	if err := m.Migrate(targetVersion); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return sourceInstance.Close()
}

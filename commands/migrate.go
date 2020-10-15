package commands

import (
	"github.com/filecoin-project/sentinel-visor/storage"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

var Migrate = &cli.Command{
	Name:  "migrate",
	Usage: "Reports and verifies the current database schema version and latest available for migration. Use --to or --latest to perform a schema migration.",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "to",
			Usage: "Migrate the schema to the `VERSION`.",
			Value: 0,
		},
		&cli.BoolFlag{
			Name:  "latest",
			Value: false,
			Usage: "Migrate the schema to the latest version.",
		},
		&cli.BoolFlag{
			Name:  "deferred",
			Value: false,
			Usage: "Run deferred migrations.",
		},
		&cli.IntFlag{
			Name:  "deferred-to",
			Usage: "Run deferred migrations to `VERSION`.",
			Value: 0,
		},
	},
	Action: func(cctx *cli.Context) error {
		if err := setupLogging(cctx); err != nil {
			return xerrors.Errorf("setup logging: %w", err)
		}

		ctx := cctx.Context

		db, err := storage.NewDatabase(ctx, cctx.String("db"), cctx.Int("db-pool-size"))
		if err != nil {
			return xerrors.Errorf("connect database: %w", err)
		}

		if cctx.IsSet("to") {
			if err := db.MigrateSchemaTo(ctx, cctx.Int("to")); err != nil {
				return xerrors.Errorf("migrate schema to: %w", err)
			}
		} else if cctx.Bool("latest") {
			if err := db.MigrateSchema(ctx); err != nil {
				return xerrors.Errorf("migrate schema: %w", err)
			}
		} else if cctx.Bool("deferred") {
			if err := db.RunDeferredMigrations(ctx); err != nil {
				return xerrors.Errorf("run deferred migrations: %w", err)
			}
		} else if cctx.IsSet("deferred-to") {
			if err := db.RunDeferredMigrationsTo(ctx, cctx.Int("deferred-to")); err != nil {
				return xerrors.Errorf("run deferred migrations: %w", err)
			}
		}

		dbVersion, latestVersion, err := db.GetSchemaVersions(ctx)
		if err != nil {
			return xerrors.Errorf("get schema versions: %w", err)
		}

		log.Infof("current database schema is version %d, latest is %d", dbVersion, latestVersion)

		dbDeferredVersion, deferredLatestVersion, err := db.GetDeferredSchemaVersions(ctx, dbVersion)
		if err != nil {
			return xerrors.Errorf("get deferred migrations versions: %w", err)
		}

		if dbDeferredVersion == deferredLatestVersion {
			log.Infof("deferred migrations for version %d have been applied, no further deferred migrations needed", dbDeferredVersion)
		} else {
			log.Warnf("deferred migrations for version %d have been applied, there are unapplied migrations for version %d", dbDeferredVersion, deferredLatestVersion)
			log.Infof("use `visor migrate --deferred` to run deferred migrations")
		}

		if err := db.VerifyCurrentSchema(ctx); err != nil {
			return xerrors.Errorf("verify schema: %w", err)
		}

		return nil
	},
}

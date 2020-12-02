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
	},
	Action: func(cctx *cli.Context) error {
		if err := setupLogging(cctx); err != nil {
			return xerrors.Errorf("setup logging: %w", err)
		}

		ctx := cctx.Context

		db, err := storage.NewDatabase(ctx, cctx.String("db"), cctx.Int("db-pool-size"), cctx.String("name"))
		if err != nil {
			return xerrors.Errorf("connect database: %w", err)
		}

		if cctx.IsSet("to") {
			return db.MigrateSchemaTo(ctx, cctx.Int("to"))
		}

		if cctx.Bool("latest") {
			return db.MigrateSchema(ctx)
		}

		dbVersion, latestVersion, err := db.GetSchemaVersions(ctx)
		if err != nil {
			return xerrors.Errorf("get schema versions: %w", err)
		}

		log.Infof("current database schema is version %d, latest is %d", dbVersion, latestVersion)

		if err := db.VerifyCurrentSchema(ctx); err != nil {
			return xerrors.Errorf("verify schema: %w", err)
		}

		return nil
	},
}

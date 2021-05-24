package commands

import (
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/storage"
)

var MigrateCmd = &cli.Command{
	Name:  "migrate",
	Usage: "Reports and verifies the current database schema version and latest available for migration. Use --to or --latest to perform a schema migration.",
	Flags: flagSet(
		dbConnectFlags,
		[]cli.Flag{
			&cli.StringFlag{
				Name:  "to",
				Usage: "Migrate the schema to the `VERSION`.",
				Value: "",
			},
			&cli.BoolFlag{
				Name:  "latest",
				Value: false,
				Usage: "Migrate the schema to the latest version.",
			},
		},
	),
	Action: func(cctx *cli.Context) error {
		if err := setupLogging(cctx); err != nil {
			return xerrors.Errorf("setup logging: %w", err)
		}

		ctx := cctx.Context

		db, err := storage.NewDatabase(ctx, cctx.String("db"), cctx.Int("db-pool-size"), cctx.String("name"), cctx.String("schema"), false)
		if err != nil {
			return xerrors.Errorf("connect database: %w", err)
		}

		if cctx.IsSet("to") {
			targetVersion, err := model.ParseVersion(cctx.String("to"))
			if err != nil {
				return xerrors.Errorf("invalid schema version: %w", err)
			}

			return db.MigrateSchemaTo(ctx, targetVersion)
		}

		if cctx.Bool("latest") {
			return db.MigrateSchema(ctx)
		}

		dbVersion, latestVersion, err := db.GetSchemaVersions(ctx)
		if err != nil {
			return xerrors.Errorf("get schema versions: %w", err)
		}

		log.Infof("current database schema is version %s, latest is %s", dbVersion, latestVersion)

		if err := db.VerifyCurrentSchema(ctx); err != nil {
			return xerrors.Errorf("verify schema: %w", err)
		}

		log.Infof("database schema is supported by this version of visor")
		return nil
	},
}

package commands

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/version"

	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/storage"
)

var defaultName = "visor"

func init() {
	defaultName = "visor_" + version.String()
	hostname, err := os.Hostname()
	if err == nil {
		defaultName = fmt.Sprintf("%s_%s_%d", defaultName, hostname, os.Getpid())
	}
}

type LilyDBOpts struct {
	DB                string
	Name              string
	DBSchema          string
	DBPoolSize        int
	DBAllowUpsert     bool
	DBAllowMigrations bool
}

var LilyDBFlags LilyDBOpts

var dbConnectFlags = []cli.Flag{
	&cli.StringFlag{
		Name:        "db",
		EnvVars:     []string{"LILY_DB"},
		Value:       "",
		Usage:       "A connection string for the TimescaleDB database, for example postgres://postgres:password@localhost:5432/postgres?sslmode=disable",
		Destination: &LilyDBFlags.DB,
	},
	&cli.IntFlag{
		Name:        "db-pool-size",
		EnvVars:     []string{"LILY_DB_POOL_SIZE"},
		Value:       75,
		Destination: &LilyDBFlags.DBPoolSize,
	},
	&cli.StringFlag{
		Name:        "name",
		EnvVars:     []string{"LILY_NAME"},
		Value:       defaultName,
		Usage:       "A name that helps to identify this instance of visor.",
		Destination: &LilyDBFlags.Name,
	},
	&cli.StringFlag{
		Name:        "schema",
		EnvVars:     []string{"LILY_SCHEMA"},
		Value:       "public",
		Usage:       "The name of the postgresql schema that holds the objects used by this instance of visor.",
		Destination: &LilyDBFlags.DBSchema,
	},
}

var MigrateCmd = &cli.Command{
	Name:  "migrate",
	Usage: "Manage the schema version installed in a database.",
	Flags: FlagSet(
		dbConnectFlags,
		[]cli.Flag{
			&cli.StringFlag{
				Name:  "to",
				Usage: "Migrate the schema to specific `VERSION`.",
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
		if err := setupLogging(LilyLogFlags); err != nil {
			return xerrors.Errorf("setup logging: %w", err)
		}

		ctx := cctx.Context

		db, err := storage.NewDatabase(ctx, LilyDBFlags.DB, LilyDBFlags.DBPoolSize, LilyDBFlags.Name, LilyDBFlags.DBSchema, false)
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

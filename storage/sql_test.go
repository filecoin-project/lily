package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// may want to change this to use a named schema such as lotus
const PGSchema = "public"

// resetSchema ensures the database schema is clean for a test
func resetSchema(db *Database) error {
	_, err := db.DB.Exec(fmt.Sprintf(`DROP SCHEMA %s CASCADE`, PGSchema))
	if err != nil {
		return err
	}

	_, err = db.DB.Exec(fmt.Sprintf(`CREATE SCHEMA %s`, PGSchema))
	if err != nil {
		return err
	}

	return nil
}

type column struct {
	name     string
	datatype string
	nullable bool
}

var expectedTables = []struct {
	name    string
	columns []column
}{

	{
		name: "blocks_synced",
		columns: []column{
			{
				name:     "cid",
				datatype: "text",
				nullable: false,
			},
		},
	},
	{
		name: "block_headers",
		columns: []column{
			{
				name:     "cid",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "miner",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "parent_weight",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "parent_base_fee",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "parent_state_root",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "height",
				datatype: "bigint",
				nullable: true,
			},
			{
				name:     "win_count",
				datatype: "bigint",
				nullable: true,
			},
			{
				name:     "timestamp",
				datatype: "bigint",
				nullable: true,
			},
			{
				name:     "fork_signaling",
				datatype: "bigint",
				nullable: true,
			},
			{
				name:     "ticket",
				datatype: "bytea",
				nullable: true,
			},
			{
				name:     "election_proof",
				datatype: "bytea",
				nullable: true,
			},
		},
	},
	{
		name: "blocks_synced",
		columns: []column{
			{
				name:     "cid",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "height",
				datatype: "bigint",
				nullable: true,
			},
			{
				name:     "synced_at",
				datatype: "timestamp with time zone",
				nullable: false,
			},
			{
				name:     "processed_at",
				datatype: "timestamp with time zone",
				nullable: true,
			},
			{
				name:     "completed_at",
				datatype: "timestamp with time zone",
				nullable: true,
			},
		},
	},
	{
		name: "block_parents",
		columns: []column{
			{
				name:     "block",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "parent",
				datatype: "text",
				nullable: false,
			},
		},
	},
	{
		name: "drand_entries",
		columns: []column{
			{
				name:     "round",
				datatype: "bigint",
				nullable: false,
			},
			{
				name:     "data",
				datatype: "bytea",
				nullable: false,
			},
		},
	},
	{
		name: "drand_block_entries",
		columns: []column{
			{
				name:     "round",
				datatype: "bigint",
				nullable: false,
			},
			{
				name:     "block",
				datatype: "text",
				nullable: false,
			},
		},
	},
	{
		name: "miner_powers",
		columns: []column{
			{
				name:     "miner_id",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "state_root",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "raw_byte_power",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "quality_adjusted_power",
				datatype: "text",
				nullable: false,
			},
		},
	},
	{
		name: "miner_states",
		columns: []column{
			{
				name:     "miner_id",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "owner_id",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "worker_id",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "peer_id",
				datatype: "bytea",
				nullable: true,
			},
			{
				name:     "sector_size",
				datatype: "text",
				nullable: false,
			},
		},
	},
	{
		name: "miner_sector_infos",
		columns: []column{
			{
				name:     "miner_id",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "sector_id",
				datatype: "bigint",
				nullable: false,
			},
			{
				name:     "state_root",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "sealed_cid",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "activation_epoch",
				datatype: "bigint",
				nullable: true,
			},
			{
				name:     "expiration_epoch",
				datatype: "bigint",
				nullable: true,
			},
			{
				name:     "deal_weight",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "verified_deal_weight",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "initial_pledge",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "expected_day_reward",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "expected_storage_pledge",
				datatype: "text",
				nullable: false,
			},
		},
	},

	{
		name: "miner_pre_commit_infos",
		columns: []column{
			{
				name:     "miner_id",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "sector_id",
				datatype: "bigint",
				nullable: false,
			},
			{
				name:     "state_root",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "sealed_cid",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "seal_rand_epoch",
				datatype: "bigint",
				nullable: true,
			},
			{
				name:     "expiration_epoch",
				datatype: "bigint",
				nullable: true,
			},
			{
				name:     "pre_commit_deposit",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "pre_commit_epoch",
				datatype: "bigint",
				nullable: true,
			},
			{
				name:     "deal_weight",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "verified_deal_weight",
				datatype: "text",
				nullable: false,
			},
			{
				name:     "is_replace_capacity",
				datatype: "boolean",
				nullable: true,
			},
			{
				name:     "replace_sector_deadline",
				datatype: "bigint",
				nullable: true,
			},
			{
				name:     "replace_sector_partition",
				datatype: "bigint",
				nullable: true,
			},
			{
				name:     "replace_sector_number",
				datatype: "bigint",
				nullable: true,
			},
		},
	},
}

func TestCreateSchema(t *testing.T) {
	if testing.Short() {
		t.Skip("short testing specified")
	}

	db, err := NewDatabase(context.Background(), "postgres://postgres@localhost:5432/postgres?sslmode=disable")
	if !assert.NoError(t, err, "connecting to database") {
		return
	}

	err = resetSchema(db)
	if !assert.NoError(t, err, "resetting schema") {
		return
	}

	err = db.CreateSchema()
	if !assert.NoError(t, err, "creating schema") {
		return
	}

	for _, table := range expectedTables {
		t.Run(table.name, func(t *testing.T) {
			var exists bool
			_, err := db.DB.QueryOne(pg.Scan(&exists), `SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema=? AND table_name=?)`, PGSchema, table.name)
			require.NoError(t, err, "table: %q", table.name)
			assert.True(t, exists, "table: %q", table.name)

			for _, col := range table.columns {
				var datatype string
				var nullable bool
				res, err := db.DB.QueryOne(pg.Scan(&datatype, &nullable), `SELECT data_type, is_nullable='YES' FROM information_schema.columns WHERE table_schema=? AND table_name=? AND column_name=?`, PGSchema, table.name, col.name)
				require.NoError(t, err, "querying column: %q", col.name)
				require.Equal(t, 1, res.RowsReturned(), "column %q not found", col.name)
				assert.Equal(t, col.datatype, datatype, "column %q datatype", col.name)
				assert.Equal(t, col.nullable, nullable, "column %q nullable", col.name)
			}
		})
	}
}

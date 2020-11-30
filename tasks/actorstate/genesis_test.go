package actorstate

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	apitest "github.com/filecoin-project/lotus/api/test"
	"github.com/filecoin-project/lotus/build"
	nodetest "github.com/filecoin-project/lotus/node/test"
	"github.com/filecoin-project/specs-actors/actors/builtin/miner"
	"github.com/filecoin-project/specs-actors/actors/builtin/power"
	"github.com/filecoin-project/specs-actors/actors/builtin/verifreg"
	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/sentinel-visor/storage"
	"github.com/filecoin-project/sentinel-visor/testutil"
)

func init() {
	build.InsecurePoStValidation = true
	err := os.Setenv("TRUST_PARAMS", "1")
	if err != nil {
		panic(err)
	}
	miner.SupportedProofTypes = map[abi.RegisteredSealProof]struct{}{
		abi.RegisteredSealProof_StackedDrg2KiBV1: {},
	}
	power.ConsensusMinerMinPower = big.NewInt(2048)
	verifreg.MinVerifiedDealSize = big.NewInt(256)
}

func TestGenesisProcessor(t *testing.T) {
	if testing.Short() {
		t.Skip("short testing requested")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	db, cleanup, err := testutil.WaitForExclusiveDatabase(ctx, t)
	require.NoError(t, err)
	defer func() { require.NoError(t, cleanup()) }()

	t.Logf("truncating database tables")
	err = truncateGenesisTables(t, db)
	require.NoError(t, err, "truncating tables")

	t.Logf("preparing chain")
	nodes, sn := nodetest.Builder(t, apitest.DefaultFullOpts(1), apitest.OneMiner)
	node := nodes[0]
	opener := testutil.NewAPIOpener(node)
	openedAPI, _, _ := opener.Open(ctx)

	apitest.MineUntilBlock(ctx, t, nodes[0], sn[0], nil)

	t.Logf("initializing genesis processor")
	d := &storage.Database{DB: db}
	p := NewGenesisProcessor(d, openedAPI)

	t.Logf("processing")
	gen, err := openedAPI.ChainGetGenesis(ctx)
	require.NoError(t, err, "chain genesis")
	err = p.ProcessGenesis(ctx, gen)
	require.NoError(t, err, "Run")

	// Not brilliantly useful tests, but they do test that data was read from the chain and written to the database

	t.Run("miner_deal_sectors", func(t *testing.T) {
		var count int
		_, err := db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM miner_sector_deals`)
		require.NoError(t, err)
		assert.NotEqual(t, 0, count)
	})

	t.Run("miner_sector_infos", func(t *testing.T) {
		var count int
		_, err := db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM miner_sector_infos`)
		require.NoError(t, err)
		assert.NotEqual(t, 0, count)
	})

	t.Run("power_actor_claims", func(t *testing.T) {
		var count int
		_, err := db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM power_actor_claims`)
		require.NoError(t, err)
		assert.NotEqual(t, 0, count)
	})

	t.Run("miner_infos", func(t *testing.T) {
		var count int
		_, err := db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM miner_infos`)
		require.NoError(t, err)
		assert.NotEqual(t, 0, count)
	})

	t.Run("market_deal_states", func(t *testing.T) {
		var count int
		_, err := db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM market_deal_states`)
		require.NoError(t, err)
		assert.NotEqual(t, 0, count)
	})

	t.Run("market_deal_proposals", func(t *testing.T) {
		var count int
		_, err := db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM market_deal_proposals`)
		require.NoError(t, err)
		assert.NotEqual(t, 0, count)
	})

	t.Run("id_addresses", func(t *testing.T) {
		var count int
		_, err := db.QueryOne(pg.Scan(&count), `SELECT COUNT(*) FROM id_addresses`)
		require.NoError(t, err)
		assert.NotEqual(t, 0, count)
	})
}

// truncateGenesisTables ensures the indexing tables are empty
func truncateGenesisTables(tb testing.TB, db *pg.DB) error {
	tables := []string{
		"miner_infos",
		"power_actor_claims",
		"miner_sector_infos",
		"miner_sector_deals",
		"market_deal_states",
		"market_deal_proposals",
		"id_addresses",
	}

	for _, tbl := range tables {
		_, err := db.Exec(fmt.Sprintf(`TRUNCATE TABLE %s`, tbl))
		require.NoError(tb, err, tbl)
	}

	return nil
}

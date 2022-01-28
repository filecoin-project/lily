package itests

import (
	"context"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lily/config"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/lens/lily"
	lutil "github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/storage"
	api2 "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/node"
	"github.com/go-pg/pg/v10"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
	"golang.org/x/xerrors"
	"testing"
	"time"
)

type VectorWalkValidatorBuilder struct {
	options []func(tv *VectorWalkValidator)
	vector  *TestVector
}

func NewVectorWalkValidatorBuilder(tv *TestVector) VectorWalkValidatorBuilder {
	var b VectorWalkValidatorBuilder
	b.vector = tv
	return b
}

func (b *VectorWalkValidatorBuilder) add(cb func(vector *VectorWalkValidator)) {
	b.options = append(b.options, cb)
}

func (b VectorWalkValidatorBuilder) WithDatabase(strg *storage.Database) VectorWalkValidatorBuilder {
	b.add(func(tv *VectorWalkValidator) {
		tv.strg = strg
	})
	return b
}

func (b VectorWalkValidatorBuilder) WithRange(from, to int64) VectorWalkValidatorBuilder {
	b.add(func(vw *VectorWalkValidator) {
		vw.from = from
		vw.to = to
	})
	return b
}

func (b VectorWalkValidatorBuilder) WithTasks(tasks ...string) VectorWalkValidatorBuilder {
	b.add(func(vw *VectorWalkValidator) {
		vw.tasks = tasks
	})
	return b
}

func (b VectorWalkValidatorBuilder) WithNodeCfg(cfg *TestNodeConfig) VectorWalkValidatorBuilder {
	b.add(func(vw *VectorWalkValidator) {
		vw.lilyCfg = cfg
	})
	return b
}

func (b VectorWalkValidatorBuilder) Build(ctx context.Context, t testing.TB) *VectorWalkValidator {
	def := config.DefaultConf()
	ncfg := *def
	storageName := "TestDatabase1"
	ncfg.Storage = config.StorageConf{
		Postgresql: map[string]config.PgStorageConf{
			storageName: {
				URLEnv:          "LILY_TEST_DB",
				PoolSize:        20,
				ApplicationName: "lily-itests",
				AllowUpsert:     false,
				SchemaName:      "public",
			},
		},
	}

	lilyCfg := &TestNodeConfig{
		LilyConfig: &ncfg,
		CacheConfig: &lutil.CacheConfig{
			BlockstoreCacheSize: 0,
			StatestoreCacheSize: 0,
		},
		RepoPath:    t.TempDir(),
		Snapshot:    b.vector.Snapshot,
		Genesis:     b.vector.Genesis,
		ApiEndpoint: "/ip4/127.0.0.1/tcp/4321",
	}

	vw := &VectorWalkValidator{
		ctx:     ctx,
		t:       t,
		strg:    nil,
		from:    0,
		to:      0,
		tasks:   nil,
		lilyCfg: lilyCfg,
	}

	for _, opt := range b.options {
		opt(vw)
	}

	if vw.strg == nil {
		t.Fatal("storage required")
	}
	if vw.tasks == nil {
		t.Fatal("tasks required")
	}
	if vw.lilyCfg == nil {
		t.Fatal("lily config required")
	}
	for _, task := range vw.tasks {
		models, ok := TaskModels[task]
		if !ok {
			t.Fatalf("no models for task: %s", task)
		}
		truncateTable(t, vw.strg.AsORM(), models...)
	}
	// always truncate this table
	truncateTable(t, vw.strg.AsORM(), "visor_processing_reports")

	return vw
}

type VectorWalkValidator struct {
	ctx      context.Context
	t        testing.TB
	strg     *storage.Database
	from, to int64
	tasks    []string
	lilyCfg  *TestNodeConfig
	api      *lily.LilyNodeAPI
}

func (vw *VectorWalkValidator) Run(ctx context.Context) node.StopFunc {
	// start the lily node
	lilyAPI, apiCleanup := NewTestNode(vw.t, ctx, vw.lilyCfg)
	api := lilyAPI.(*lily.LilyNodeAPI)
	vw.api = api

	// create a walk config from the builder values
	walkCfg := &lily.LilyWalkConfig{
		From:                vw.from,
		To:                  vw.to,
		Name:                vw.t.Name(),
		Tasks:               vw.tasks,
		Window:              0,
		RestartOnFailure:    false,
		RestartOnCompletion: false,
		RestartDelay:        0,
		Storage:             "TestDatabase1",
	}

	walkStart := time.Now()
	// walk that walk
	vw.t.Logf("starting walk from %d to %d with tasks %s", walkCfg.From, walkCfg.To, walkCfg.Tasks)
	res, err := vw.api.LilyWalk(ctx, walkCfg)
	require.NoError(vw.t, err)
	require.NotEmpty(vw.t, res)

	vw.t.Log("waiting for walk to complete")
	// wait for the job to get to the scheduler else the job ID isn't found
	time.Sleep(3 * time.Second)
	// wait for the walk to complete
	ress, err := vw.api.LilyJobWait(ctx, res.ID)
	require.NoError(vw.t, err)
	require.NotEmpty(vw.t, ress)

	vw.t.Logf("walk from %d to %d took %s", vw.from, vw.to, time.Since(walkStart))

	vw.t.Log("waiting for persistence to complete")
	// wait for persistence to complete
	time.Sleep(3 * time.Second)
	return apiCleanup
}

func (vw *VectorWalkValidator) Validate(t *testing.T) {
	var tsv []TipSetStateValidator
	var epv []EpochValidator
	for _, task := range vw.tasks {
		taskValidators, ok := TaskValidators[task]
		if !ok {
			t.Fatal("no validators for task", task)
		}
		for _, validator := range taskValidators {
			switch v := validator.(type) {
			case TipSetStateValidator:
				tsv = append(tsv, v)
			case EpochValidator:
				epv = append(epv, v)
			default:
				t.Fatalf("unknown validator type: %T", v)
			}
		}
	}
	if len(tsv) > 0 {
		vw.ValidateTipSetStates(t, tsv...)
	}
	if len(epv) > 0 {
		vw.ValidateEpochState(t, epv...)
	}
}

func (vw *VectorWalkValidator) ValidateEpochState(t *testing.T, mv ...EpochValidator) {
	t.Run("validate epoch states", func(t *testing.T) {

		walkHead, err := vw.api.ChainGetTipSetByHeight(vw.ctx, abi.ChainEpoch(vw.to), types.EmptyTSK)
		require.NoError(vw.t, err)

		if int64(walkHead.Height()) != vw.to {
			// TODO
			t.Fatal("TODO to was a null round, not handled yet")
		}

		tss, err := collectEpochsWithNullRoundsRange(vw.ctx, vw.api, vw.from, vw.to)
		require.NoError(vw.t, err)

		for epoch := vw.from; epoch <= vw.to; epoch++ {
			for _, m := range mv {
				m.Validate(t, epoch, tss[epoch], vw.strg, vw.api)
			}
		}
	})
}

// ValidateModels validates that the data in the database for the passed models matches the results returned by the lotus api.
func (vw *VectorWalkValidator) ValidateTipSetStates(t *testing.T, mv ...TipSetStateValidator) {
	t.Run("validate tipset states", func(t *testing.T) {
		walkHead, err := vw.api.ChainGetTipSetByHeight(vw.ctx, abi.ChainEpoch(vw.to), types.EmptyTSK)
		require.NoError(vw.t, err)

		if int64(walkHead.Height()) != vw.to {
			// TODO
			t.Fatal("TODO to was a null round, not handled yet")
		}

		tss, err := collectTipSetRange(vw.ctx, vw.api, vw.from, vw.to)
		require.NoError(vw.t, err)

		for _, ts := range tss {
			if ts.Height() < abi.ChainEpoch(vw.from) {
				break
			}
			tsState, err := StateForTipSet(vw.ctx, vw.api, ts)
			require.NoError(vw.t, err)

			for _, m := range mv {
				m.Validate(t, tsState, vw.strg)
			}
		}
	})
}

// TipSetState contains the state of actors, blocks, messages, and receipts for the tipset it was derived from.
type TipSetState struct {
	ts *types.TipSet
	// actors changed whiloe producing this tipset
	actorsChanges map[address.Address]types.Actor
	// blocks in this tipset
	blocks []*types.BlockHeader
	// messages in the blocks of this TipSet (will contain duplicate messages)
	blockMsgs map[*types.BlockHeader]*api2.BlockMessages
	// messages and their receipts
	msgRects map[cid.Cid]*types.MessageReceipt
}

// actorsFromGenesisBlock returns the set of actors found in the genesis block.
func actorsFromGenesisBlock(ctx context.Context, n *lily.LilyNodeAPI, ts *types.TipSet) (map[address.Address]types.Actor, error) {
	actors, err := n.StateListActors(ctx, ts.Key())
	if err != nil {
		return nil, err
	}

	actorsChanged := make(map[address.Address]types.Actor)
	for _, addr := range actors {
		act, err := n.StateGetActor(ctx, addr, ts.Key())
		if err != nil {
			return nil, err
		}
		actorsChanged[addr] = *act
	}
	return actorsChanged, nil
}

// StateForTipSet returns a TipSetState for TipSet `ts`. All state is derived from Lotus API calls.
func StateForTipSet(ctx context.Context, n *lily.LilyNodeAPI, ts *types.TipSet) (*TipSetState, error) {
	pts, err := n.ChainGetTipSet(ctx, ts.Parents())
	if err != nil {
		return nil, err
	}

	actorsChanged := make(map[address.Address]types.Actor)
	if pts.Height() == 0 {
		actorsChanged, err = actorsFromGenesisBlock(ctx, n, ts)
		if err != nil {
			return nil, err
		}
	} else {
		// the actors who changed while producing this tipset
		tsActorChanges, err := n.StateChangedActors(ctx, pts.ParentState(), ts.ParentState())
		if err != nil {
			return nil, err
		}

		for addrStr, act := range tsActorChanges {
			addr, err := address.NewFromString(addrStr)
			if err != nil {
				return nil, err
			}
			actorsChanged[addr] = act
		}
	}

	// messages from the parent tipset, their receipts will be in (child) ts
	parentMessages, err := n.ChainAPI.Chain.MessagesForTipset(pts)
	if err != nil {
		return nil, err
	}

	blkMsgs := make(map[*types.BlockHeader]*api2.BlockMessages)
	msgRects := make(map[cid.Cid]*types.MessageReceipt)
	for _, blk := range ts.Blocks() {
		// map of blocks to their messages
		msgs, err := n.ChainGetBlockMessages(ctx, blk.Cid())
		if err != nil {
			return nil, err
		}
		blkMsgs[blk] = msgs

		// map of parent messages to their receipts
		for i := 0; i < len(parentMessages); i++ {
			r, err := n.ChainAPI.Chain.GetParentReceipt(blk, i)
			if err != nil {
				return nil, err
			}
			msgRects[parentMessages[i].Cid()] = r
		}

	}

	return &TipSetState{
		ts:            ts,
		actorsChanges: actorsChanged,
		blocks:        ts.Blocks(),
		blockMsgs:     blkMsgs,
		msgRects:      msgRects,
	}, nil
}

func collectEpochsWithNullRoundsRange(ctx context.Context, api lens.API, from, to int64) (map[int64]*types.TipSet, error) {
	head, err := api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(to), types.EmptyTSK)
	if err != nil {
		return nil, err
	}
	if int64(head.Height()) != to {
		// TODO
		return nil, xerrors.Errorf("TODO to (%d) was a null round, not handled yet", to)
	}

	tail, err := api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(from), types.EmptyTSK)
	if err != nil {
		return nil, err
	}
	if int64(tail.Height()) != from {
		// TODO
		return nil, xerrors.Errorf("TODO from (%d) was a null round, not handled yet", from)
	}

	out := make(map[int64]*types.TipSet)

	current := head
	for {
		out[int64(current.Height())] = current
		parent, err := api.ChainGetTipSet(ctx, current.Parents())
		if err != nil {
			return nil, err
		}
		current = parent
		if current.Height() == tail.Height() {
			out[int64(current.Height())] = current
			break
		}
	}
	return out, nil

}

func collectTipSetRange(ctx context.Context, api lens.API, from, to int64) ([]*types.TipSet, error) {
	head, err := api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(to), types.EmptyTSK)
	if err != nil {
		return nil, err
	}
	if int64(head.Height()) != to {
		// TODO
		return nil, xerrors.Errorf("TODO to (%d) was a null round, not handled yet", to)
	}

	tail, err := api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(from), types.EmptyTSK)
	if err != nil {
		return nil, err
	}
	if int64(tail.Height()) != from {
		// TODO
		return nil, xerrors.Errorf("TODO from (%d) was a null round, not handled yet", from)
	}

	out := make([]*types.TipSet, 0, head.Height()-tail.Height())

	current := head
	for {
		parent, err := api.ChainGetTipSet(ctx, current.Parents())
		if err != nil {
			return nil, err
		}
		out = append(out, current)
		current = parent
		if current.Height() == tail.Height() {
			break
		}
	}
	return out, nil

}

// truncateTables ensures the tables are truncated
func truncateTable(tb testing.TB, db *pg.DB, tableNames ...string) {
	for _, table := range tableNames {
		_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s", table))
		require.NoError(tb, err, table)
	}
}

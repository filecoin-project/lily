package commands

import (
	"context"
	"math"
	"math/big"
	"os"
	"os/signal"
	"syscall"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/go-pg/pg/v10"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/lens"
	messagemodel "github.com/filecoin-project/sentinel-visor/model/messages"
	"github.com/filecoin-project/sentinel-visor/storage"
)

var FixGas = &cli.Command{
	Name: "fixgas",
	Action: func(cctx *cli.Context) error {
		ctx, rctx, err := setupStorageAndAPI(cctx)
		if err != nil {
			return xerrors.Errorf("setup storage and api: %w", err)
		}
		defer func() {
			rctx.closer()
			if err := rctx.db.Close(ctx); err != nil {
				log.Errorw("close database", "error", err)
			}
		}()

		// Set up a context that is canceled when the command is interrupted
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Set up a signal handler to cancel the context
		go func() {
			interrupt := make(chan os.Signal, 1)
			signal.Notify(interrupt, syscall.SIGTERM, syscall.SIGINT)
			select {
			case <-interrupt:
				cancel()
			case <-ctx.Done():
			}
		}()
		return FixMessageGasEconomy(ctx, rctx.db, rctx.opener)
	},
}

func FixMessageGasEconomy(ctx context.Context, db *storage.Database, opener lens.APIOpener) error {
	api, closer, err := opener.Open(ctx)
	if err != nil {
		return err
	}
	defer closer()

	var msgs []*messagemodel.MessageGasEconomy
	if err := db.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
		err := tx.ModelContext(ctx, &msgs).Order("height DESC").Select()
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	for _, msg := range msgs {
		// get the tipset to recompute the base fee.
		ts, err := api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(msg.Height), types.EmptyTSK)
		if err != nil {
			return err
		}

		if ts.ParentState().String() != msg.StateRoot {
			log.Errorw("Invalid stateroot", "expected", msg.StateRoot, "actual", ts.ParentState().String())
			return xerrors.New("Cannot update message table, stateroot mismatch")
		}

		// bug fix: these value were swapped.
		unique := msg.GasLimitTotal
		total := msg.GasLimitUniqueTotal

		newBaseFee := store.ComputeNextBaseFee(ts.Blocks()[0].ParentBaseFee, unique, len(ts.Blocks()), ts.Height())
		baseFeeRat := new(big.Rat).SetFrac(newBaseFee.Int, new(big.Int).SetUint64(build.FilecoinPrecision))
		baseFee, _ := baseFeeRat.Float64()
		baseFeeChange := new(big.Rat).SetFrac(newBaseFee.Int, ts.Blocks()[0].ParentBaseFee.Int)
		baseFeeChangeF, _ := baseFeeChange.Float64()

		msg.GasLimitTotal = total
		msg.GasLimitUniqueTotal = unique
		msg.BaseFee = baseFee
		msg.BaseFeeChangeLog = math.Log(baseFeeChangeF) / math.Log(1.125)
		msg.GasFillRatio = float64(total) / float64(len(ts.Blocks())*build.BlockGasTarget)
		msg.GasCapacityRatio = float64(unique) / float64(len(ts.Blocks())*build.BlockGasTarget)
		msg.GasWasteRatio = float64(total-unique) / float64(len(ts.Blocks())*build.BlockGasTarget)
	}
	batchSize := 1000
	batch := make([]*messagemodel.MessageGasEconomy, 0, batchSize)
	for idx, msg := range msgs {
		batch = append(batch, msg)
		if idx != 0 && idx%batchSize == 0 {
			log.Infow("Updating", "index", idx, "remaining", len(msgs)-idx, "start", batch[0].Height, "end", batch[len(batch)-1].Height)
			if err := db.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
				_, err := tx.ModelContext(ctx, &batch).Update()
				return err
			}); err != nil {
				return err
			}
			// reset the batch
			batch = batch[:0]
		}
	}
	// update anything remaining for case when len(msgs)/batchSize != 0
	if len(batch) > 0 {
		log.Infow("Final Update", "start", batch[0].Height, "end", batch[len(batch)-1].Height)
		if err := db.DB.RunInTransaction(ctx, func(tx *pg.Tx) error {
			_, err := tx.ModelContext(ctx, &batch).Update()
			return err
		}); err != nil {
			return err
		}
	}
	return nil
}

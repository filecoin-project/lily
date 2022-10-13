package cborable

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"time"

	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/v8/actors/util/adt"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	v1car "github.com/ipld/go-car"
	"github.com/ipld/go-car/util"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/cborable"
)

var log = logging.Logger("load/car")

const BitWidth = 8

func NewCarResultConsumer(ts, pts *types.TipSet) *CarResultConsumer {
	return &CarResultConsumer{
		Current:  ts,
		Executed: pts,
	}
}

type CarResultConsumer struct {
	Current  *types.TipSet
	Executed *types.TipSet
}

func (c *CarResultConsumer) Name() string {
	return reflect.TypeOf(CarResultConsumer{}).Name()
}

func (c *CarResultConsumer) Type() transform.Kind {
	return "cborable"
}

func (c *CarResultConsumer) Consume(ctx context.Context, in chan transform.Result) error {
	start := time.Now()
	defer func() {
		log.Infow("CarResult Consume complete", "duration", time.Since(start))
	}()
	bs := blockstore.NewMemorySync()
	store := adt.WrapBlockStore(ctx, bs)
	mw, err := NewModelWriter(store, BitWidth)
	if err != nil {
		return err
	}

	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if res.Data() == nil {
				continue
			}
			models := res.Data().(cborable.CborablResult)
			for _, m := range models.Model {
				if err := mw.StageModel(ctx, m); err != nil {
					return err
				}
			}
		}
	}
	log.Infow("staged models", "duration", time.Since(start), "size", len(mw.cache))

	metaModelMapRoot, err := mw.Finalize(ctx)
	if err != nil {
		return err
	}

	stateContainer := &ModelStateContainer{
		Current:  c.Current,
		Executed: c.Executed,
		Models:   metaModelMapRoot,
	}

	stateMap, err := adt.MakeEmptyMap(store, BitWidth)
	if err != nil {
		return err
	}
	if err = stateMap.Put(TipsetKeyer{c.Current.Key()}, stateContainer); err != nil {
		return err
	}
	stateRoot, err := stateMap.Root()
	if err != nil {
		return err
	}
	log.Infow("model state root", "root", stateRoot.String())
	f, err := os.Create(fmt.Sprintf("./%d_%s.car", c.Current.Height(), c.Current.ParentState()))
	if err != nil {
		return err
	}
	defer f.Close()
	if err := WriteCAR(ctx, stateRoot, bs, f); err != nil {
		return err
	}
	return nil
}

type ModelStateContainer struct {
	Current  *types.TipSet
	Executed *types.TipSet
	Models   cid.Cid
}

func WriteCAR(ctx context.Context, root cid.Cid, bs blockstore.Blockstore, w io.Writer) error {
	if err := v1car.WriteHeader(&v1car.CarHeader{
		Roots:   []cid.Cid{root},
		Version: 1,
	}, w); err != nil {
		return err
	}
	keys, err := bs.AllKeysChan(ctx)
	if err != nil {
		return err
	}
	count := 0
	for key := range keys {
		count++
		blk, err := bs.Get(ctx, key)
		if err != nil {
			log.Errorw("getting block", "error", err)
			return err
		}
		if err := util.LdWrite(w, blk.Cid().Bytes(), blk.RawData()); err != nil {
			return err
		}
	}
	log.Infow("wrote keys", "count", count)
	return nil
}

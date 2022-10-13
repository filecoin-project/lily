package miner

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/miner"
)

type PostSectorMessageTransform struct {
	meta v2.ModelMeta
}

func NewPostSectorMessageTransform() *PostSectorMessageTransform {
	info := miner.PostSectorMessage{}
	return &PostSectorMessageTransform{meta: info.Meta()}
}

func (s *PostSectorMessageTransform) Run(ctx context.Context, in chan transform.IndexState, out chan transform.Result) error {
	log.Debugf("run %s", s.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sqlModels := make(minermodel.MinerSectorPostList, 0, len(res.State().Data))
			for _, modeldata := range res.State().Data {
				sm := modeldata.(*miner.PostSectorMessage)
				sqlModels = append(sqlModels, &minermodel.MinerSectorPost{
					Height:         int64(sm.Height),
					MinerID:        sm.Miner.String(),
					SectorID:       uint64(sm.SectorNumber),
					PostMessageCID: sm.PostMessageCID.String(),
				})
			}
			if len(sqlModels) > 0 {
				out <- &persistable.Result{Model: sqlModels}
			}
		}
	}
	return nil
}

func (s *PostSectorMessageTransform) ModelType() v2.ModelMeta {
	return s.meta
}

func (s *PostSectorMessageTransform) Name() string {
	info := PostSectorMessageTransform{}
	return reflect.TypeOf(info).Name()
}

func (s *PostSectorMessageTransform) Matcher() string {
	return fmt.Sprintf("^%s$", s.meta.String())
}

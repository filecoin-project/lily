package miner

import (
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/lily/chain/indexer/v2/extract"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable"
	"github.com/filecoin-project/lily/chain/indexer/v2/transform/persistable/actor"
	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/model/v2/actors/miner"
)

type PostSectorMessageTransform struct {
	meta     v2.ModelMeta
	taskName string
}

func NewPostSectorMessageTransform(taskName string) *PostSectorMessageTransform {
	info := miner.PostSectorMessage{}
	return &PostSectorMessageTransform{meta: info.Meta(), taskName: taskName}
}

func (s *PostSectorMessageTransform) Run(ctx context.Context, reporter string, in chan *extract.ActorStateResult, out chan transform.Result) error {
	log.Debugf("run %s", s.Name())
	for res := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			report := actor.ToProcessingReport(s.taskName, reporter, res)
			data := model.PersistableList{report}
			log.Debugw("received data", "count", len(res.Results.Models()))
			sqlModels := make(minermodel.MinerSectorPostList, 0, len(res.Results.Models()))
			for _, modeldata := range res.Results.Models() {
				sm := modeldata.(*miner.PostSectorMessage)
				sqlModels = append(sqlModels, &minermodel.MinerSectorPost{
					Height:         int64(sm.Height),
					MinerID:        sm.Miner.String(),
					SectorID:       uint64(sm.SectorNumber),
					PostMessageCID: sm.PostMessageCID.String(),
				})
			}
			if len(sqlModels) > 0 {
				data = append(data, sqlModels)
			}
			out <- &persistable.Result{Model: data}
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

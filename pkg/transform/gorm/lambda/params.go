package lambda

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"gorm.io/gorm"

	"github.com/filecoin-project/lily/lens/util"
	"github.com/filecoin-project/lily/pkg/transform/gorm/models"
	"github.com/filecoin-project/lily/tasks"
)

func ParseParams(ctx context.Context, api tasks.DataSource, db *gorm.DB) error {
	var messages []models.Message
	res := db.Find(&messages)
	if res.Error != nil {
		return res.Error
	}
	out := make([]models.ParsedMessageParams, 0, len(messages))
	for _, msg := range messages {
		act, err := api.Actor(ctx, msg.To.Addr, types.EmptyTSK)
		if err != nil {
			return err
		}
		params, method, err := util.ParseParams(msg.Params, abi.MethodNum(msg.Method), act.Code)
		if err != nil {
			// TODO could continue
			return err
		}
		out = append(out, models.ParsedMessageParams{
			Cid:    msg.Cid,
			Params: params,
			Method: method,
		})
	}
	if err := db.Create(out).Error; err != nil {
		return err
	}
	return nil
}

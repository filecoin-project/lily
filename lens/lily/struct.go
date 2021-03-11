package lily

import (
	"context"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/node/impl/common"
	"github.com/filecoin-project/lotus/node/impl/full"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/sentinel-visor/lens"
)

var log = logging.Logger("lily-api")

type LilyAPIStruct struct {
	// authentication
	common.CommonAPI
	// chain notifications and inspection
	full.ChainAPI
	// actor state extraction
	full.StateAPI

	Internal struct {
		Store                        func() adt.Store                                                                     `perm:"admin"`
		LilyWatchStart               func(context.Context, *LilyWatchConfig) error                                        `perm:"read"`
		GetExecutedMessagesForTipset func(context.Context, *types.TipSet, *types.TipSet) ([]*lens.ExecutedMessage, error) `perm:"read"`
	}
}

func (s *LilyAPIStruct) Store() adt.Store {
	return s.Internal.Store()
}

func (s *LilyAPIStruct) LilyWatchStart(ctx context.Context, cfg *LilyWatchConfig) error {
	return s.Internal.LilyWatchStart(ctx, cfg)
}

func (s *LilyAPIStruct) GetExecutedMessagesForTipset(ctx context.Context, ts, pts *types.TipSet) ([]*lens.ExecutedMessage, error) {
	return s.Internal.GetExecutedMessagesForTipset(ctx, ts, pts)
}

var _ LilyAPI = &LilyAPIStruct{}

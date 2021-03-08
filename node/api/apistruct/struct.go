package apistruct

import (
	"context"

	lotusstruct "github.com/filecoin-project/lotus/api/apistruct"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/sentinel-visor/node/api"
)

var log = logging.Logger("lily-api")

type LilyNodeStruct struct {
	lotusstruct.FullNodeStruct

	Internal struct {
		LilyWatchStart func(context.Context, *api.LilyWatchConfig) error `perm:"read"`
	}
}

func (s *LilyNodeStruct) LilyWatchStart(ctx context.Context, cfg *api.LilyWatchConfig) error {
	return s.Internal.LilyWatchStart(ctx, cfg)
}

var _ api.LilyNode = &LilyNodeStruct{}

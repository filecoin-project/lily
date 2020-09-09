package tasks

import (
	"github.com/filecoin-project/lotus/chain/types"
	"time"
)

type BlockProcessTask struct {
	Cid    string `pg:",pk,notnull"`
	Height int64  `pg:",use_zero"`

	CreatedAt   time.Time `pg:",notnull"`
	AttemptedAt time.Time
	CompletedAt time.Time
}

func NewBlockProcessTask(bh *types.BlockHeader, createdAt time.Time) *BlockProcessTask {
	return &BlockProcessTask{
		Cid:       bh.Cid().String(),
		Height:    int64(bh.Height),
		CreatedAt: createdAt,
	}
}

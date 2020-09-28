package visor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	"github.com/go-pg/pg/v10"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
)

func NewProcessingStateChange(ts *types.TipSet) *ProcessingStateChange {
	return &ProcessingStateChange{
		TipSet:  ts.Key().String(),
		Height:  int64(ts.Height()),
		AddedAt: time.Now(),
	}
}

type ProcessingStateChange struct {
	tableName struct{} `pg:"visor_processing_statechanges"`

	TipSet string `pg:",pk,notnull"`

	Height int64 `pg:",use_zero"`

	// AddedAt is the time the block was discovered and written to the table
	AddedAt time.Time `pg:",notnull"`

	// ClaimedUntil marks the block as claimed for processing until the set time
	ClaimedUntil time.Time

	// CompletedAt is the time the block was read from the chain and analysed for actor state changes
	CompletedAt time.Time

	// ErrorsDetected contains any error encountered when analysing the block for actor state changes
	ErrorsDetected string
}

func (p *ProcessingStateChange) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, p).
		OnConflict("do nothing").
		Insert(); err != nil {
		return fmt.Errorf("persisting processing block: %w", err)
	}
	return nil
}

func (p *ProcessingStateChange) TipSetKey() (types.TipSetKey, error) {
	return TipSetKeyFromString(p.TipSet)
}

type ProcessingStateChangeList []*ProcessingStateChange

func (pl ProcessingStateChangeList) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "ProcessingBlockList.PersistWithTx", trace.WithAttributes(label.Int("count", len(pl))))
	defer span.End()
	for _, p := range pl {
		if err := p.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}

type ProcessingActor struct {
	tableName struct{} `pg:"visor_processing_actors"`

	Head            string `pg:",pk,notnull"`
	Code            string `pg:",pk,notnull"`
	Nonce           string
	Balance         string
	Address         string
	ParentStateRoot string // cid
	TipSet          string
	ParentTipSet    string

	Height int64 `pg:",use_zero"`

	// AddedAt is the time the actor was discovered and written to the table
	AddedAt time.Time `pg:",notnull"`

	// ClaimedUntil marks the actor as claimed for processing until the set time
	ClaimedUntil time.Time

	// CompletedAt is the time the actor was read from the chain and its state read
	CompletedAt time.Time

	// ErrorsDetected contains any error encountered when reading the actor's state
	ErrorsDetected string
}

func (p *ProcessingActor) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, p).
		OnConflict("do nothing").
		Insert(); err != nil {
		return fmt.Errorf("persisting processing actor: %w", err)
	}
	return nil
}

func (p *ProcessingActor) TipSetKey() (types.TipSetKey, error) {
	return TipSetKeyFromString(p.TipSet)
}

func (p *ProcessingActor) ParentTipSetKey() (types.TipSetKey, error) {
	return TipSetKeyFromString(p.ParentTipSet)
}

type ProcessingActorList []*ProcessingActor

func (pl ProcessingActorList) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "ProcessingActorList.PersistWithTx", trace.WithAttributes(label.Int("count", len(pl))))
	defer span.End()
	for _, p := range pl {
		if err := p.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}

func NewProcessingMessage(ts *types.TipSet) *ProcessingMessage {
	return &ProcessingMessage{
		TipSet:  ts.Key().String(),
		Height:  int64(ts.Height()),
		AddedAt: time.Now(),
	}
}

type ProcessingMessage struct {
	tableName struct{} `pg:"visor_processing_messages"`

	TipSet string `pg:",pk,notnull"`
	Height int64  `pg:",use_zero"`

	// AddedAt is the time the block was discovered and written to the table
	AddedAt time.Time `pg:",notnull"`

	// ClaimedUntil marks the block as claimed for message processing until the set time
	ClaimedUntil time.Time

	// CompletedAt is the time the block was read from the chain and its messages read
	CompletedAt time.Time

	// ErrorsDetected contains any error encountered when reading the block's messages
	ErrorsDetected string
}

func (p *ProcessingMessage) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	if _, err := tx.ModelContext(ctx, p).
		OnConflict("do nothing").
		Insert(); err != nil {
		return fmt.Errorf("persisting processing actor: %w", err)
	}
	return nil
}

func (p *ProcessingMessage) TipSetKey() (types.TipSetKey, error) {
	return TipSetKeyFromString(p.TipSet)
}

type ProcessingMessageList []*ProcessingMessage

func (pl ProcessingMessageList) PersistWithTx(ctx context.Context, tx *pg.Tx) error {
	ctx, span := global.Tracer("").Start(ctx, "ProcessingMessageList.PersistWithTx", trace.WithAttributes(label.Int("count", len(pl))))
	defer span.End()
	for _, p := range pl {
		if err := p.PersistWithTx(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}

func TipSetKeyFromString(s string) (types.TipSetKey, error) {
	if len(s) < 2 {
		return types.EmptyTSK, xerrors.Errorf("invalid tipset")
	}

	s = s[1 : len(s)-1]

	cids := []cid.Cid{}
	cidStrs := strings.Split(s, ",")
	for _, cidStr := range cidStrs {
		c, err := cid.Decode(cidStr)
		if err != nil {
			return types.EmptyTSK, xerrors.Errorf("invalid cid: %w", err)
		}
		cids = append(cids, c)
	}

	return types.NewTipSetKey(cids...), nil
}

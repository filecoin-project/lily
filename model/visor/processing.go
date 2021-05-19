package visor

import (
	"context"
	"strings"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
)

func NewProcessingTipSet(ts *types.TipSet) *ProcessingTipSet {
	return &ProcessingTipSet{
		TipSet:  ts.Key().String(),
		Height:  int64(ts.Height()),
		AddedAt: time.Now(),
	}
}

type ProcessingTipSet struct {
	tableName struct{} `pg:"visor_processing_tipsets"` // nolint: structcheck,unused

	TipSet string `pg:",pk,notnull"`

	Height int64 `pg:",use_zero"`

	// AddedAt is the time the tipset was discovered and written to the table
	AddedAt time.Time `pg:",notnull"`

	// State change processing

	// StatechangeClaimedUntil marks the tipset as claimed for actor state change processing until the set time
	StatechangeClaimedUntil time.Time

	// StatechangeCompletedAt is the time the tipset was read from the chain and analysed for actor state changes
	StatechangeCompletedAt time.Time

	// StatechangeErrorsDetected contains any error encountered when analysing the tipset for actor state changes
	StatechangeErrorsDetected string

	// Message reading

	// MessageClaimedUntil marks the tipset as claimed for message processing until the set time
	MessageClaimedUntil time.Time

	// MessageCompletedAt is the time the tipset was read from the chain and its messages read
	MessageCompletedAt time.Time

	// MessageErrorsDetected contains any error encountered when reading the tipset's messages
	MessageErrorsDetected string

	// Chain economics processing

	// EconomicsClaimedUntil marks the tipset as claimed for chain economics processing until the set time
	EconomicsClaimedUntil time.Time

	// EconomicsCompletedAt is the time the tipset was read from the chain and its chain economics read
	EconomicsCompletedAt time.Time

	// EconomicsErrorsDetected contains any error encountered when reading the tipset's chain economics
	EconomicsErrorsDetected string
}

func (p *ProcessingTipSet) Persist(ctx context.Context, s model.StorageBatch, version int) error {
	return s.PersistModel(ctx, p)
}

func (p *ProcessingTipSet) TipSetKey() (types.TipSetKey, error) {
	return TipSetKeyFromString(p.TipSet)
}

type ProcessingTipSetList []*ProcessingTipSet

func (pl ProcessingTipSetList) Persist(ctx context.Context, s model.StorageBatch, version int) error {
	if len(pl) == 0 {
		return nil
	}
	ctx, span := global.Tracer("").Start(ctx, "ProcessingTipSetList.Persist", trace.WithAttributes(label.Int("count", len(pl))))
	defer span.End()

	return s.PersistModel(ctx, pl)
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

type ProcessingActor struct {
	tableName struct{} `pg:"visor_processing_actors"` // nolint: structcheck,unused

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

func (p *ProcessingActor) Persist(ctx context.Context, s model.StorageBatch, version int) error {
	return s.PersistModel(ctx, p)
}

func (p *ProcessingActor) TipSetKey() (types.TipSetKey, error) {
	return TipSetKeyFromString(p.TipSet)
}

func (p *ProcessingActor) ParentTipSetKey() (types.TipSetKey, error) {
	return TipSetKeyFromString(p.ParentTipSet)
}

type ProcessingActorList []*ProcessingActor

func (pl ProcessingActorList) Persist(ctx context.Context, s model.StorageBatch, version int) error {
	if len(pl) == 0 {
		return nil
	}
	ctx, span := global.Tracer("").Start(ctx, "ProcessingActorList.Persist", trace.WithAttributes(label.Int("count", len(pl))))
	defer span.End()

	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, pl)
}

func NewProcessingMessage(m *types.Message, height int64) *ProcessingMessage {
	return &ProcessingMessage{
		Cid:     m.Cid().String(),
		Height:  height,
		AddedAt: time.Now(),
	}
}

type ProcessingMessage struct {
	tableName struct{} `pg:"visor_processing_messages"` // nolint: structcheck,unused

	Cid    string `pg:",pk,notnull"`
	Height int64  `pg:",use_zero"`

	// AddedAt is the time the message was discovered and written to the table
	AddedAt time.Time `pg:",notnull"`

	// GasOutputsClaimedUntil marks the message as claimed for gas output processing until the set time
	GasOutputsClaimedUntil time.Time

	// GasOutputsCompletedAt is the time when processing gas output completed
	GasOutputsCompletedAt time.Time

	// GasOutputsErrorsDetected contains any error encountered when processing gas output
	GasOutputsErrorsDetected string
}

func (p *ProcessingMessage) Persist(ctx context.Context, s model.StorageBatch, version int) error {
	return s.PersistModel(ctx, p)
}

type ProcessingMessageList []*ProcessingMessage

func (pl ProcessingMessageList) Persist(ctx context.Context, s model.StorageBatch, version int) error {
	if len(pl) == 0 {
		return nil
	}
	ctx, span := global.Tracer("").Start(ctx, "ProcessingMessageList.Persist", trace.WithAttributes(label.Int("count", len(pl))))
	defer span.End()
	return s.PersistModel(ctx, pl)
}

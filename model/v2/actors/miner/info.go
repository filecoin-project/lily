package miner

import (
	"bytes"
	"context"
	"fmt"
	"reflect"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	v2 "github.com/filecoin-project/lily/model/v2"
	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
	miner2 "github.com/filecoin-project/lily/tasks/actorstate/miner"
)

var log = logging.Logger("miner")

func init() {
	// relate this model to its corresponding extractor
	v2.RegisterActorExtractor(&MinerInfo{}, ExtractMinerInfo)
	// relate the actors this model can contain to their codes
	supportedActors := cid.NewSet()
	for _, c := range miner.AllCodes() {
		supportedActors.Add(c)
	}
	v2.RegisterActorType(&MinerInfo{}, supportedActors)
}

var _ v2.LilyModel = (*MinerInfo)(nil)

type MinerInfo struct {
	Height                     abi.ChainEpoch
	StateRoot                  cid.Cid
	Miner                      address.Address
	Owner                      address.Address
	Worker                     address.Address
	ControlAddresses           []address.Address
	PendingWorkerKey           *WorkerKeyChange
	PeerID                     abi.PeerID
	Multiaddrs                 []abi.Multiaddrs
	WindowPoStProofType        abi.RegisteredPoStProof
	SectorSize                 abi.SectorSize
	WindowPoStPartitionSectors uint64
	ConsensusFaultElapsed      abi.ChainEpoch
	PendingOwnerAddress        *address.Address
}

type WorkerKeyChange struct {
	NewWorker   address.Address
	EffectiveAt abi.ChainEpoch
}

func (t *MinerInfo) Meta() v2.ModelMeta {
	return v2.ModelMeta{
		Version: 1,
		Type:    v2.ModelType(reflect.TypeOf(MinerInfo{}).Name()),
		Kind:    v2.ModelActorKind,
	}
}

func (t *MinerInfo) ChainEpochTime() v2.ChainEpochTime {
	return v2.ChainEpochTime{
		Height:    t.Height,
		StateRoot: t.StateRoot,
	}
}

func (t *MinerInfo) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := t.MarshalCBOR(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (t *MinerInfo) ToStorageBlock() (block.Block, error) {
	data, err := t.Serialize()
	if err != nil {
		return nil, err
	}

	c, err := abi.CidBuilder.Sum(data)
	if err != nil {
		return nil, err
	}

	return block.NewBlockWithCid(data, c)
}

func (t *MinerInfo) Cid() cid.Cid {
	sb, err := t.ToStorageBlock()
	if err != nil {
		fmt.Printf("%+v", t)
		panic(err)
	}

	return sb.Cid()
}

func ExtractMinerInfo(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet, a actorstate.ActorInfo) ([]v2.LilyModel, error) {
	log.Debugw("extract", zap.String("extractor", "InfoExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "InfoExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}
	ec, err := miner2.NewMinerStateExtractionContext(ctx, a, api)
	if err != nil {
		return nil, fmt.Errorf("creating miner state extraction context: %w", err)
	}

	if !ec.HasPreviousState() {
		// means this miner was created in this tipset or genesis special case
	} else if changed, err := ec.CurrState.MinerInfoChanged(ec.PrevState); err != nil {
		return nil, err
	} else if !changed {
		return nil, nil
	}
	// miner info has changed.

	curMinerInfo, err := ec.CurrState.Info()
	if err != nil {
		return nil, err
	}

	// check if there is a work key change
	wkc := new(WorkerKeyChange)
	if pendingWorkerKey := curMinerInfo.PendingWorkerKey; pendingWorkerKey != nil {
		if !pendingWorkerKey.NewWorker.Empty() {
			wkc.NewWorker = pendingWorkerKey.NewWorker
			wkc.EffectiveAt = pendingWorkerKey.EffectiveAt
		}
	}
	// if there wasn't a key change set to nil else cbor marshal will fail to marshal an empty address type.
	if wkc.NewWorker.Empty() {
		wkc = nil
	}

	out := []v2.LilyModel{
		&MinerInfo{
			Height:                     current.Height(),
			StateRoot:                  current.ParentState(),
			Miner:                      a.Address,
			Owner:                      curMinerInfo.Owner,
			Worker:                     curMinerInfo.Worker,
			ControlAddresses:           curMinerInfo.ControlAddresses,
			PendingWorkerKey:           wkc,
			PeerID:                     curMinerInfo.PeerId,
			Multiaddrs:                 curMinerInfo.Multiaddrs,
			WindowPoStProofType:        curMinerInfo.WindowPoStProofType,
			SectorSize:                 curMinerInfo.SectorSize,
			WindowPoStPartitionSectors: curMinerInfo.WindowPoStPartitionSectors,
			ConsensusFaultElapsed:      curMinerInfo.ConsensusFaultElapsed,
			PendingOwnerAddress:        curMinerInfo.PendingOwnerAddress,
		},
	}

	return out, nil
}

package v0

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/util"

	minerdiff "github.com/filecoin-project/lily/pkg/extract/actors/minerdiff/v1"

	miner "github.com/filecoin-project/specs-actors/actors/builtin/miner"
)

type Info struct{}

func (i Info) Transform(ctx context.Context, current, executed *types.TipSet, addr address.Address, change *minerdiff.StateDiffResult) (model.Persistable, error) {
	// if there is no info nothing changed.
	if change.InfoChange == nil {
		return nil, nil
	}
	// if the info was removed there is nothing to record
	if change.InfoChange.Change == core.ChangeTypeRemove {
		return nil, nil
	}
	// unmarshal the miners info to its type
	minerInfo := new(miner.MinerInfo)
	if err := minerInfo.UnmarshalCBOR(bytes.NewReader(change.InfoChange.Info.Raw)); err != nil {
		return nil, err
	}
	// wrap the versioned miner info type in an interface for reusable extraction
	out, err := util.ExtractMinerInfo(ctx, current, executed, addr, &InfoWrapper{info: minerInfo})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// InfoWrapper satisfies the interface required by ExtractMinerInfo.
type InfoWrapper struct {
	info *miner.MinerInfo
}

type WorkerKeyChangeWrapper struct {
	keys *miner.WorkerKeyChange
}

func (v *WorkerKeyChangeWrapper) NewWorker() address.Address {
	return v.keys.NewWorker
}

func (v *WorkerKeyChangeWrapper) EffectiveAt() abi.ChainEpoch {
	return v.keys.EffectiveAt
}

func (v *InfoWrapper) PendingWorkerKey() (util.WorkerKeyChanges, bool) {
	if v.info.PendingWorkerKey == nil {
		return nil, false
	}
	return &WorkerKeyChangeWrapper{keys: v.info.PendingWorkerKey}, true
}

func (v *InfoWrapper) ControlAddresses() []address.Address {
	return v.info.ControlAddresses
}

func (v *InfoWrapper) Multiaddrs() []abi.Multiaddrs {
	return v.info.Multiaddrs
}

func (v *InfoWrapper) Owner() address.Address {
	return v.info.Owner
}

func (v *InfoWrapper) Worker() address.Address {
	return v.info.Worker
}

func (v *InfoWrapper) SectorSize() abi.SectorSize {
	return v.info.SectorSize
}

func (v *InfoWrapper) PeerId() abi.PeerID {
	return v.info.PeerId
}

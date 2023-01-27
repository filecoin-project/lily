package v3

import (
	"bytes"
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"

	"github.com/filecoin-project/lily/chain/indexer/tasktype"
	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/pkg/core"
	"github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/util"
	minertypes "github.com/filecoin-project/lily/pkg/transform/timescale/actors/miner/v1/types"
	"github.com/filecoin-project/lily/pkg/transform/timescale/data"

	miner "github.com/filecoin-project/specs-actors/v3/actors/builtin/miner"
)

type Info struct{}

func (i Info) Transform(ctx context.Context, current, parent *types.TipSet, miners []*minertypes.MinerStateChange) model.Persistable {
	report := data.StartProcessingReport(tasktype.MinerInfo, current)
	for _, m := range miners {
		// if there is no info nothing changed.
		if m.StateChange.InfoChange == nil {
			continue
		}
		// if the info was removed there is nothing to record
		if m.StateChange.InfoChange.Change == core.ChangeTypeRemove {
			continue
		}
		// unmarshal the miners info to its type
		minerInfo := new(miner.MinerInfo)
		if err := minerInfo.UnmarshalCBOR(bytes.NewReader(m.StateChange.InfoChange.Info.Raw)); err != nil {
			report.AddError(err)
			continue
		}
		// wrap the versioned miner info type in an interface for reusable extraction
		infoModel, err := util.ExtractMinerInfo(ctx, current, parent, m.Address, &InfoWrapper{info: minerInfo})
		if err != nil {
			report.AddError(err)
		}
		report.AddModels(infoModel)
	}
	return report.Finish()
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

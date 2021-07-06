package extractors

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate/miner/extract"
	logging "github.com/ipfs/go-log/v2"
	"github.com/multiformats/go-multiaddr"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
)

var log = logging.Logger("miner")

func init() {
	extract.Register(&MinerInfo{}, ExtractMinerInfo)
}

func ExtractMinerInfo(ctx context.Context, ec *extract.MinerStateExtractionContext) (model.Persistable, error) {
	_, span := global.Tracer("").Start(ctx, "ExtractMinerInfo")
	defer span.End()
	if !ec.HasPreviousState() {
		// means this miner was created in this tipset or genesis special case
	} else if changed, err := ec.CurrState.MinerInfoChanged(ec.PrevState); err != nil {
		return nil, err
	} else if !changed {
		return nil, nil
	}
	// miner info has changed.

	newInfo, err := ec.CurrState.Info()
	if err != nil {
		return nil, err
	}

	var newWorker string
	if newInfo.NewWorker != address.Undef {
		newWorker = newInfo.NewWorker.String()
	}

	var newCtrlAddresses []string
	for _, addr := range newInfo.ControlAddresses {
		newCtrlAddresses = append(newCtrlAddresses, addr.String())
	}

	// best effort to decode, we have no control over what miners put in this field, its just bytes.
	var newMultiAddrs []string
	for _, addr := range newInfo.Multiaddrs {
		newMaddr, err := multiaddr.NewMultiaddrBytes(addr)
		if err == nil {
			newMultiAddrs = append(newMultiAddrs, newMaddr.String())
		} else {
			log.Debugw("failed to decode miner multiaddr", "miner", ec.Address, "multiaddress", addr, "error", err)
		}
	}
	mi := &MinerInfo{
		Height:                  int64(ec.CurrTs.Height()),
		MinerID:                 ec.Address.String(),
		StateRoot:               ec.CurrTs.ParentState().String(),
		OwnerID:                 newInfo.Owner.String(),
		WorkerID:                newInfo.Worker.String(),
		NewWorker:               newWorker,
		WorkerChangeEpoch:       int64(newInfo.WorkerChangeEpoch),
		ConsensusFaultedElapsed: int64(newInfo.ConsensusFaultElapsed),
		ControlAddresses:        newCtrlAddresses,
		MultiAddresses:          newMultiAddrs,
		SectorSize:              uint64(newInfo.SectorSize),
	}

	if newInfo.PeerId != nil {
		mi.PeerID = newInfo.PeerId.String()
	}

	return mi, nil
}

type MinerInfo struct {
	Height    int64  `pg:",pk,notnull,use_zero"`
	MinerID   string `pg:",pk,notnull"`
	StateRoot string `pg:",pk,notnull"`

	OwnerID  string `pg:",notnull"`
	WorkerID string `pg:",notnull"`

	NewWorker         string
	WorkerChangeEpoch int64 `pg:",notnull,use_zero"`

	ConsensusFaultedElapsed int64 `pg:",notnull,use_zero"`

	PeerID           string
	ControlAddresses []string
	MultiAddresses   []string

	SectorSize uint64 `pg:",notnull,use_zero"`
}

func (m *MinerInfo) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerInfoModel.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_infos"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	return s.PersistModel(ctx, m)
}

type MinerInfoList []*MinerInfo

func (ml MinerInfoList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerInfoList.Persist")
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_infos"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	if len(ml) == 0 {
		return nil
	}
	return s.PersistModel(ctx, ml)
}

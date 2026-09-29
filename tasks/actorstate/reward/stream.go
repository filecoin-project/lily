package reward

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	reward19 "github.com/filecoin-project/go-state-types/builtin/v19/reward"

	"github.com/filecoin-project/lily/chain/actors/builtin/reward"
	"github.com/filecoin-project/lily/model"
	rewardmodel "github.com/filecoin-project/lily/model/actors/reward"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

// StreamExtractor extracts per-epoch FIP-0118 reward stream weights from the reward actor.
// Reward actor versions before v19 produce no row.
type StreamExtractor struct{}

func (StreamExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "StreamExtractor"), zap.Inline(a))
	_, span := otel.Tracer("").Start(ctx, "StreamExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	rstate, err := reward.Load(node.Store(), &a.Actor)
	if err != nil {
		return nil, err
	}
	ledger, ok := rstate.(reward.StreamLedger)
	if !ok {
		return nil, nil
	}

	streams, err := ledger.LoadStreams()
	if err != nil {
		return nil, err
	}

	epoch := a.Current.Height()
	row := &rewardmodel.ChainRewardStream{
		Height:              int64(epoch),
		StateRoot:           a.Current.ParentState().String(),
		TotalMintedReward:   ledger.TotalMintedReward().String(),
		TotalBurnMinted:     ledger.TotalBurnMinted().String(),
		TotalExplicitMinted: ledger.TotalExplicitMinted().String(),
	}
	fillStreamWeights(row, streams, epoch)
	return row, nil
}

func fillStreamWeights(row *rewardmodel.ChainRewardStream, streams *reward19.StreamsState, epoch abi.ChainEpoch) {
	evaluated := make([]big.Int, 0, len(streams.Streams))
	var consensus, service *reward19.WeightRecord
	consensusWeight := big.Zero()
	serviceWeight := big.Zero()
	for i := range streams.Streams {
		st := &streams.Streams[i]
		w := reward.ComputeStreamWeight(st.Weight, epoch)
		evaluated = append(evaluated, w)
		switch st.ID {
		case reward.ConsensusStreamID:
			consensus = &st.Weight
			consensusWeight = w
		case reward.ServiceStreamID:
			service = &st.Weight
			serviceWeight = w
		}
	}

	row.BurnWeight = reward.BurnWeight(evaluated).String()
	row.ConsensusWeight = consensusWeight.String()
	row.ServiceWeight = serviceWeight.String()
	row.ConsensusVStart, row.ConsensusSlope, row.ConsensusTStart, row.ConsensusFloor, row.ConsensusCap = weightFields(consensus)
	row.ServiceVStart, row.ServiceSlope, row.ServiceTStart, row.ServiceFloor, row.ServiceCap = weightFields(service)
}

func weightFields(w *reward19.WeightRecord) (vStart, slope string, tStart int64, floor, cap string) {
	if w == nil {
		return "0", "0", 0, "0", "0"
	}
	return big.NewIntUnsigned(w.VStart).String(),
		big.NewInt(w.Slope).String(),
		int64(w.TStart),
		big.NewIntUnsigned(w.Floor).String(),
		big.NewIntUnsigned(w.Cap).String()
}

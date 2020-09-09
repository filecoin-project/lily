package actors

import "github.com/filecoin-project/visor/model"

type MinerPower struct {
	MinerID              string `pg:",pk,notnull"`
	StateRoot            string `pg:",pk,notnull"`
	RawBytePower         string `pg:",notnull"`
	QualityAdjustedPower string `pg:",notnull"`
}

type MinerState struct {
	MinerID    string `pg:",pk,notnull"`
	OwnerID    string `pg:",notnull"`
	WorkerID   string `pg:",notnull"`
	PeerID     []byte
	SectorSize string `pg:",notnull"`
}

func NewMinerPowerModel(res model.MinerStateResult) *MinerPower {
	return &MinerPower{
		MinerID:              res.MinerAddr.String(),
		StateRoot:            res.StateRoot.String(),
		RawBytePower:         res.Claim.RawBytePower.String(),
		QualityAdjustedPower: res.Claim.QualityAdjPower.String(),
	}
}

func NewMinerStateModel(res model.MinerStateResult) *MinerState {
	return &MinerState{
		MinerID:    res.MinerAddr.String(),
		OwnerID:    res.Info.Owner.String(),
		WorkerID:   res.Info.Worker.String(),
		PeerID:     res.Info.PeerId,
		SectorSize: res.Info.SectorSize.ShortString(),
	}
}

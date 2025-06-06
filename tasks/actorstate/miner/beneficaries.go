package miner

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/filecoin-project/lily/chain/actors/builtin/miner"
	"github.com/filecoin-project/lily/model"
	minermodel "github.com/filecoin-project/lily/model/actors/miner"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

type BeneficiaryExtractor struct{}

func (BeneficiaryExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	log.Debugw("extract", zap.String("extractor", "BeneficiaryExtractor"), zap.Inline(a))
	ctx, span := otel.Tracer("").Start(ctx, "BeneficiaryExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}
	ec, err := NewMinerStateExtractionContext(ctx, a, node)
	if err != nil {
		return nil, fmt.Errorf("creating miner state extraction context: %w", err)
	}

	if !ec.HasPreviousState() {
		// means this miner was created in this tipset, persist current state.
		curInfo, err := ec.CurrState.Info()
		if err != nil {
			return nil, err
		}
		if curInfo.Beneficiary.Empty() {
			return nil, nil
		}
		var (
			newBeneficiary        string
			newQuota              string
			newExpiration         int64
			approvedByBeneficiary bool
			approvedByNominee     bool
		)
		if curInfo.PendingBeneficiaryTerm != nil {
			if !curInfo.PendingBeneficiaryTerm.NewBeneficiary.Empty() {
				newBeneficiary = curInfo.PendingBeneficiaryTerm.NewBeneficiary.String()
			}
			if !curInfo.PendingBeneficiaryTerm.NewQuota.Nil() {
				newQuota = curInfo.PendingBeneficiaryTerm.NewQuota.String()
			}
			newExpiration = int64(curInfo.PendingBeneficiaryTerm.NewExpiration)
			approvedByBeneficiary = curInfo.PendingBeneficiaryTerm.ApprovedByBeneficiary
			approvedByNominee = curInfo.PendingBeneficiaryTerm.ApprovedByNominee
		}
		return &minermodel.MinerBeneficiary{
			Height:                int64(a.Current.Height()),
			StateRoot:             a.Current.ParentState().String(),
			MinerID:               a.Address.String(),
			Beneficiary:           curInfo.Beneficiary.String(),
			Quota:                 curInfo.BeneficiaryTerm.Quota.String(),
			UsedQuota:             curInfo.BeneficiaryTerm.UsedQuota.String(),
			Expiration:            int64(curInfo.BeneficiaryTerm.Expiration),
			NewBeneficiary:        newBeneficiary,
			NewQuota:              newQuota,
			NewExpiration:         newExpiration,
			ApprovedByBeneficiary: approvedByBeneficiary,
			ApprovedByNominee:     approvedByNominee,
		}, nil
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
	oldInfo, err := ec.PrevState.Info()
	if err != nil {
		return nil, err
	}

	// check if beneficiary data has changed.
	term, termChanged := minerBeneficiaryTermChanged(oldInfo, newInfo)

	pending, pendingChanged := minerPendingBeneficiaryChanged(oldInfo, newInfo)

	// nothing changed, bail
	if !termChanged && !pendingChanged {
		return nil, nil
	}

	// model has changed, persist
	bene := &minermodel.MinerBeneficiary{
		Height:      int64(a.Current.Height()),
		StateRoot:   a.Current.ParentState().String(),
		MinerID:     a.Address.String(),
		Beneficiary: newInfo.Beneficiary.String(),
	}

	// if there are pending changes persist them, and also ensure non-nil fields (quota, usedQuota, expiration) are populated with the latest values if if unchanged.
	if pendingChanged {
		bene.NewBeneficiary = pending.NewBeneficiary
		bene.NewQuota = pending.NewQuota
		bene.NewExpiration = pending.NewExpiration
		bene.ApprovedByBeneficiary = pending.ApprovedByBeneficiary
		bene.ApprovedByNominee = pending.ApprovedByNominee

		// these fields are non-nullable so we must persist them (even if unchanged) when the pending changes.
		bene.Quota = newInfo.BeneficiaryTerm.Quota.String()
		bene.UsedQuota = newInfo.BeneficiaryTerm.UsedQuota.String()
		bene.Expiration = int64(newInfo.BeneficiaryTerm.Expiration)
	}
	// ensure the latest values are used if these changed.
	if termChanged {
		bene.Quota = term.quota
		bene.UsedQuota = term.usedQuota
		bene.Expiration = term.expiration
	}
	return bene, nil
}

type beneficiaryTermChanges struct {
	quota      string
	usedQuota  string
	expiration int64
}

func minerBeneficiaryTermChanged(oldInfo, newInfo miner.MinerInfo) (*beneficiaryTermChanges, bool) {
	// are they identical?
	if oldInfo.Beneficiary == newInfo.Beneficiary &&
		oldInfo.BeneficiaryTerm.Quota.Equals(newInfo.BeneficiaryTerm.Quota) &&
		oldInfo.BeneficiaryTerm.UsedQuota.Equals(newInfo.BeneficiaryTerm.UsedQuota) &&
		oldInfo.BeneficiaryTerm.Expiration == newInfo.BeneficiaryTerm.Expiration {
		// not changed
		return nil, false
	}

	// changed.
	return &beneficiaryTermChanges{
		quota:      newInfo.BeneficiaryTerm.Quota.String(),
		usedQuota:  newInfo.BeneficiaryTerm.UsedQuota.String(),
		expiration: int64(newInfo.BeneficiaryTerm.Expiration),
	}, true
}

type pendingBeneficiaryChanges struct {
	NewBeneficiary        string
	NewQuota              string
	NewExpiration         int64
	ApprovedByBeneficiary bool
	ApprovedByNominee     bool
}

func minerPendingBeneficiaryChanged(oldInfo, newInfo miner.MinerInfo) (*pendingBeneficiaryChanges, bool) {
	// if both nil there is no change
	if oldInfo.PendingBeneficiaryTerm == nil && newInfo.PendingBeneficiaryTerm == nil {
		return nil, false
	}
	// at least one of them isn't nil, something changed
	// if they are both non-nil check if their contents differs
	if oldInfo.PendingBeneficiaryTerm != nil && newInfo.PendingBeneficiaryTerm != nil {
		// are they identical?
		if oldInfo.PendingBeneficiaryTerm.ApprovedByBeneficiary == newInfo.PendingBeneficiaryTerm.ApprovedByBeneficiary &&
			oldInfo.PendingBeneficiaryTerm.ApprovedByNominee == newInfo.PendingBeneficiaryTerm.ApprovedByNominee &&
			oldInfo.PendingBeneficiaryTerm.NewBeneficiary == newInfo.PendingBeneficiaryTerm.NewBeneficiary &&
			oldInfo.PendingBeneficiaryTerm.NewExpiration == newInfo.PendingBeneficiaryTerm.NewExpiration &&
			oldInfo.PendingBeneficiaryTerm.NewQuota.Equals(newInfo.PendingBeneficiaryTerm.NewQuota) {
			return nil, false
		}
		// at least one field differs and both are non-nil, return the latest.
		return &pendingBeneficiaryChanges{
			NewBeneficiary:        newInfo.PendingBeneficiaryTerm.NewBeneficiary.String(),
			NewQuota:              newInfo.PendingBeneficiaryTerm.NewQuota.String(),
			NewExpiration:         int64(newInfo.PendingBeneficiaryTerm.NewExpiration),
			ApprovedByBeneficiary: newInfo.PendingBeneficiaryTerm.ApprovedByBeneficiary,
			ApprovedByNominee:     newInfo.PendingBeneficiaryTerm.ApprovedByNominee,
		}, true
	}
	// we know at least 1 struct is non-nil

	// new is empty, then old was populated, return struct with null-able sql values
	if newInfo.PendingBeneficiaryTerm == nil {
		return &pendingBeneficiaryChanges{
			NewBeneficiary:        "",
			NewQuota:              "",
			NewExpiration:         0,
			ApprovedByBeneficiary: false,
			ApprovedByNominee:     false,
		}, true
	} // else
	// new is non-empty, old was empty, return latest values.
	return &pendingBeneficiaryChanges{
		NewBeneficiary:        newInfo.PendingBeneficiaryTerm.NewBeneficiary.String(),
		NewQuota:              newInfo.PendingBeneficiaryTerm.NewQuota.String(),
		NewExpiration:         int64(newInfo.PendingBeneficiaryTerm.NewExpiration),
		ApprovedByBeneficiary: newInfo.PendingBeneficiaryTerm.ApprovedByBeneficiary,
		ApprovedByNominee:     newInfo.PendingBeneficiaryTerm.ApprovedByNominee,
	}, true
}

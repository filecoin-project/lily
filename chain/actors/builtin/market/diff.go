package market

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-amt-ipld/v3"
	"github.com/filecoin-project/go-state-types/abi"

	builtin0 "github.com/filecoin-project/specs-actors/actors/builtin"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	builtin3 "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	market3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/market"
	builtin4 "github.com/filecoin-project/specs-actors/v4/actors/builtin"
	market4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/market"

	"github.com/filecoin-project/sentinel-visor/chain/actors/adt"
	"github.com/filecoin-project/sentinel-visor/chain/actors/adt/diff"
)

var log = logging.Logger("actor/marker")

func DiffDealProposals(ctx context.Context, store adt.Store, pre, cur State) (*DealProposalChanges, error) {
	preOpts, err := ProposalsAmtBitwidth(pre.Code())
	if err != nil {
		return nil, err
	}
	curOpts, err := ProposalsAmtBitwidth(cur.Code())
	if err != nil {
		return nil, err
	}
	preP, err := pre.Proposals()
	if err != nil {
		return nil, err
	}
	curP, err := cur.Proposals()
	if err != nil {
		return nil, err
	}

	diffContainer := NewMarketProposalsDiffContainer(preP, curP)
	if requiresLegacyDiffing(pre, cur, preOpts, curOpts) {
		log.Warn("actor AMT opts differ, running slower generic array diff", "preCID", pre.Code(), "curCID", cur.Code())
		if err := diff.CompareArray(preP.array(), curP.array(), diffContainer); err != nil {
			return nil, fmt.Errorf("diffing deal states: %w", err)
		}
		return diffContainer.Results, nil
	}

	changes, err := diff.Amt(ctx, preP.array(), curP.array(), store, store, amt.UseTreeBitWidth(uint(preOpts)))
	if err != nil {
		return nil, err
	}

	for _, change := range changes {
		switch change.Type {
		case amt.Add:
			if err := diffContainer.Add(change.Key, change.After); err != nil {
				return nil, err
			}
		case amt.Remove:
			if err := diffContainer.Remove(change.Key, change.Before); err != nil {
				return nil, err
			}
		case amt.Modify:
			if err := diffContainer.Modify(change.Key, change.Before, change.After); err != nil {
				return nil, err
			}
		}
	}

	return diffContainer.Results, nil
}

func NewMarketProposalsDiffContainer(pre, cur DealProposals) *marketProposalsDiffContainer {
	return &marketProposalsDiffContainer{
		Results: new(DealProposalChanges),
		pre:     pre,
		cur:     cur,
	}
}

type marketProposalsDiffContainer struct {
	Results  *DealProposalChanges
	pre, cur DealProposals
}

func (d *marketProposalsDiffContainer) Add(key uint64, val *cbg.Deferred) error {
	dp, err := d.cur.decode(val)
	if err != nil {
		return err
	}
	d.Results.Added = append(d.Results.Added, ProposalIDState{abi.DealID(key), *dp})
	return nil
}

func (d *marketProposalsDiffContainer) Modify(key uint64, before, after *cbg.Deferred) error {
	// short circuit, DealProposals are static
	return nil
}

func (d *marketProposalsDiffContainer) Remove(key uint64, val *cbg.Deferred) error {
	dp, err := d.pre.decode(val)
	if err != nil {
		return err
	}
	d.Results.Removed = append(d.Results.Removed, ProposalIDState{abi.DealID(key), *dp})
	return nil
}

func DiffDealStates(ctx context.Context, store adt.Store, pre, cur State) (*DealStateChanges, error) {
	preOpts, err := StatesAmtBitwidth(pre.Code())
	if err != nil {
		return nil, err
	}
	curOpts, err := StatesAmtBitwidth(cur.Code())
	if err != nil {
		return nil, err
	}
	preS, err := pre.States()
	if err != nil {
		return nil, err
	}
	curS, err := cur.States()
	if err != nil {
		return nil, err
	}

	diffContainer := NewMarketStatesDiffContainer(preS, curS)
	if requiresLegacyDiffing(pre, cur, preOpts, curOpts) {
		log.Warn("actor AMT opts differ, running slower generic array diff", "preCID", pre.Code(), "curCID", cur.Code())
		if err := diff.CompareArray(preS.array(), curS.array(), diffContainer); err != nil {
			return nil, fmt.Errorf("diffing deal states: %w", err)
		}
		return diffContainer.Results, nil
	}

	changes, err := diff.Amt(ctx, preS.array(), curS.array(), store, store, amt.UseTreeBitWidth(uint(preOpts)))
	if err != nil {
		return nil, err
	}

	for _, change := range changes {
		switch change.Type {
		case amt.Add:
			if err := diffContainer.Add(change.Key, change.After); err != nil {
				return nil, err
			}
		case amt.Remove:
			if err := diffContainer.Remove(change.Key, change.Before); err != nil {
				return nil, err
			}
		case amt.Modify:
			if err := diffContainer.Modify(change.Key, change.Before, change.After); err != nil {
				return nil, err
			}
		}
	}

	return diffContainer.Results, nil
}

func requiresLegacyDiffing(pre, cur State, pOpts, cOpts int) bool {
	// amt/v3 cannot read amt/v2 nodes. Their Pointers struct has changed cbor marshalers.
	if pre.Code() == builtin0.StorageMarketActorCodeID {
		return true
	}
	if pre.Code() == builtin2.StorageMarketActorCodeID {
		return true
	}
	if cur.Code() == builtin0.StorageMarketActorCodeID {
		return true
	}
	if cur.Code() == builtin2.StorageMarketActorCodeID {
		return true
	}
	// bitwidth differences requires legacy diffing.
	if pOpts != cOpts {
		return true
	}
	return false
}

func NewMarketStatesDiffContainer(pre, cur DealStates) *marketStatesDiffContainer {
	return &marketStatesDiffContainer{
		Results: new(DealStateChanges),
		pre:     pre,
		cur:     cur,
	}
}

type marketStatesDiffContainer struct {
	Results  *DealStateChanges
	pre, cur DealStates
}

func (d *marketStatesDiffContainer) Add(key uint64, val *cbg.Deferred) error {
	ds, err := d.cur.decode(val)
	if err != nil {
		return err
	}
	d.Results.Added = append(d.Results.Added, DealIDState{abi.DealID(key), *ds})
	return nil
}

func (d *marketStatesDiffContainer) Modify(key uint64, from, to *cbg.Deferred) error {
	dsFrom, err := d.pre.decode(from)
	if err != nil {
		return err
	}
	dsTo, err := d.cur.decode(to)
	if err != nil {
		return err
	}
	if *dsFrom != *dsTo {
		d.Results.Modified = append(d.Results.Modified, DealStateChange{abi.DealID(key), dsFrom, dsTo})
	}
	return nil
}

func (d *marketStatesDiffContainer) Remove(key uint64, val *cbg.Deferred) error {
	ds, err := d.pre.decode(val)
	if err != nil {
		return err
	}
	d.Results.Removed = append(d.Results.Removed, DealIDState{abi.DealID(key), *ds})
	return nil
}

func ProposalsAmtBitwidth(c cid.Cid) (int, error) {
	switch c {
	case builtin0.StorageMarketActorCodeID:
		// https://github.com/filecoin-project/go-amt-ipld/blob/v2.1.0/amt.go#L21
		return 3, nil
	case builtin2.StorageMarketActorCodeID:
		// https://github.com/filecoin-project/go-amt-ipld/blob/v2.1.0/amt.go#L21
		return 3, nil
	case builtin3.StorageMarketActorCodeID:
		return market3.ProposalsAmtBitwidth, nil
	case builtin4.StorageMarketActorCodeID:
		return market4.ProposalsAmtBitwidth, nil
	}
	return -1, xerrors.Errorf("unknown actor code: %s", c)
}

func StatesAmtBitwidth(c cid.Cid) (int, error) {
	switch c {
	case builtin0.StorageMarketActorCodeID:
		// https://github.com/filecoin-project/go-amt-ipld/blob/v2.1.0/amt.go#L21
		return 3, nil
	case builtin2.StorageMarketActorCodeID:
		// https://github.com/filecoin-project/go-amt-ipld/blob/v2.1.0/amt.go#L21
		return 3, nil
	case builtin3.StorageMarketActorCodeID:
		return market3.StatesAmtBitwidth, nil
	case builtin4.StorageMarketActorCodeID:
		return market4.StatesAmtBitwidth, nil
	}
	return -1, xerrors.Errorf("unknown actor code: %s", c)
}

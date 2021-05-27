package power

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-hamt-ipld/v3"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/sentinel-visor/chain/actors/adt"
	"github.com/filecoin-project/sentinel-visor/chain/actors/adt/diff"
	builtin0 "github.com/filecoin-project/specs-actors/actors/builtin"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	cbg "github.com/whyrusleeping/cbor-gen"
)

type ClaimChanges struct {
	Added    []ClaimInfo
	Modified []ClaimModification
	Removed  []ClaimInfo
}

type ClaimModification struct {
	Miner address.Address
	From  Claim
	To    Claim
}

type ClaimInfo struct {
	Miner address.Address
	Claim Claim
}

func DiffClaims(ctx context.Context, store adt.Store, pre, cur State) (*ClaimChanges, error) {
	prec, err := pre.claims()
	if err != nil {
		return nil, err
	}

	curc, err := cur.claims()
	if err != nil {
		return nil, err
	}

	preOpts, err := adt.MapOptsForActorCode(pre.Code())
	if err != nil {
		return nil, err
	}

	curOpts, err := adt.MapOptsForActorCode(cur.Code())
	if err != nil {
		return nil, err
	}

	diffContainer := NewClaimDiffContainer(pre, cur)

	if requiresLegacyDiffing(pre, cur, preOpts, curOpts) {
		if err := diff.CompareMap(prec, curc, diffContainer); err != nil {
			return nil, err
		}
		return diffContainer.Results, nil
	}

	changes, err := diff.Hamt(ctx, prec, curc, store, store, hamt.UseTreeBitWidth(preOpts.Bitwidth), hamt.UseHashFunction(hamt.HashFunc(preOpts.HashFunc)))
	if err != nil {
		return nil, err
	}

	for _, change := range changes {
		switch change.Type {
		case hamt.Add:
			if err := diffContainer.Add(change.Key, change.After); err != nil {
				return nil, err
			}
		case hamt.Remove:
			if err := diffContainer.Remove(change.Key, change.Before); err != nil {
				return nil, err
			}
		case hamt.Modify:
			if err := diffContainer.Modify(change.Key, change.Before, change.After); err != nil {
				return nil, err
			}
		}
	}

	return diffContainer.Results, nil
}

func NewClaimDiffContainer(pre, cur State) *claimDiffContainer {
	return &claimDiffContainer{
		Results: new(ClaimChanges),
		pre:     pre,
		after:   cur,
	}
}

type claimDiffContainer struct {
	Results    *ClaimChanges
	pre, after State
}

func (c *claimDiffContainer) AsKey(key string) (abi.Keyer, error) {
	addr, err := address.NewFromBytes([]byte(key))
	if err != nil {
		return nil, err
	}
	return abi.AddrKey(addr), nil
}

func (c *claimDiffContainer) Add(key string, val *cbg.Deferred) error {
	ci, err := c.after.decodeClaim(val)
	if err != nil {
		return err
	}
	addr, err := address.NewFromBytes([]byte(key))
	if err != nil {
		return err
	}
	c.Results.Added = append(c.Results.Added, ClaimInfo{
		Miner: addr,
		Claim: ci,
	})
	return nil
}

func (c *claimDiffContainer) Modify(key string, from, to *cbg.Deferred) error {
	ciFrom, err := c.pre.decodeClaim(from)
	if err != nil {
		return err
	}

	ciTo, err := c.after.decodeClaim(to)
	if err != nil {
		return err
	}

	addr, err := address.NewFromBytes([]byte(key))
	if err != nil {
		return err
	}

	if ciFrom != ciTo {
		c.Results.Modified = append(c.Results.Modified, ClaimModification{
			Miner: addr,
			From:  ciFrom,
			To:    ciTo,
		})
	}
	return nil
}

func (c *claimDiffContainer) Remove(key string, val *cbg.Deferred) error {
	ci, err := c.after.decodeClaim(val)
	if err != nil {
		return err
	}
	addr, err := address.NewFromBytes([]byte(key))
	if err != nil {
		return err
	}
	c.Results.Removed = append(c.Results.Removed, ClaimInfo{
		Miner: addr,
		Claim: ci,
	})
	return nil
}

func requiresLegacyDiffing(pre, cur State, pOpts, cOpts *adt.MapOpts) bool {
	// hamt/v3 cannot read hamt/v2 nodes. Their Pointers struct has changed cbor marshalers.
	if pre.Code() == builtin0.StoragePowerActorCodeID {
		return true
	}
	if pre.Code() == builtin2.StoragePowerActorCodeID {
		return true
	}
	if cur.Code() == builtin0.StoragePowerActorCodeID {
		return true
	}
	if cur.Code() == builtin2.StoragePowerActorCodeID {
		return true
	}
	// bitwidth or hashfunction differences mean legacy diffing.
	if !pOpts.Equal(cOpts) {
		return true
	}
	return false
}

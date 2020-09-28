package actorstate

import (
	"context"

	"go.opentelemetry.io/otel/api/global"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lotus/chain/events/state"
	"github.com/filecoin-project/specs-actors/actors/builtin"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
	initmodel "github.com/filecoin-project/sentinel-visor/model/actors/init"
)

// was services/processor/tasks/init/init_actor.go

// InitExtracter extracts init actor state
type InitExtracter struct{}

func init() {
	Register(builtin.InitActorCodeID, InitExtracter{})
}

func (InitExtracter) Extract(ctx context.Context, a ActorInfo, node lens.API) (model.Persistable, error) {
	ctx, span := global.Tracer("").Start(ctx, "InitExtracter")
	defer span.End()

	pred := state.NewStatePredicates(node)
	stateDiff := pred.OnInitActorChange(pred.OnAddressMapChange())
	changed, val, err := stateDiff(ctx, a.ParentTipSet, a.TipSet)
	if err != nil {
		return nil, err
	}
	if !changed {
		return nil, xerrors.Errorf("no state change detected")
	}
	changes, ok := val.(*state.InitActorAddressChanges)
	if !ok {
		return nil, xerrors.Errorf("unknown type returned by init actor hamt predicate: %T", val)
	}

	out := make(initmodel.IdAddressList, 0, len(changes.Added)+len(changes.Modified))
	for _, add := range changes.Added {
		out = append(out, &initmodel.IdAddress{
			ID:        add.ID.String(),
			Address:   add.PK.String(),
			StateRoot: a.ParentStateRoot.String(),
		})
	}
	for _, mod := range changes.Modified {
		out = append(out, &initmodel.IdAddress{
			ID:        mod.To.ID.String(),
			Address:   mod.To.PK.String(),
			StateRoot: a.ParentStateRoot.String(),
		})
	}

	return out, nil
}

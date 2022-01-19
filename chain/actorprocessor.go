package chain

import (
	"context"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lily/chain/actors/builtin"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/model"
	visormodel "github.com/filecoin-project/lily/model/visor"
	"github.com/filecoin-project/lily/tasks/actorstate"
	"github.com/filecoin-project/lotus/chain/types"
)

var _ ActorExtractorProcessor = (*ActorExtractorProcessorImpl)(nil)

func NewActorExtractorProcessorImpl(api lens.API) *ActorExtractorProcessorImpl {
	return &ActorExtractorProcessorImpl{api: api}
}

type ActorExtractorProcessorImpl struct {
	api lens.API
}

type ActorStateForExtraction struct {
	addr   address.Address
	change lens.ActorStateChange
}

func (a *ActorExtractorProcessorImpl) ProcessActorExtractors(ctx context.Context, current *types.TipSet, previous *types.TipSet, actors map[address.Address]lens.ActorStateChange, extractors []ActorStateExtractor) (model.Persistable, *visormodel.ProcessingReport, error) {
	report := &visormodel.ProcessingReport{
		Height:    int64(current.Height()),
		StateRoot: current.ParentState().String(),
		Status:    visormodel.ProcessingStatusOK,
	}

	inFlight := 0
	things := make(map[ActorStateForExtraction][]ActorStateExtractor)
	for _, e := range extractors {
		for addr, ch := range actors {
			if e.Allow(ch.Actor.Code) {
				stuff := ActorStateForExtraction{
					addr:   addr,
					change: ch,
				}
				things[stuff] = append(things[stuff], e)
				inFlight++
			}
		}
	}
	// TODO check if there is anything to run extraction for
	results := make(chan *actorstate.ActorStateResult, inFlight)
	data := make(model.PersistableList, 0, inFlight)
	errorsDetected := make([]*actorstate.ActorStateError, 0, inFlight)
	skippedActors := 0

	for toExtract, exs := range things {
		go a.runActorExtractor(ctx, current, previous, toExtract, exs, results)
	}

	for inFlight > 0 {
		res := <-results
		inFlight--

		if res.Error != nil {
			// TODO I don't thinks this is actually used anywhere
			errorsDetected = append(errorsDetected, &actorstate.ActorStateError{
				Code:    res.Code.String(),
				Name:    builtin.ActorNameByCode(res.Code),
				Head:    res.Head.String(),
				Address: res.Address,
				Error:   res.Error.Error(),
			})
			continue
		}

		if res.SkippedParse {
			skippedActors++
		}

		data = append(data, res.Data)
	}

	if skippedActors > 0 {
		report.StatusInformation = fmt.Sprintf("did not parse %d actors", skippedActors)
	}

	if len(errorsDetected) != 0 {
		report.ErrorsDetected = errorsDetected
	}

	return data, report, nil
}

func (a *ActorExtractorProcessorImpl) runActorExtractor(ctx context.Context, current, previous *types.TipSet, state ActorStateForExtraction, extractors []ActorStateExtractor, results chan *actorstate.ActorStateResult) {
	info := actorstate.ActorInfo{
		Actor:           state.change.Actor,
		ChangeType:      state.change.ChangeType,
		Address:         state.addr,
		ParentStateRoot: current.ParentState(),
		Epoch:           current.Height(),
		TipSet:          current,
		ParentTipSet:    previous,
	}

	for _, ex := range extractors {
		res := &actorstate.ActorStateResult{
			Code:    info.Actor.Code,
			Head:    info.Actor.Head,
			Address: info.Address.String(),
		}
		data, err := ex.Extract(ctx, info, a.api)
		if err != nil {
			res.Error = err
		} else {
			res.Data = data
		}
		results <- res
	}
}

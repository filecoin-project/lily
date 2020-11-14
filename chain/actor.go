package chain

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
	visormodel "github.com/filecoin-project/sentinel-visor/model/visor"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate"
)

type ActorStateProcessor struct {
	node          lens.API
	opener        lens.APIOpener
	closer        lens.APICloser
	extractRaw    bool
	extractParsed bool
	lastTipSet    *types.TipSet
	lastStateTree *state.StateTree
}

func NewActorStateProcessor(opener lens.APIOpener, extractRaw bool, extractParsed bool) *ActorStateProcessor {
	p := &ActorStateProcessor{
		opener:        opener,
		extractRaw:    extractRaw,
		extractParsed: extractParsed,
	}
	return p
}

func (p *ActorStateProcessor) ProcessTipSet(ctx context.Context, ts *types.TipSet) (model.PersistableWithTx, *visormodel.ProcessingReport, error) {
	if p.node == nil {
		node, closer, err := p.opener.Open(ctx)
		if err != nil {
			return nil, nil, xerrors.Errorf("unable to open lens: %w", err)
		}
		p.node = node
		p.closer = closer
	}

	var data model.PersistableWithTx
	var report *visormodel.ProcessingReport
	var err error

	stateTree, err := state.LoadStateTree(p.node.Store(), ts.ParentState())
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to load state tree: %w", err)
	}

	if p.lastTipSet != nil && p.lastStateTree != nil {
		if p.lastTipSet.Height() > ts.Height() {
			// last tipset seen was the child
			data, report, err = p.processStateChanges(ctx, p.lastTipSet, ts, p.lastStateTree, stateTree)
		} else if p.lastTipSet.Height() < ts.Height() {
			// last tipset seen was the parent
			data, report, err = p.processStateChanges(ctx, ts, p.lastTipSet, stateTree, p.lastStateTree)
		} else {
			log.Errorw("out of order tipsets", "height", ts.Height(), "last_height", p.lastTipSet.Height())
		}
	}

	p.lastTipSet = ts
	p.lastStateTree = stateTree

	if err != nil {
		log.Errorw("error received while processing actors, closing lens", "error", err)
		if cerr := p.Close(); cerr != nil {
			log.Errorw("error received while closing lens", "error", cerr)
		}
	}
	return data, report, err
}

func (p *ActorStateProcessor) processStateChanges(ctx context.Context, ts *types.TipSet, pts *types.TipSet, stateTree *state.StateTree, parentStateTree *state.StateTree) (model.PersistableWithTx, *visormodel.ProcessingReport, error) {
	log.Debugw("processing state changes", "height", ts.Height(), "parent_height", pts.Height())

	report := &visormodel.ProcessingReport{
		Height:    int64(ts.Height()),
		StateRoot: ts.ParentState().String(),
		Status:    visormodel.ProcessingStatusOK,
	}

	if !types.CidArrsEqual(ts.Parents().Cids(), pts.Cids()) {
		report.ErrorsDetected = xerrors.Errorf("child is not on the same chain")
		return nil, report, nil
	}

	changes, err := state.Diff(parentStateTree, stateTree)
	if err != nil {
		report.ErrorsDetected = xerrors.Errorf("failed to diff state trees: %w", err)
		return nil, report, nil
	}

	ll := log.With("height", int64(ts.Height()))

	ll.Debugw("found actor state changes", "count", len(changes))

	start := time.Now()

	// Run each task concurrently
	results := make(chan *ActorStateResult, len(changes))
	for addr, act := range changes {
		go p.runActorStateExtraction(ctx, ts, pts, addr, act, results)
	}

	data := make(PersistableWithTxList, 0, len(changes))
	errorsDetected := make([]*ActorStateError, 0, len(changes))
	skippedActors := 0

	// Gather results
	inFlight := len(changes)
	for inFlight > 0 {
		res := <-results
		inFlight--
		elapsed := time.Since(start)
		lla := log.With("height", int64(ts.Height()), "actor", actorstate.ActorNameByCode(res.Code), "address", res.Address)

		if res.Error != nil {
			lla.Errorw("actor returned with error", "error", res.Error.Error())
			errorsDetected = append(errorsDetected, &ActorStateError{
				Code:    res.Code.String(),
				Name:    actorstate.ActorNameByCode(res.Code),
				Head:    res.Head.String(),
				Address: res.Address,
				Error:   res.Error.Error(),
			})
			continue
		}

		if res.SkippedParse {
			lla.Debugw("skipped actor without extracter")
			skippedActors++
		}

		lla.Debugw("actor returned with data", "time", elapsed)
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

func (p *ActorStateProcessor) runActorStateExtraction(ctx context.Context, ts *types.TipSet, pts *types.TipSet, addrStr string, act types.Actor, results chan *ActorStateResult) {
	res := &ActorStateResult{
		Code:    act.Code,
		Head:    act.Head,
		Address: addrStr,
	}
	defer func() {
		results <- res
	}()

	addr, err := address.NewFromString(addrStr)
	if err != nil {
		res.Error = xerrors.Errorf("failed to parse address: %w", err)
		return
	}

	info := actorstate.ActorInfo{
		Actor:           act,
		Address:         addr,
		ParentStateRoot: pts.ParentState(),
		Epoch:           ts.Height(),
		TipSet:          pts.Key(),
		ParentTipSet:    pts.Parents(),
	}

	// TODO: we have the state trees available, can we optimize actor state extraction further?

	var data PersistableWithTxList

	// Extract raw state
	if p.extractRaw {
		var ae actorstate.ActorExtractor
		raw, err := ae.Extract(ctx, info, p.node)
		if err != nil {
			res.Error = xerrors.Errorf("failed to extract raw actor state: %w", err)
			return
		}
		data = append(data, raw)
	}

	if p.extractParsed {
		extracter, ok := actorstate.GetActorStateExtractor(act.Code)
		if !ok {
			res.SkippedParse = true
		} else {
			// Parse state
			parsed, err := extracter.Extract(ctx, info, p.node)
			if err != nil {
				res.Error = xerrors.Errorf("failed to extract parsed actor state: %w", err)
				return
			}

			data = append(data, parsed)
		}
	}
	res.Data = data
}

func (p *ActorStateProcessor) Close() error {
	if p.closer != nil {
		p.closer()
		p.closer = nil
	}
	p.node = nil
	return nil
}

type ActorStateResult struct {
	Code         cid.Cid
	Head         cid.Cid
	Address      string
	Error        error
	SkippedParse bool
	Data         model.PersistableWithTx
}

type ActorStateError struct {
	Code    string
	Name    string
	Head    string
	Address string
	Error   string
}

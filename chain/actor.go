package chain

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
	visormodel "github.com/filecoin-project/sentinel-visor/model/visor"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate"
)

type ActorStateProcessor struct {
	node         lens.API
	opener       lens.APIOpener
	closer       lens.APICloser
	extracterMap ActorExtractorMap
	lastTipSet   *types.TipSet
}

func NewActorStateProcessor(opener lens.APIOpener, extracterMap ActorExtractorMap) *ActorStateProcessor {
	p := &ActorStateProcessor{
		opener:       opener,
		extracterMap: extracterMap,
	}
	return p
}

func (p *ActorStateProcessor) ProcessActors(ctx context.Context, ts *types.TipSet, pts *types.TipSet, candidates map[string]types.Actor) (model.PersistableWithTx, *visormodel.ProcessingReport, error) {
	if p.node == nil {
		node, closer, err := p.opener.Open(ctx)
		if err != nil {
			return nil, nil, xerrors.Errorf("unable to open lens: %w", err)
		}
		p.node = node
		p.closer = closer
	}

	log.Debugw("processing actor state changes", "height", ts.Height(), "parent_height", pts.Height())

	report := &visormodel.ProcessingReport{
		Height:    int64(ts.Height()),
		StateRoot: ts.ParentState().String(),
		Status:    visormodel.ProcessingStatusOK,
	}

	ll := log.With("height", int64(ts.Height()))

	// Filter to just allowed actors
	actors := map[string]types.Actor{}
	for addr, act := range candidates {
		if p.extracterMap.Allow(act.Code) {
			actors[addr] = act
		}
	}

	data := make(PersistableWithTxList, 0, len(actors))
	errorsDetected := make([]*ActorStateError, 0, len(actors))
	skippedActors := 0

	if len(actors) == 0 {
		ll.Debugw("no actor state changes found")
		return data, report, nil
	}

	start := time.Now()
	ll.Debugw("found actor state changes", "count", len(actors))

	// Run each task concurrently
	results := make(chan *ActorStateResult, len(actors))
	for addr, act := range actors {
		go p.runActorStateExtraction(ctx, ts, pts, addr, act, results)
	}

	// Gather results
	inFlight := len(actors)
	for inFlight > 0 {
		res := <-results
		inFlight--
		elapsed := time.Since(start)
		lla := log.With("height", int64(ts.Height()), "actor", actorstate.ActorNameByCode(res.Code), "address", res.Address)

		if res.Error != nil {
			lla.Errorw("actor returned with error", "error", res.Error.Error())
			report.ErrorsDetected = append(errorsDetected, &ActorStateError{
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

	extracter, ok := p.extracterMap.GetExtractor(act.Code)
	if !ok {
		res.SkippedParse = true
	} else {
		// Parse state
		data, err := extracter.Extract(ctx, info, p.node)
		if err != nil {
			res.Error = xerrors.Errorf("failed to extract parsed actor state: %w", err)
			return
		}
		res.Data = data
	}
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

type ActorExtractorMap interface {
	Allow(code cid.Cid) bool
	GetExtractor(code cid.Cid) (actorstate.ActorStateExtractor, bool)
}

type RawActorExtractorMap struct{}

func (RawActorExtractorMap) Allow(code cid.Cid) bool {
	return true
}

func (RawActorExtractorMap) GetExtractor(code cid.Cid) (actorstate.ActorStateExtractor, bool) {
	return actorstate.ActorExtractor{}, true
}

type TypedActorExtractorMap struct {
	// Simplistic for now, will need to make into a slice when we have more actor versions
	CodeV1 cid.Cid
	CodeV2 cid.Cid
}

func (t *TypedActorExtractorMap) Allow(code cid.Cid) bool {
	return code == t.CodeV1 || code == t.CodeV2
}

func (t *TypedActorExtractorMap) GetExtractor(code cid.Cid) (actorstate.ActorStateExtractor, bool) {
	if !t.Allow(code) {
		return nil, false
	}
	return actorstate.GetActorStateExtractor(code)
}

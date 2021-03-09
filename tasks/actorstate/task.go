package actorstate

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"go.opencensus.io/tag"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
	visormodel "github.com/filecoin-project/sentinel-visor/model/visor"
)

// A Task processes the extraction of actor state according the allowed types in its extracter map.
type Task struct {
	nodeMu sync.Mutex // guards mutations to node, opener and closer
	node   lens.API
	opener lens.APIOpener
	closer lens.APICloser

	extracterMap ActorExtractorMap
}

func NewTask(opener lens.APIOpener, extracterMap ActorExtractorMap) *Task {
	p := &Task{
		opener:       opener,
		extracterMap: extracterMap,
	}
	return p
}

func (t *Task) ProcessActors(ctx context.Context, ts *types.TipSet, pts *types.TipSet, candidates map[string]types.Actor) (model.Persistable, *visormodel.ProcessingReport, error) {
	// t.node is used only by goroutines started by this method
	t.nodeMu.Lock()
	if t.node == nil {
		node, closer, err := t.opener.Open(ctx)
		if err != nil {
			t.nodeMu.Unlock()
			return nil, nil, xerrors.Errorf("unable to open lens: %w", err)
		}
		t.node = node
		t.closer = closer
	}
	t.nodeMu.Unlock()

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
		if t.extracterMap.Allow(act.Code) {
			actors[addr] = act
		}
	}

	data := make(model.PersistableList, 0, len(actors))
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
		go t.runActorStateExtraction(ctx, ts, pts, addr, act, results)
	}

	// Gather results
	inFlight := len(actors)
	for inFlight > 0 {
		res := <-results
		inFlight--
		elapsed := time.Since(start)
		lla := log.With("height", int64(ts.Height()), "actor", ActorNameByCode(res.Code), "address", res.Address)

		if res.Error != nil {
			lla.Errorw("actor returned with error", "error", res.Error.Error())
			report.ErrorsDetected = append(errorsDetected, &ActorStateError{
				Code:    res.Code.String(),
				Name:    ActorNameByCode(res.Code),
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

func (t *Task) runActorStateExtraction(ctx context.Context, ts *types.TipSet, pts *types.TipSet, addrStr string, act types.Actor, results chan *ActorStateResult) {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.ActorCode, ActorNameByCode(act.Code)))

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

	info := ActorInfo{
		Actor:           act,
		Address:         addr,
		ParentStateRoot: pts.ParentState(),
		Epoch:           ts.Height(),
		TipSet:          pts.Key(),
		ParentTipSet:    pts.Parents(),
	}

	extracter, ok := t.extracterMap.GetExtractor(act.Code)
	if !ok {
		res.SkippedParse = true
	} else {
		// get reference to the lens api, which may have been closed due to a failure elsewhere
		t.nodeMu.Lock()
		nodeAPI := t.node
		t.nodeMu.Unlock()

		if nodeAPI == nil {
			res.Error = xerrors.Errorf("failed to extract parsed actor state: no connection to api")
			return
		}

		// Parse state
		data, err := extracter.Extract(ctx, info, nodeAPI)
		if err != nil {
			res.Error = xerrors.Errorf("failed to extract parsed actor state: %w", err)
			return
		}
		res.Data = data
	}
}

func (t *Task) Close() error {
	t.nodeMu.Lock()
	defer t.nodeMu.Unlock()
	if t.closer != nil {
		t.closer()
		t.closer = nil
	}
	t.node = nil
	return nil
}

type ActorStateResult struct {
	Code         cid.Cid
	Head         cid.Cid
	Address      string
	Error        error
	SkippedParse bool
	Data         model.Persistable
}

type ActorStateError struct {
	Code    string
	Name    string
	Head    string
	Address string
	Error   string
}

// An ActorExtractorMap controls which actor types may be extracted.
type ActorExtractorMap interface {
	Allow(code cid.Cid) bool
	GetExtractor(code cid.Cid) (ActorStateExtractor, bool)
}

type ActorExtractorFilter interface {
	AllowAddress(addr string) bool
}

// A RawActorExtractorMap extracts all types of actors using basic actor extraction which only parses shallow state.
type RawActorExtractorMap struct{}

func (RawActorExtractorMap) Allow(code cid.Cid) bool {
	return true
}

func (RawActorExtractorMap) GetExtractor(code cid.Cid) (ActorStateExtractor, bool) {
	return ActorExtractor{}, true
}

// A TypedActorExtractorMap extracts a single type of actor using full parsing of actor state
type TypedActorExtractorMap struct {
	// Simplistic for now, will need to make into a slice when we have more actor versions
	CodeV1 cid.Cid
	CodeV2 cid.Cid
	CodeV3 cid.Cid
}

func (t *TypedActorExtractorMap) Allow(code cid.Cid) bool {
	return code == t.CodeV1 || code == t.CodeV2 || code == t.CodeV3
}

func (t *TypedActorExtractorMap) GetExtractor(code cid.Cid) (ActorStateExtractor, bool) {
	if !t.Allow(code) {
		return nil, false
	}
	return GetActorStateExtractor(code)
}

package model

import (
	"context"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"reflect"
	"sync"
)

// TODO find a home
type ActorInfo struct {
	Actor           types.Actor
	ChangeType      lens.ChangeType
	Address         address.Address
	ParentStateRoot cid.Cid
	Epoch           abi.ChainEpoch
	TipSet          *types.TipSet
	ParentTipSet    *types.TipSet
}

// TODO find a home

// ActorStateAPI is the minimal subset of lens.API that is needed for actor state extraction
type ActorStateAPI interface {
	StateGetActor(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*types.Actor, error)
	StateMinerPower(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.MinerPower, error)
	StateReadState(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.ActorState, error)
	GetExecutedAndBlockMessagesForTipset(ctx context.Context, ts, pts *types.TipSet) (*lens.TipSetMessages, error)
	Store() adt.Store
}

var (
	tsExtractorMu sync.Mutex
	tsExctractors = map[reflect.Type]TipSetStateExtractor{}

	actExtractorMu sync.Mutex
	actExtractors  = map[reflect.Type]ActorStateExtractor{}
)

type TipSetStateExtractor interface {
	Extract(ctx context.Context, current, previous *types.TipSet) (Persistable, error)
	Name() string
}

type ActorStateExtractor interface {
	Extract(ctx context.Context, actor ActorInfo, api ActorStateAPI) (Persistable, error)
	Allow(code cid.Cid) bool
	Name() string
}

func RegisterActorModelExtractor(m Persistable, e ActorStateExtractor) {
	actExtractorMu.Lock()
	defer actExtractorMu.Unlock()
	v := reflect.TypeOf(m)
	actExtractors[v] = e
}

func RegisterTipSetModelExtractor(m Persistable, e TipSetStateExtractor) {
	tsExtractorMu.Lock()
	defer tsExtractorMu.Unlock()
	v := reflect.TypeOf(m)
	tsExctractors[v] = e
}

func TipSetExtractorForModel(m Persistable) TipSetStateExtractor {
	v := reflect.TypeOf(m)
	return tsExctractors[v]
}

func ActorStateExtractorForModel(m Persistable) ActorStateExtractor {
	v := reflect.TypeOf(m)
	return actExtractors[v]
}

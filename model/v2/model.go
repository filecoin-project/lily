package v2

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/tasks"
	"github.com/filecoin-project/lily/tasks/actorstate"
)

type ModelVersion int
type ModelType string
type ModelKind string

const (
	ModelActorKind ModelKind = "actor"
	ModelTsKind    ModelKind = "tipset"
)

type ModelMeta struct {
	Version ModelVersion
	Type    ModelType
	Kind    ModelKind
}

const modelMetaSeparator = "@v"

func (mm ModelMeta) String() string {
	return fmt.Sprintf("%s%s%d", mm.Type, modelMetaSeparator, mm.Version)
}

func DecodeModelMeta(str string) (ModelMeta, error) {
	tokens := strings.Split(str, modelMetaSeparator)
	if len(tokens) != 2 {
		return ModelMeta{}, fmt.Errorf("invalid")
	}
	mv, err := strconv.ParseInt(tokens[1], 10, 64)
	if err != nil {
		return ModelMeta{}, err
	}
	mt := tokens[0]
	return ModelMeta{
		Version: ModelVersion(mv),
		Type:    ModelType(mt),
	}, nil
}

type LilyModel interface {
	cbor.Er
	Meta() ModelMeta
	ChainEpochTime() ChainEpochTime
}

type ChainEpochTime struct {
	Height    abi.ChainEpoch
	StateRoot cid.Cid
}

// TODO consider making registry functions generic

var ActorExtractorRegistry map[ModelMeta]ActorExtractorFn
var ActorTypeRegistry map[ModelMeta]*cid.Set
var ExtractorRegistry map[ModelMeta]ExtractorFn

func init() {
	ActorExtractorRegistry = make(map[ModelMeta]ActorExtractorFn)
	ActorTypeRegistry = make(map[ModelMeta]*cid.Set)
	ExtractorRegistry = make(map[ModelMeta]ExtractorFn)
}

type ActorExtractorFn func(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet, a actorstate.ActorInfo) ([]LilyModel, error)

// RegisterActorExtractor associates a model with extractor that produces it.
func RegisterActorExtractor(model LilyModel, efn ActorExtractorFn) {
	ActorExtractorRegistry[model.Meta()] = efn
}

func RegisterActorType(model LilyModel, actors *cid.Set) {
	ActorTypeRegistry[model.Meta()] = actors
}

type ExtractorFn func(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet) ([]LilyModel, error)

func RegisterExtractor(model LilyModel, efn ExtractorFn) {
	ExtractorRegistry[model.Meta()] = efn
}

func LookupExtractor(meta ModelMeta) (ExtractorFn, error) {
	efn, found := ExtractorRegistry[meta]
	if !found {
		return nil, fmt.Errorf("no extractor for %s", meta)
	}
	return efn, nil
}

func LookupActorExtractor(meta ModelMeta) (ActorExtractorFn, error) {
	efn, found := ActorExtractorRegistry[meta]
	if !found {
		return nil, fmt.Errorf("no extractor for %s", meta)
	}
	return efn, nil
}

func LookupActorTypeThing(meta ModelMeta) (*cid.Set, error) {
	actors, found := ActorTypeRegistry[meta]
	if !found {
		return nil, fmt.Errorf("no actors for %s", meta)
	}
	return actors, nil
}

package actorstate

import (
	"context"
	"github.com/filecoin-project/lily/chain/actors/builtin/verifreg"
	"github.com/filecoin-project/lily/model"
	verifregmodel "github.com/filecoin-project/lily/model/actors/verifreg"
	"github.com/ipfs/go-cid"
)

var verifiedAllowed map[cid.Cid]bool

func init() {
	verifiedAllowed = make(map[cid.Cid]bool)
	for _, c := range verifreg.AllCodes() {
		verifiedAllowed[c] = true
	}
	model.RegisterActorModelExtractor(&verifregmodel.VerifiedRegistryVerifier{}, VerifiersExtractor{})
	model.RegisterActorModelExtractor(&verifregmodel.VerifiedRegistryVerifiedClient{}, VerifiedClientsExtractor{})
}

var _ model.ActorStateExtractor = (*VerifiersExtractor)(nil)

type VerifiersExtractor struct{}

func (VerifiersExtractor) Extract(ctx context.Context, actor model.ActorInfo, api model.ActorStateAPI) (model.Persistable, error) {
	ec, err := NewVerifiedRegistryExtractorContext(ctx, ActorInfo(actor), api)
	if err != nil {
		return nil, err
	}
	return ExtractVerifiers(ctx, ec)
}

func (VerifiersExtractor) Allow(code cid.Cid) bool {
	return verifiedAllowed[code]
}

func (VerifiersExtractor) Name() string {
	return "verified_registry_verifiers"
}

var _ model.ActorStateExtractor = (*VerifiedClientsExtractor)(nil)

type VerifiedClientsExtractor struct{}

func (VerifiedClientsExtractor) Extract(ctx context.Context, actor model.ActorInfo, api model.ActorStateAPI) (model.Persistable, error) {
	ec, err := NewVerifiedRegistryExtractorContext(ctx, ActorInfo(actor), api)
	if err != nil {
		return nil, err
	}
	return ExtractVerifiedClients(ctx, ec)
}

func (VerifiedClientsExtractor) Allow(code cid.Cid) bool {
	return verifiedAllowed[code]
}

func (VerifiedClientsExtractor) Name() string {
	return "verified_registry_verified_clients"
}

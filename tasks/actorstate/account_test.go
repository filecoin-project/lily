package actorstate

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/sentinel-visor/model"
)

func TestAccountExtract(t *testing.T) {
	ae := AccountExtractor{}
	d, err := ae.Extract(context.Background(), ActorInfo{}, nil)
	if d != model.NoData {
		t.Fatal("expected not to extract any data")
	}
	if err != nil {
		t.Fatal("unexpected error", err)
	}
}

func TestAccountFilter(t *testing.T) {
	ae := AccountExtractor{}
	filteredAddr, _ := address.NewIDAddress(1138)
	if ae.Filter(ActorInfo{Address: filteredAddr}) != false {
		t.Fatal("should be false")
	}
	if ae.Filter(ActorInfo{Address: includeAddrs[0]}) != true {
		t.Fatal("should be true")
	}
}

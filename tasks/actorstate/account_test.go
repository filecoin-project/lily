package actorstate

import (
	"context"
	"testing"

	"github.com/filecoin-project/sentinel-visor/lens"
	"github.com/filecoin-project/sentinel-visor/model"
)

func TestAccountExtract(t *testing.T) {
	ae := AccountExtractor{}
	d, err := ae.Extract(context.Background(), ActorInfo{}, []*lens.ExecutedMessage{}, nil)
	if d != model.NoData {
		t.Fatal("expected not to extract any extra data")
	}
	if err != nil {
		t.Fatal("unexpected error", err)
	}
}

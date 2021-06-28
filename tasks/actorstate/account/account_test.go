package account

import (
	"context"
	"testing"

	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate/actor"
)

func TestAccountExtract(t *testing.T) {
	ae := AccountExtractor{}
	d, err := ae.Extract(context.Background(), actor.ActorInfo{}, nil)
	if d != model.NoData {
		t.Fatal("expected not to extract any extra data")
	}
	if err != nil {
		t.Fatal("unexpected error", err)
	}
}

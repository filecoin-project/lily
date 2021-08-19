package messages

import (
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	// builtin5 "github.com/filecoin-project/specs-actors/v5/actors/builtin"
	"github.com/ipfs/go-cid"
)

func TestParseMessageParams(t *testing.T) {
	testCases := []struct {
		name        string
		method      abi.MethodNum
		params      []byte
		actorCode   cid.Cid
		wantMethod  string
		wantEncoded string
		wantErr     bool
	}{

		{
			name:        "unknown actor code",
			method:      4,
			params:      nil,
			actorCode:   cid.Undef,
			wantMethod:  "Unknown",
			wantEncoded: "",
			wantErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			task := NewTask()

			to, _ := address.NewIDAddress(1)
			from, _ := address.NewIDAddress(2)

			msg := &types.Message{
				To:   to,
				From: from,

				Method: tc.method,
				Params: tc.params,
			}

			method, encoded, err := task.parseMessageParams(msg, tc.actorCode)
			switch {
			case tc.wantErr && err == nil:
				t.Errorf("got no error but wanted one")
				return
			case !tc.wantErr && err != nil:
				t.Errorf("got unexpected error: %v", err)
				return
			}

			if method != tc.wantMethod {
				t.Errorf("got method %q, wanted %q", method, tc.wantMethod)
			}
			if encoded != tc.wantEncoded {
				t.Errorf("got encoded %q, wanted %q", encoded, tc.wantEncoded)
			}
		})
	}
}

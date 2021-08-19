package messages

import (
	"encoding/base64"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	builtin3 "github.com/filecoin-project/specs-actors/v4/actors/builtin"
	builtin4 "github.com/filecoin-project/specs-actors/v4/actors/builtin"
	builtin5 "github.com/filecoin-project/specs-actors/v5/actors/builtin"
	"github.com/ipfs/go-cid"
)

func mustDecodeBase64(t *testing.T, s string) []byte {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatalf("bad base64 data: %v", err)
	}
	return b
}

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
		{
			// Derived from message bafy2bzaceah56ky4mny2qv3eg4zzjr7xxlht2bvxvcz6oozpe7k5ytjjhpezc
			name:        "issue-664",
			method:      2,
			params:      mustDecodeBase64(t, "gm5maWwvMi9tdWx0aXNpZ1hFhVUBVZOiUErA4tz2x5Zz9iV/aBZBdHxVARJCEmhAYnLXQYa/1IScyah3BRgEVQE3/ahkH/xoK6Xpa0ZbDxkY/oPxiAIA"),
			actorCode:   builtin5.InitActorCodeID,
			wantMethod:  "InitExecParams",
			wantEncoded: "",
			wantErr:     true,
		},
		{
			// Derived from message bafy2bzacebgq3cph66gsik2ii7sxweepqpcthvihh2oc2mzuglim2arwu4v4e
			name:        "issue-665",
			method:      2,
			params:      mustDecodeBase64(t, "glgkAXEAIKIYsG+CNkex11XGSGd1TapZ7E3Pnv/QS+HCi5f8FgI/WERYPwFVk6JQSsDi3PbHlnP2JX9oFkF0fAESQhJoQGJy10GGv9SEnMmodwUYBAE3/ahkH/xoK6Xpa0ZbDxkY/oPxiAAAAA=="),
			actorCode:   builtin5.InitActorCodeID,
			wantMethod:  "InitExecParams",
			wantEncoded: "",
			wantErr:     true,
		},
		{
			// Derived from message bafy2bzacedbvv6o3xbydanuzokerd5gwj7lv5anqb5wfxp7oqoctdf6xmssua
			name:        "issue-666",
			method:      2,
			params:      mustDecodeBase64(t, "WCcBcaDkAiAzgZ+v0x2W+uoEsJxtRrsaX365EEcIgSCz/gbOPO8fIoQ="),
			actorCode:   builtin5.InitActorCodeID,
			wantMethod:  "InitExecParams",
			wantEncoded: "",
			wantErr:     true,
		},
		{
			// Derived from message bafy2bzaceaubtkxigzl2pbhxc2rpvdprcfgg3b66gk7g5wzttxtmjfuy7lpq6
			name:        "issue-667",
			method:      2,
			params:      mustDecodeBase64(t, "gtgqomZzaWduZXL1ZG5hbWVOZmlsLzUvbXVsdGlzaWdYW4SCeClmMXVldnZsdzdqeG9nc2NmZW42Y2N1dGFtbGRjdmMzZ2hwaG9zM3FkYXgpZjFheTJueXo2bm16dXlzeG9tdWp5NWc3bmF5YjQ1M2doZ2hycW55Y3kCAAA="),
			actorCode:   builtin5.InitActorCodeID,
			wantMethod:  "InitExecParams",
			wantEncoded: "",
			wantErr:     true,
		},
		{
			// Derived from message bafy2bzacedqt52o7iyghmiqvvgjvcc2bw3xivspj635tl7dtlrsasreaby5ug
			name:        "issue-668",
			method:      24,
			params:      mustDecodeBase64(t, "ghIB"),
			actorCode:   builtin4.StorageMinerActorCodeID,
			wantMethod:  "DisputeWindowedPoSt",
			wantEncoded: "{\n\t\"Deadline\": 18,\n\t\"PoStIndex\": 1\n}\n",
			wantErr:     false,
		},
		{
			// Derived from message bafy2bzacebpiuu7tgya6yz56sfllpqc3rqbo5s5xl7353xeuavc53qlpb4sqw
			// Account actor is not supported for parameter parsing
			name:        "issue-663",
			method:      1,
			params:      nil,
			actorCode:   builtin3.AccountActorCodeID,
			wantMethod:  "",
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

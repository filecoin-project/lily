package messages

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lily/chain/actors/builtin"
	"github.com/filecoin-project/lily/chain/actors/builtin/market"
	miner "github.com/filecoin-project/lily/chain/actors/builtin/miner"
	market0 "github.com/filecoin-project/specs-actors/actors/builtin/market"
	"reflect"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	builtin3 "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	builtin4 "github.com/filecoin-project/specs-actors/v4/actors/builtin"
	builtin5 "github.com/filecoin-project/specs-actors/v5/actors/builtin"
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
		deepEqual   bool
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
			wantMethod:  "Exec",
			wantEncoded: "",
			wantErr:     true,
		},
		{
			// Derived from message bafy2bzacebgq3cph66gsik2ii7sxweepqpcthvihh2oc2mzuglim2arwu4v4e
			name:        "issue-665",
			method:      2,
			params:      mustDecodeBase64(t, "glgkAXEAIKIYsG+CNkex11XGSGd1TapZ7E3Pnv/QS+HCi5f8FgI/WERYPwFVk6JQSsDi3PbHlnP2JX9oFkF0fAESQhJoQGJy10GGv9SEnMmodwUYBAE3/ahkH/xoK6Xpa0ZbDxkY/oPxiAAAAA=="),
			actorCode:   builtin5.InitActorCodeID,
			wantMethod:  "Exec",
			wantEncoded: "",
			wantErr:     true,
		},
		{
			// Derived from message bafy2bzacedbvv6o3xbydanuzokerd5gwj7lv5anqb5wfxp7oqoctdf6xmssua
			name:        "issue-666",
			method:      2,
			params:      mustDecodeBase64(t, "WCcBcaDkAiAzgZ+v0x2W+uoEsJxtRrsaX365EEcIgSCz/gbOPO8fIoQ="),
			actorCode:   builtin5.InitActorCodeID,
			wantMethod:  "Exec",
			wantEncoded: "",
			wantErr:     true,
		},
		{
			// Derived from message bafy2bzaceaubtkxigzl2pbhxc2rpvdprcfgg3b66gk7g5wzttxtmjfuy7lpq6
			name:        "issue-667",
			method:      2,
			params:      mustDecodeBase64(t, "gtgqomZzaWduZXL1ZG5hbWVOZmlsLzUvbXVsdGlzaWdYW4SCeClmMXVldnZsdzdqeG9nc2NmZW42Y2N1dGFtbGRjdmMzZ2hwaG9zM3FkYXgpZjFheTJueXo2bm16dXlzeG9tdWp5NWc3bmF5YjQ1M2doZ2hycW55Y3kCAAA="),
			actorCode:   builtin5.InitActorCodeID,
			wantMethod:  "Exec",
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
			wantEncoded: `{"Deadline":18,"PoStIndex":1}`,
			wantErr:     false,
		},
		{
			// Derived from message bafy2bzacea4ndy46bfhxa4uucflufo6g3wupppumlas57l5bz4dbjyzydpy5m
			// Account actor is supported for parameter parsing it the constructor method.
			name:        "issue-663",
			method:      16,
			params:      nil,
			actorCode:   builtin3.AccountActorCodeID,
			wantMethod:  "",
			wantEncoded: "",
			wantErr:     true,
		},
		{
			// Derived from message bafy2bzacebpiuu7tgya6yz56sfllpqc3rqbo5s5xl7353xeuavc53qlpb4sqw
			// Account actor is supported for parameter parsing it the constructor method.
			// expect error since parameters we not passed but expected:
			// https://filfox.info/en/message/bafy2bzacebpiuu7tgya6yz56sfllpqc3rqbo5s5xl7353xeuavc53qlpb4sqw
			name:        "issue-709",
			method:      1,
			params:      nil,
			actorCode:   builtin3.AccountActorCodeID,
			wantMethod:  "Constructor",
			wantEncoded: ``,
			wantErr:     true,
		},
		{
			// Derived from message bafy2bzacedzfkgkgwmyhnrty3nenkmxuhlkfhskywb3olqolhxln3yeb2cklu
			// Account actor is supported for parameter parsing it the constructor method.
			name:        "issue-709",
			method:      2,
			params:      nil,
			actorCode:   builtin3.AccountActorCodeID,
			wantMethod:  "PubkeyAddress",
			wantEncoded: ``,
			wantErr:     false,
		},
		{
			// Derived from message bafy2bzaceaoyvylhmpn6foboyajbbcjvczszyjs4do7mgp4vuutbgrk5z42fu
			// Account actor methods may receive unexpect params, they should not be parsed as they will result in invalid json.
			name:        "issue-772",
			method:      2,
			params:      mustDecodeBase64(t, "dHJhbnNmZXI="),
			actorCode:   builtin3.AccountActorCodeID,
			wantMethod:  "PubkeyAddress",
			wantEncoded: ``,
			wantErr:     false,
		},
		// Derived from message bafy2bzaceclkxipts274qqchuqn4ri6w5cce26f7lbxljfpbsgalv4zeq4huk
		{
			name:      "issue-741",
			method:    4,
			actorCode: builtin3.StorageMarketActorCodeID,
			params: mustMarshalCbor(t, &market.PublishStorageDealsParams{
				Deals: []market.ClientDealProposal{
					{
						Proposal: market0.DealProposal(market.DealProposal{
							PieceCID:             mustDecodeCID(t, "baga6ea4seaqgqzxo27ongakwwef5x3cihl6fgritvgeq5akvjqij6lpgofsogiq"),
							PieceSize:            1310720,
							VerifiedDeal:         false,
							Client:               mustDecodeAddress(t, "f1nslxql4pck5pq7hddlzym3orxlx35wkepzjkm3i"),
							Provider:             mustDecodeAddress(t, "f08178"),
							Label:                `\ufffdepcids\ufffd\ufffd*X'\u0000\u0001U\ufffd\ufffd\u0002 \u0011;\u0012\ufffd\ufffd0\ufffd3\ufffdMA\ufffd\ufffd}b\ufffd\rf\ufffdmX\u001b>\ufffd\ufffdm\ufffd۬\ufffd\ufffd\ufffd`,
							StartEpoch:           475750,
							EndEpoch:             750173,
							StoragePricePerEpoch: abi.NewTokenAmount(61035),
							ProviderCollateral:   abi.NewTokenAmount(0),
							ClientCollateral:     abi.NewTokenAmount(0),
						}),
						ClientSignature: crypto.Signature{
							Type: 1,
							Data: mustDecodeBase64(t, "9a8sdvutVlu0fizD0JmqZjKJaQLj3W3ZtJ2yTReIry8kZ8cDa33V3Pe0sdZzSjz9mRdM/KPm1jL/PZhqpDeYNwE="),
						},
					},
				},
			}),
			wantMethod: "PublishStorageDeals",
			wantEncoded: `{"Deals":[{"Proposal":{"PieceCID":{"/":"baga6ea4seaqgqzxo27ongakwwef5x3cihl6fgritvgeq5akvjqij6lpgofsogiq"},"PieceSize":1310720,"VerifiedDeal":false,"Client":"f1nslxql4pck5pq7hddlzym3orxlx35wkepzjkm3i","Provider":"f08178",` +
				`"Label":"\\ufffdepcids\\ufffd\\ufffd*X'\\u0000\\u0001U\\ufffd\\ufffd\\u0002 \\u0011;\\u0012\\ufffd\\ufffd0\\ufffd3\\ufffdMA\\ufffd\\ufffd}b\\ufffd\\rf\\ufffdmX\\u001b\u003e\\ufffd\\ufffdm\\ufffd۬\\ufffd\\ufffd\\ufffd",` +
				`"StartEpoch":475750,"EndEpoch":750173,"StoragePricePerEpoch":{"Int":61035},"ProviderCollateral":{"Int":0},"ClientCollateral":{"Int":0}},"ClientSignature":{"Type":1,"Data":"9a8sdvutVlu0fizD0JmqZjKJaQLj3W3ZtJ2yTReIry8kZ8cDa33V3Pe0sdZzSjz9mRdM/KPm1jL/PZhqpDeYNwE="}}]}`,
			wantErr:   false,
			deepEqual: true,
		},
		// Derived from message bafy2bzaceagu4lpvlejnfudfiwdu6icdumtaqzblvw7ebf3kwgt3y4iskgsyq
		{
			name:      "SubmitWindowedPost",
			method:    5,
			actorCode: builtin3.StorageMinerActorCodeID,
			params: mustMarshalCbor(t, &miner.SubmitWindowedPoStParams{
				Deadline: 38,
				Partitions: []miner.PoStPartition{
					{
						Index:   0,
						Skipped: bitfield.BitField{},
					},
				},
				Proofs: []builtin.PoStProof{
					{
						PoStProof:  9,
						ProofBytes: mustDecodeBase64(t, "j67Zt7FnIDTt+WZ+JicmOvZJWNShtEUKQp2djqEamrWFJlJ5WWGfhpTmYSimapHhjnSJoQYddySoqKHw6klIY6INz0A4aHmF2xveYKLYcqKxaB6Izis7zWyw4CMLTc3GE93wBuajQ32V1qH5qBsTw3ELzUdlFNgClUhHWushxg7kvmqvtmh9lipXzGnPnrG9p68KnBp40dvhiMBVedNch/pP7cxMH5piwGIQxsn99sQVrZxfVy1+y0SN30t03yjc"),
					},
				},
				ChainCommitRand:  mustDecodeBase64(t, "VyY9gIC10V8w8C1ltrpjqNN6mkk3xgW3Stpnp2ThNW4="),
				ChainCommitEpoch: 1287345,
			}),
			wantMethod:  "SubmitWindowedPoSt",
			wantEncoded: "{\"Partitions\":[{\"Index\":0,\"Skipped\":{\"Count\":0,\"RLE\":[0]}}],\"Proofs\":[{\"PoStProof\":9,\"ProofBytes\":\"j67Zt7FnIDTt+WZ+JicmOvZJWNShtEUKQp2djqEamrWFJlJ5WWGfhpTmYSimapHhjnSJoQYddySoqKHw6klIY6INz0A4aHmF2xveYKLYcqKxaB6Izis7zWyw4CMLTc3GE93wBuajQ32V1qH5qBsTw3ELzUdlFNgClUhHWushxg7kvmqvtmh9lipXzGnPnrG9p68KnBp40dvhiMBVedNch/pP7cxMH5piwGIQxsn99sQVrZxfVy1+y0SN30t03yjc\"}],\"ChainCommitEpoch\":1287345,\"ChainCommitRand\":\"VyY9gIC10V8w8C1ltrpjqNN6mkk3xgW3Stpnp2ThNW4=\",\"Deadline\":38}",
			wantErr:     false,
			deepEqual:   true,
		},
		// Derived from message bafy2bzacea5tk2y74kqovcm564ulsufab3lzbhbpeluh2s33hxehv6rbp22na
		{
			name:      "ProveCommitSector",
			method:    7,
			actorCode: builtin3.StorageMinerActorCodeID,
			params: mustMarshalCbor(t, &miner.ProveCommitSectorParams{
				SectorNumber: 299696,
				Proof:        mustDecodeBase64(t, "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"),
			}),
			wantMethod:  "ProveCommitSector",
			wantEncoded: "{\"Proof\": \"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\", \"SectorNumber\": 299696}",
			wantErr:     false,
			deepEqual:   true,
		},
		{
			name:      "DeclareFaultsRecovered",
			method:    11,
			actorCode: builtin5.StorageMinerActorCodeID,
			params: mustMarshalCbor(t, &miner.DeclareFaultsRecoveredParams{
				Recoveries: []miner.RecoveryDeclaration{
					{
						Deadline:  1,
						Partition: 1,
						Sectors:   bitfield.NewFromSet([]uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
					},
				},
			}),
			wantMethod:  "DeclareFaultsRecovered",
			wantEncoded: "{\"Recoveries\":[{\"Deadline\":1,\"Partition\":1,\"Sectors\":{\"Count\":10,\"RLE\":[1,10]}}]}",
			wantErr:     false,
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
			if tc.deepEqual {
				// marshal the encoded to a map for deep comparison, we care that the keys are there, but not about their order in the string.
				if !reflect.DeepEqual(mustMakeMapFromJsonString(t, encoded), mustMakeMapFromJsonString(t, tc.wantEncoded)) {
					t.Errorf("got encoded %q, wanted %q", encoded, tc.wantEncoded)
				}
			} else {
				if encoded != tc.wantEncoded {
					t.Errorf("got encoded %q, wanted %q", encoded, tc.wantEncoded)
				}
			}
		})
	}
}
func mustDecodeBase64(t *testing.T, s string) []byte {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatalf("bad base64 data: %v", err)
	}
	return b
}

func mustMarshalCbor(t *testing.T, v cbor.Marshaler) []byte {
	t.Helper()
	var buf bytes.Buffer
	err := v.MarshalCBOR(&buf)
	if err != nil {
		t.Fatalf("bad cbor: %v", err)
	}
	return buf.Bytes()
}

func mustMakeMapFromJsonString(t *testing.T, str string) map[string]interface{} {
	if str == "" {
		return nil
	}
	in := []byte(str)

	var out map[string]interface{}
	err := json.Unmarshal(in, &out)
	if err != nil {
		t.Fatalf("bad json string: %v", err)
	}
	return out
}

func mustDecodeCID(t *testing.T, cidStr string) cid.Cid {
	out, err := cid.Decode(cidStr)
	if err != nil {
		t.Fatalf("bad cid string: %v", err)
	}
	return out
}

func mustDecodeAddress(t *testing.T, addrStr string) address.Address {
	out, err := address.NewFromString(addrStr)
	if err != nil {
		t.Fatalf("bad address string: %v", err)
	}
	return out
}

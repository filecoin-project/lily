package util

import (
	"strconv"

	"github.com/fxamacker/cbor/v2"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/codec/dagcbor"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/ipld/go-ipld-prime/node/bindnode"

	"github.com/filecoin-project/lotus/chain/types"
)

type KVEvent struct {
	Key   string
	Value string
}

const (
	INT    = "int"
	STRING = "string"
	CID    = "cid"
	BIGINT = "bigint"
)

// ------------ convert ------------
// https://fips.filecoin.io/FIPS/fip-0083.html
var convert = map[string]string{
	"$type":        STRING,
	"verifier":     INT,
	"client":       INT,
	"balance":      BIGINT,
	"id":           INT,
	"provider":     INT,
	"piece-cid":    CID,
	"piece-size":   INT,
	"term-min":     INT,
	"term-max":     INT,
	"expiration":   INT,
	"term-start":   INT,
	"sector":       INT,
	"unsealed-cid": CID,
}

func GenFilterFields(targetEvents []string) map[string][]types.ActorEventBlock {
	filterFields := []types.ActorEventBlock{}
	for _, filteredEvent := range targetEvents {
		fieldByte, err := ipld.Encode(basicnode.NewString(filteredEvent), dagcbor.Encode)
		if err == nil {
			filterFields = append(filterFields, types.ActorEventBlock{Codec: 0x51, Value: fieldByte})
		}
	}

	return map[string][]types.ActorEventBlock{"$type": filterFields}
}

func CborValueDecode(key string, value []byte) interface{} {
	var (
		resultSTR    string
		resultINT    int
		resultBIGINT types.BigInt
		resultCID    cid.Cid
		err          error
	)

	switch convert[key] {
	case STRING:
		err = cbor.Unmarshal(value, &resultSTR)
		if err != nil {
			log.Errorf("cbor.Unmarshal err: %v, key: %v", err, key)
			return nil
		}
		return resultSTR
	case INT:
		err = cbor.Unmarshal(value, &resultINT)
		if err != nil {
			log.Errorf("cbor.Unmarshal err: %v, key: %v", err, key)
			return nil
		}
		return resultINT
	case BIGINT:
		err = cbor.Unmarshal(value, &resultBIGINT)
		if err != nil {
			log.Errorf("cbor.Unmarshal err: %v, key: %v", err, key)
			return nil
		}
		return resultBIGINT
	case CID:
		nd, err := ipld.DecodeUsingPrototype(value, dagcbor.Decode, bindnode.Prototype((*cid.Cid)(nil), nil))
		if err != nil {
			log.Errorf("cbor.Unmarshal err: %v, key: %v, value: %v", err, key, value)
			return nil
		}
		resultCID = *bindnode.Unwrap(nd).(*cid.Cid)
		return resultCID
	}

	return nil
}

func HandleEventEntries(event *types.ActorEvent) (string, map[string]interface{}, []*KVEvent) {
	var eventsSlice []*KVEvent
	var eventType string
	actorEvent := make(map[string]interface{})
	for _, e := range event.Entries {
		if e.Codec != 0x51 { // 81
			continue
		}

		var kvEvent KVEvent
		kvEvent.Key = e.Key

		v := CborValueDecode(e.Key, e.Value)
		switch convert[e.Key] {
		case STRING:
			kvEvent.Value = v.(string)
			if kvEvent.Key == "$type" {
				eventType = kvEvent.Value
			}
		case INT:
			kvEvent.Value = strconv.Itoa(v.(int))
		case BIGINT:
			kvEvent.Value = v.(types.BigInt).String()
		case CID:
			if v != nil {
				kvEvent.Value = v.(cid.Cid).String()
			} else {
				kvEvent.Value = ""
			}
		}
		if kvEvent.Key != "$type" {
			actorEvent[kvEvent.Key] = kvEvent.Value
		}
		eventsSlice = append(eventsSlice, &kvEvent)
	}

	return eventType, actorEvent, eventsSlice
}

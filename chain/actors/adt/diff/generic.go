package diff

import (
	"bytes"

	"github.com/filecoin-project/go-state-types/abi"
	logging "github.com/ipfs/go-log/v2"
	typegen "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/lily/chain/actors/adt"
)

var log = logging.Logger("lily/chain/diff")

// ArrayDiffer generalizes adt.Array diffing by accepting a Deferred type that can unmarshalled to its corresponding struct
// in an interface implantation.
// Add should be called when a new k,v is added to the array
// Modify should be called when a value is modified in the array
// Remove should be called when a value is removed from the array
type ArrayDiffer interface {
	Add(key uint64, val *typegen.Deferred) error
	Modify(key uint64, from, to *typegen.Deferred) error
	Remove(key uint64, val *typegen.Deferred) error
}

// CompareArray accepts two *adt.Array's and an ArrayDiffer implementation. It does the following:
// - All values that exist in preArr and not in curArr are passed to ArrayDiffer.Remove()
// - All values that exist in curArr nnd not in prevArr are passed to ArrayDiffer.Add()
// - All values that exist in preArr and in curArr are passed to ArrayDiffer.Modify()
//   - It is the responsibility of ArrayDiffer.Modify() to determine if the values it was passed have been modified.
//
// If `preArr` and `curArr` are both backed by /v3/AMTs with the same bitwidth use the more efficient Amt method.
func CompareArray(preArr, curArr adt.Array, out ArrayDiffer) error {
	notNew := make(map[int64]struct{}, curArr.Length())
	prevVal := new(typegen.Deferred)

	preArrRoot, err := preArr.Root()
	if err != nil {
		return err
	}

	curArrRoot, err := curArr.Root()
	if err != nil {
		return err
	}

	if preArrRoot.Equals(curArrRoot) {
		return nil
	}

	if err := preArr.ForEach(prevVal, func(i int64) error {
		curVal := new(typegen.Deferred)
		found, err := curArr.Get(uint64(i), curVal)
		if err != nil {
			return err
		}
		if !found {
			if err := out.Remove(uint64(i), prevVal); err != nil {
				return err
			}
			return nil
		}

		// no modification
		if !bytes.Equal(prevVal.Raw, curVal.Raw) {
			if err := out.Modify(uint64(i), prevVal, curVal); err != nil {
				return err
			}
		}
		notNew[i] = struct{}{}
		return nil
	}); err != nil {
		return err
	}

	curVal := new(typegen.Deferred)
	return curArr.ForEach(curVal, func(i int64) error {
		if _, ok := notNew[i]; ok {
			return nil
		}
		return out.Add(uint64(i), curVal)
	})
}

// MapDiffer generalizes adt.Map diffing by accepting a Deferred type that can unmarshalled to its corresponding struct
// in an interface implantation.
// AsKey should return the Keyer implementation specific to the map
// Add should be called when a new k,v is added to the map
// Modify should be called when a value is modified in the map
// Remove should be called when a value is removed from the map
type MapDiffer interface {
	AsKey(key string) (abi.Keyer, error)
	Add(key string, val *typegen.Deferred) error
	Modify(key string, from, to *typegen.Deferred) error
	Remove(key string, val *typegen.Deferred) error
}

// CompareMap accepts two *adt.Map's and an MapDiffer implementation. It does the following:
// - All values that exist in preMap and not in curMap are passed to MapDiffer.Remove()
// - All values that exist in curMap nnd not in prevArr are passed to MapDiffer.Add()
// - All values that exist in preMap and in curMap are passed to MapDiffer.Modify()
//   - It is the responsibility of ArrayDiffer.Modify() to determine if the values it was passed have been modified.
//
// If `preMap` and `curMap` are both backed by /v3/HAMTs with the same bitwidth and hash function use the more efficient Hamt method.
func CompareMap(preMap, curMap adt.Map, out MapDiffer) error {
	notNew := make(map[string]struct{})
	prevVal := new(typegen.Deferred)
	if err := preMap.ForEach(prevVal, func(key string) error {
		curVal := new(typegen.Deferred)
		k, err := out.AsKey(key)
		if err != nil {
			return err
		}

		found, err := curMap.Get(k, curVal)
		if err != nil {
			return err
		}
		if !found {
			if err := out.Remove(key, prevVal); err != nil {
				return err
			}
			return nil
		}

		// no modification
		if !bytes.Equal(prevVal.Raw, curVal.Raw) {
			if err := out.Modify(key, prevVal, curVal); err != nil {
				return err
			}
		}
		notNew[key] = struct{}{}
		return nil
	}); err != nil {
		return err
	}

	curVal := new(typegen.Deferred)
	return curMap.ForEach(curVal, func(key string) error {
		if _, ok := notNew[key]; ok {
			return nil
		}
		return out.Add(key, curVal)
	})
}

// Code generated by github.com/whyrusleeping/cbor-gen. DO NOT EDIT.

package v0

import (
	"fmt"
	"io"
	"math"
	"sort"

	core "github.com/filecoin-project/lily/pkg/core"
	cid "github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	xerrors "golang.org/x/xerrors"
)

var _ = xerrors.Errorf
var _ = cid.Undef
var _ = math.E
var _ = sort.Sort

func (t *StateChange) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)

	if _, err := cw.Write([]byte{161}); err != nil {
		return err
	}

	// t.Claims (cid.Cid) (struct)
	if len("claims") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"claims\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("claims"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("claims")); err != nil {
		return err
	}

	if t.Claims == nil {
		if _, err := cw.Write(cbg.CborNull); err != nil {
			return err
		}
	} else {
		if err := cbg.WriteCid(cw, *t.Claims); err != nil {
			return xerrors.Errorf("failed to write cid field t.Claims: %w", err)
		}
	}

	return nil
}

func (t *StateChange) UnmarshalCBOR(r io.Reader) (err error) {
	*t = StateChange{}

	cr := cbg.NewCborReader(r)

	maj, extra, err := cr.ReadHeader()
	if err != nil {
		return err
	}
	defer func() {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
	}()

	if maj != cbg.MajMap {
		return fmt.Errorf("cbor input should be of type map")
	}

	if extra > cbg.MaxLength {
		return fmt.Errorf("StateChange: map struct too large (%d)", extra)
	}

	var name string
	n := extra

	for i := uint64(0); i < n; i++ {

		{
			sval, err := cbg.ReadString(cr)
			if err != nil {
				return err
			}

			name = string(sval)
		}

		switch name {
		// t.Claims (cid.Cid) (struct)
		case "claims":

			{

				b, err := cr.ReadByte()
				if err != nil {
					return err
				}
				if b != cbg.CborNull[0] {
					if err := cr.UnreadByte(); err != nil {
						return err
					}

					c, err := cbg.ReadCid(cr)
					if err != nil {
						return xerrors.Errorf("failed to read cid field t.Claims: %w", err)
					}

					t.Claims = &c
				}

			}

		default:
			// Field doesn't exist on this type, so ignore it
			cbg.ScanForLinks(r, func(cid.Cid) {})
		}
	}

	return nil
}
func (t *ClaimsChange) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)

	if _, err := cw.Write([]byte{164}); err != nil {
		return err
	}

	// t.Miner ([]uint8) (slice)
	if len("miner") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"miner\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("miner"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("miner")); err != nil {
		return err
	}

	if len(t.Miner) > cbg.ByteArrayMaxLen {
		return xerrors.Errorf("Byte array in field t.Miner was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajByteString, uint64(len(t.Miner))); err != nil {
		return err
	}

	if _, err := cw.Write(t.Miner[:]); err != nil {
		return err
	}

	// t.Current (typegen.Deferred) (struct)
	if len("current") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"current\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("current"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("current")); err != nil {
		return err
	}

	if err := t.Current.MarshalCBOR(cw); err != nil {
		return err
	}

	// t.Previous (typegen.Deferred) (struct)
	if len("previous") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"previous\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("previous"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("previous")); err != nil {
		return err
	}

	if err := t.Previous.MarshalCBOR(cw); err != nil {
		return err
	}

	// t.Change (core.ChangeType) (uint8)
	if len("change") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"change\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("change"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("change")); err != nil {
		return err
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajUnsignedInt, uint64(t.Change)); err != nil {
		return err
	}
	return nil
}

func (t *ClaimsChange) UnmarshalCBOR(r io.Reader) (err error) {
	*t = ClaimsChange{}

	cr := cbg.NewCborReader(r)

	maj, extra, err := cr.ReadHeader()
	if err != nil {
		return err
	}
	defer func() {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
	}()

	if maj != cbg.MajMap {
		return fmt.Errorf("cbor input should be of type map")
	}

	if extra > cbg.MaxLength {
		return fmt.Errorf("ClaimsChange: map struct too large (%d)", extra)
	}

	var name string
	n := extra

	for i := uint64(0); i < n; i++ {

		{
			sval, err := cbg.ReadString(cr)
			if err != nil {
				return err
			}

			name = string(sval)
		}

		switch name {
		// t.Miner ([]uint8) (slice)
		case "miner":

			maj, extra, err = cr.ReadHeader()
			if err != nil {
				return err
			}

			if extra > cbg.ByteArrayMaxLen {
				return fmt.Errorf("t.Miner: byte array too large (%d)", extra)
			}
			if maj != cbg.MajByteString {
				return fmt.Errorf("expected byte array")
			}

			if extra > 0 {
				t.Miner = make([]uint8, extra)
			}

			if _, err := io.ReadFull(cr, t.Miner[:]); err != nil {
				return err
			}
			// t.Current (typegen.Deferred) (struct)
		case "current":

			{

				t.Current = new(cbg.Deferred)

				if err := t.Current.UnmarshalCBOR(cr); err != nil {
					return xerrors.Errorf("failed to read deferred field: %w", err)
				}
			}
			// t.Previous (typegen.Deferred) (struct)
		case "previous":

			{

				t.Previous = new(cbg.Deferred)

				if err := t.Previous.UnmarshalCBOR(cr); err != nil {
					return xerrors.Errorf("failed to read deferred field: %w", err)
				}
			}
			// t.Change (core.ChangeType) (uint8)
		case "change":

			maj, extra, err = cr.ReadHeader()
			if err != nil {
				return err
			}
			if maj != cbg.MajUnsignedInt {
				return fmt.Errorf("wrong type for uint8 field")
			}
			if extra > math.MaxUint8 {
				return fmt.Errorf("integer in input was too large for uint8 field")
			}
			t.Change = core.ChangeType(extra)

		default:
			// Field doesn't exist on this type, so ignore it
			cbg.ScanForLinks(r, func(cid.Cid) {})
		}
	}

	return nil
}

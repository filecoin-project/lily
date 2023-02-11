// Code generated by github.com/whyrusleeping/cbor-gen. DO NOT EDIT.

package v1

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

func (t *SectorStatusChange) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)

	if _, err := cw.Write([]byte{164}); err != nil {
		return err
	}

	// t.Removed (bitfield.BitField) (struct)
	if len("removed") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"removed\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("removed"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("removed")); err != nil {
		return err
	}

	if err := t.Removed.MarshalCBOR(cw); err != nil {
		return err
	}

	// t.Recovering (bitfield.BitField) (struct)
	if len("recovering") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"recovering\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("recovering"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("recovering")); err != nil {
		return err
	}

	if err := t.Recovering.MarshalCBOR(cw); err != nil {
		return err
	}

	// t.Faulted (bitfield.BitField) (struct)
	if len("faulted") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"faulted\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("faulted"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("faulted")); err != nil {
		return err
	}

	if err := t.Faulted.MarshalCBOR(cw); err != nil {
		return err
	}

	// t.Recovered (bitfield.BitField) (struct)
	if len("recovered") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"recovered\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("recovered"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("recovered")); err != nil {
		return err
	}

	if err := t.Recovered.MarshalCBOR(cw); err != nil {
		return err
	}
	return nil
}

func (t *SectorStatusChange) UnmarshalCBOR(r io.Reader) (err error) {
	*t = SectorStatusChange{}

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
		return fmt.Errorf("SectorStatusChange: map struct too large (%d)", extra)
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
		// t.Removed (bitfield.BitField) (struct)
		case "removed":

			{

				if err := t.Removed.UnmarshalCBOR(cr); err != nil {
					return xerrors.Errorf("unmarshaling t.Removed: %w", err)
				}

			}
			// t.Recovering (bitfield.BitField) (struct)
		case "recovering":

			{

				if err := t.Recovering.UnmarshalCBOR(cr); err != nil {
					return xerrors.Errorf("unmarshaling t.Recovering: %w", err)
				}

			}
			// t.Faulted (bitfield.BitField) (struct)
		case "faulted":

			{

				if err := t.Faulted.UnmarshalCBOR(cr); err != nil {
					return xerrors.Errorf("unmarshaling t.Faulted: %w", err)
				}

			}
			// t.Recovered (bitfield.BitField) (struct)
		case "recovered":

			{

				if err := t.Recovered.UnmarshalCBOR(cr); err != nil {
					return xerrors.Errorf("unmarshaling t.Recovered: %w", err)
				}

			}

		default:
			// Field doesn't exist on this type, so ignore it
			cbg.ScanForLinks(r, func(cid.Cid) {})
		}
	}

	return nil
}
func (t *PreCommitChange) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)

	if _, err := cw.Write([]byte{164}); err != nil {
		return err
	}

	// t.SectorNumber ([]uint8) (slice)
	if len("sector_number") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"sector_number\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("sector_number"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("sector_number")); err != nil {
		return err
	}

	if len(t.SectorNumber) > cbg.ByteArrayMaxLen {
		return xerrors.Errorf("Byte array in field t.SectorNumber was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajByteString, uint64(len(t.SectorNumber))); err != nil {
		return err
	}

	if _, err := cw.Write(t.SectorNumber[:]); err != nil {
		return err
	}

	// t.Current (typegen.Deferred) (struct)
	if len("current_pre_commit") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"current_pre_commit\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("current_pre_commit"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("current_pre_commit")); err != nil {
		return err
	}

	if err := t.Current.MarshalCBOR(cw); err != nil {
		return err
	}

	// t.Previous (typegen.Deferred) (struct)
	if len("previous_pre_commit") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"previous_pre_commit\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("previous_pre_commit"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("previous_pre_commit")); err != nil {
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

func (t *PreCommitChange) UnmarshalCBOR(r io.Reader) (err error) {
	*t = PreCommitChange{}

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
		return fmt.Errorf("PreCommitChange: map struct too large (%d)", extra)
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
		// t.SectorNumber ([]uint8) (slice)
		case "sector_number":

			maj, extra, err = cr.ReadHeader()
			if err != nil {
				return err
			}

			if extra > cbg.ByteArrayMaxLen {
				return fmt.Errorf("t.SectorNumber: byte array too large (%d)", extra)
			}
			if maj != cbg.MajByteString {
				return fmt.Errorf("expected byte array")
			}

			if extra > 0 {
				t.SectorNumber = make([]uint8, extra)
			}

			if _, err := io.ReadFull(cr, t.SectorNumber[:]); err != nil {
				return err
			}
			// t.Current (typegen.Deferred) (struct)
		case "current_pre_commit":

			{

				t.Current = new(cbg.Deferred)

				if err := t.Current.UnmarshalCBOR(cr); err != nil {
					return xerrors.Errorf("failed to read deferred field: %w", err)
				}
			}
			// t.Previous (typegen.Deferred) (struct)
		case "previous_pre_commit":

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
func (t *SectorChange) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)

	if _, err := cw.Write([]byte{164}); err != nil {
		return err
	}

	// t.SectorNumber (uint64) (uint64)
	if len("sector_number") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"sector_number\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("sector_number"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("sector_number")); err != nil {
		return err
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajUnsignedInt, uint64(t.SectorNumber)); err != nil {
		return err
	}

	// t.Current (typegen.Deferred) (struct)
	if len("current_sector") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"current_sector\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("current_sector"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("current_sector")); err != nil {
		return err
	}

	if err := t.Current.MarshalCBOR(cw); err != nil {
		return err
	}

	// t.Previous (typegen.Deferred) (struct)
	if len("previous_sector") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"previous_sector\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("previous_sector"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("previous_sector")); err != nil {
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

func (t *SectorChange) UnmarshalCBOR(r io.Reader) (err error) {
	*t = SectorChange{}

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
		return fmt.Errorf("SectorChange: map struct too large (%d)", extra)
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
		// t.SectorNumber (uint64) (uint64)
		case "sector_number":

			{

				maj, extra, err = cr.ReadHeader()
				if err != nil {
					return err
				}
				if maj != cbg.MajUnsignedInt {
					return fmt.Errorf("wrong type for uint64 field")
				}
				t.SectorNumber = uint64(extra)

			}
			// t.Current (typegen.Deferred) (struct)
		case "current_sector":

			{

				t.Current = new(cbg.Deferred)

				if err := t.Current.UnmarshalCBOR(cr); err != nil {
					return xerrors.Errorf("failed to read deferred field: %w", err)
				}
			}
			// t.Previous (typegen.Deferred) (struct)
		case "previous_sector":

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
func (t *InfoChange) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)

	if _, err := cw.Write([]byte{162}); err != nil {
		return err
	}

	// t.Info (typegen.Deferred) (struct)
	if len("info") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"info\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("info"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("info")); err != nil {
		return err
	}

	if err := t.Info.MarshalCBOR(cw); err != nil {
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

func (t *InfoChange) UnmarshalCBOR(r io.Reader) (err error) {
	*t = InfoChange{}

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
		return fmt.Errorf("InfoChange: map struct too large (%d)", extra)
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
		// t.Info (typegen.Deferred) (struct)
		case "info":

			{

				t.Info = new(cbg.Deferred)

				if err := t.Info.UnmarshalCBOR(cr); err != nil {
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
func (t *StateChange) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)

	if _, err := cw.Write([]byte{164}); err != nil {
		return err
	}

	// t.SectorStatus (cid.Cid) (struct)
	if len("sector_status") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"sector_status\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("sector_status"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("sector_status")); err != nil {
		return err
	}

	if t.SectorStatus == nil {
		if _, err := cw.Write(cbg.CborNull); err != nil {
			return err
		}
	} else {
		if err := cbg.WriteCid(cw, *t.SectorStatus); err != nil {
			return xerrors.Errorf("failed to write cid field t.SectorStatus: %w", err)
		}
	}

	// t.Info (cid.Cid) (struct)
	if len("info") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"info\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("info"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("info")); err != nil {
		return err
	}

	if t.Info == nil {
		if _, err := cw.Write(cbg.CborNull); err != nil {
			return err
		}
	} else {
		if err := cbg.WriteCid(cw, *t.Info); err != nil {
			return xerrors.Errorf("failed to write cid field t.Info: %w", err)
		}
	}

	// t.PreCommits (cid.Cid) (struct)
	if len("pre_commits") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"pre_commits\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("pre_commits"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("pre_commits")); err != nil {
		return err
	}

	if t.PreCommits == nil {
		if _, err := cw.Write(cbg.CborNull); err != nil {
			return err
		}
	} else {
		if err := cbg.WriteCid(cw, *t.PreCommits); err != nil {
			return xerrors.Errorf("failed to write cid field t.PreCommits: %w", err)
		}
	}

	// t.Sectors (cid.Cid) (struct)
	if len("sectors") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"sectors\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("sectors"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("sectors")); err != nil {
		return err
	}

	if t.Sectors == nil {
		if _, err := cw.Write(cbg.CborNull); err != nil {
			return err
		}
	} else {
		if err := cbg.WriteCid(cw, *t.Sectors); err != nil {
			return xerrors.Errorf("failed to write cid field t.Sectors: %w", err)
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
		// t.SectorStatus (cid.Cid) (struct)
		case "sector_status":

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
						return xerrors.Errorf("failed to read cid field t.SectorStatus: %w", err)
					}

					t.SectorStatus = &c
				}

			}
			// t.Info (cid.Cid) (struct)
		case "info":

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
						return xerrors.Errorf("failed to read cid field t.Info: %w", err)
					}

					t.Info = &c
				}

			}
			// t.PreCommits (cid.Cid) (struct)
		case "pre_commits":

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
						return xerrors.Errorf("failed to read cid field t.PreCommits: %w", err)
					}

					t.PreCommits = &c
				}

			}
			// t.Sectors (cid.Cid) (struct)
		case "sectors":

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
						return xerrors.Errorf("failed to read cid field t.Sectors: %w", err)
					}

					t.Sectors = &c
				}

			}

		default:
			// Field doesn't exist on this type, so ignore it
			cbg.ScanForLinks(r, func(cid.Cid) {})
		}
	}

	return nil
}

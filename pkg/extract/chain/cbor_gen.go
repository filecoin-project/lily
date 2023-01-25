// Code generated by github.com/whyrusleeping/cbor-gen. DO NOT EDIT.

package chain

import (
	"fmt"
	"io"
	"math"
	"sort"

	exitcode "github.com/filecoin-project/go-state-types/exitcode"
	types "github.com/filecoin-project/lotus/chain/types"
	cid "github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	xerrors "golang.org/x/xerrors"
)

var _ = xerrors.Errorf
var _ = cid.Undef
var _ = math.E
var _ = sort.Sort

func (t *ChainMessageReceipt) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)

	if _, err := cw.Write([]byte{164}); err != nil {
		return err
	}

	// t.Receipt (types.MessageReceipt) (struct)
	if len("receipt") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"receipt\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("receipt"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("receipt")); err != nil {
		return err
	}

	if err := t.Receipt.MarshalCBOR(cw); err != nil {
		return err
	}

	// t.GasOutputs (chain.MessageGasOutputs) (struct)
	if len("gas") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"gas\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("gas"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("gas")); err != nil {
		return err
	}

	if err := t.GasOutputs.MarshalCBOR(cw); err != nil {
		return err
	}

	// t.ActorError (chain.ActorError) (struct)
	if len("errors") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"errors\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("errors"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("errors")); err != nil {
		return err
	}

	if err := t.ActorError.MarshalCBOR(cw); err != nil {
		return err
	}

	// t.Index (int64) (int64)
	if len("index") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"index\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("index"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("index")); err != nil {
		return err
	}

	if t.Index >= 0 {
		if err := cw.WriteMajorTypeHeader(cbg.MajUnsignedInt, uint64(t.Index)); err != nil {
			return err
		}
	} else {
		if err := cw.WriteMajorTypeHeader(cbg.MajNegativeInt, uint64(-t.Index-1)); err != nil {
			return err
		}
	}
	return nil
}

func (t *ChainMessageReceipt) UnmarshalCBOR(r io.Reader) (err error) {
	*t = ChainMessageReceipt{}

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
		return fmt.Errorf("ChainMessageReceipt: map struct too large (%d)", extra)
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
		// t.Receipt (types.MessageReceipt) (struct)
		case "receipt":

			{

				if err := t.Receipt.UnmarshalCBOR(cr); err != nil {
					return xerrors.Errorf("unmarshaling t.Receipt: %w", err)
				}

			}
			// t.GasOutputs (chain.MessageGasOutputs) (struct)
		case "gas":

			{

				b, err := cr.ReadByte()
				if err != nil {
					return err
				}
				if b != cbg.CborNull[0] {
					if err := cr.UnreadByte(); err != nil {
						return err
					}
					t.GasOutputs = new(MessageGasOutputs)
					if err := t.GasOutputs.UnmarshalCBOR(cr); err != nil {
						return xerrors.Errorf("unmarshaling t.GasOutputs pointer: %w", err)
					}
				}

			}
			// t.ActorError (chain.ActorError) (struct)
		case "errors":

			{

				b, err := cr.ReadByte()
				if err != nil {
					return err
				}
				if b != cbg.CborNull[0] {
					if err := cr.UnreadByte(); err != nil {
						return err
					}
					t.ActorError = new(ActorError)
					if err := t.ActorError.UnmarshalCBOR(cr); err != nil {
						return xerrors.Errorf("unmarshaling t.ActorError pointer: %w", err)
					}
				}

			}
			// t.Index (int64) (int64)
		case "index":
			{
				maj, extra, err := cr.ReadHeader()
				var extraI int64
				if err != nil {
					return err
				}
				switch maj {
				case cbg.MajUnsignedInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 positive overflow")
					}
				case cbg.MajNegativeInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 negative oveflow")
					}
					extraI = -1 - extraI
				default:
					return fmt.Errorf("wrong type for int64 field: %d", maj)
				}

				t.Index = int64(extraI)
			}

		default:
			// Field doesn't exist on this type, so ignore it
			cbg.ScanForLinks(r, func(cid.Cid) {})
		}
	}

	return nil
}
func (t *ImplicitMessageReceipt) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)

	if _, err := cw.Write([]byte{163}); err != nil {
		return err
	}

	// t.Receipt (types.MessageReceipt) (struct)
	if len("receipt") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"receipt\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("receipt"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("receipt")); err != nil {
		return err
	}

	if err := t.Receipt.MarshalCBOR(cw); err != nil {
		return err
	}

	// t.GasOutputs (chain.MessageGasOutputs) (struct)
	if len("gas") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"gas\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("gas"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("gas")); err != nil {
		return err
	}

	if err := t.GasOutputs.MarshalCBOR(cw); err != nil {
		return err
	}

	// t.ActorError (chain.ActorError) (struct)
	if len("errors") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"errors\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("errors"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("errors")); err != nil {
		return err
	}

	if err := t.ActorError.MarshalCBOR(cw); err != nil {
		return err
	}
	return nil
}

func (t *ImplicitMessageReceipt) UnmarshalCBOR(r io.Reader) (err error) {
	*t = ImplicitMessageReceipt{}

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
		return fmt.Errorf("ImplicitMessageReceipt: map struct too large (%d)", extra)
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
		// t.Receipt (types.MessageReceipt) (struct)
		case "receipt":

			{

				if err := t.Receipt.UnmarshalCBOR(cr); err != nil {
					return xerrors.Errorf("unmarshaling t.Receipt: %w", err)
				}

			}
			// t.GasOutputs (chain.MessageGasOutputs) (struct)
		case "gas":

			{

				b, err := cr.ReadByte()
				if err != nil {
					return err
				}
				if b != cbg.CborNull[0] {
					if err := cr.UnreadByte(); err != nil {
						return err
					}
					t.GasOutputs = new(MessageGasOutputs)
					if err := t.GasOutputs.UnmarshalCBOR(cr); err != nil {
						return xerrors.Errorf("unmarshaling t.GasOutputs pointer: %w", err)
					}
				}

			}
			// t.ActorError (chain.ActorError) (struct)
		case "errors":

			{

				b, err := cr.ReadByte()
				if err != nil {
					return err
				}
				if b != cbg.CborNull[0] {
					if err := cr.UnreadByte(); err != nil {
						return err
					}
					t.ActorError = new(ActorError)
					if err := t.ActorError.UnmarshalCBOR(cr); err != nil {
						return xerrors.Errorf("unmarshaling t.ActorError pointer: %w", err)
					}
				}

			}

		default:
			// Field doesn't exist on this type, so ignore it
			cbg.ScanForLinks(r, func(cid.Cid) {})
		}
	}

	return nil
}
func (t *MessageGasOutputs) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)

	if _, err := cw.Write([]byte{167}); err != nil {
		return err
	}

	// t.BaseFeeBurn (big.Int) (struct)
	if len("basefeeburn") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"basefeeburn\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("basefeeburn"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("basefeeburn")); err != nil {
		return err
	}

	if err := t.BaseFeeBurn.MarshalCBOR(cw); err != nil {
		return err
	}

	// t.OverEstimationBurn (big.Int) (struct)
	if len("overestimationburn") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"overestimationburn\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("overestimationburn"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("overestimationburn")); err != nil {
		return err
	}

	if err := t.OverEstimationBurn.MarshalCBOR(cw); err != nil {
		return err
	}

	// t.MinerPenalty (big.Int) (struct)
	if len("minerpenalty") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"minerpenalty\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("minerpenalty"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("minerpenalty")); err != nil {
		return err
	}

	if err := t.MinerPenalty.MarshalCBOR(cw); err != nil {
		return err
	}

	// t.MinerTip (big.Int) (struct)
	if len("minertip") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"minertip\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("minertip"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("minertip")); err != nil {
		return err
	}

	if err := t.MinerTip.MarshalCBOR(cw); err != nil {
		return err
	}

	// t.Refund (big.Int) (struct)
	if len("refund") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"refund\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("refund"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("refund")); err != nil {
		return err
	}

	if err := t.Refund.MarshalCBOR(cw); err != nil {
		return err
	}

	// t.GasRefund (int64) (int64)
	if len("gasrufund") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"gasrufund\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("gasrufund"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("gasrufund")); err != nil {
		return err
	}

	if t.GasRefund >= 0 {
		if err := cw.WriteMajorTypeHeader(cbg.MajUnsignedInt, uint64(t.GasRefund)); err != nil {
			return err
		}
	} else {
		if err := cw.WriteMajorTypeHeader(cbg.MajNegativeInt, uint64(-t.GasRefund-1)); err != nil {
			return err
		}
	}

	// t.GasBurned (int64) (int64)
	if len("gasburned") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"gasburned\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("gasburned"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("gasburned")); err != nil {
		return err
	}

	if t.GasBurned >= 0 {
		if err := cw.WriteMajorTypeHeader(cbg.MajUnsignedInt, uint64(t.GasBurned)); err != nil {
			return err
		}
	} else {
		if err := cw.WriteMajorTypeHeader(cbg.MajNegativeInt, uint64(-t.GasBurned-1)); err != nil {
			return err
		}
	}
	return nil
}

func (t *MessageGasOutputs) UnmarshalCBOR(r io.Reader) (err error) {
	*t = MessageGasOutputs{}

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
		return fmt.Errorf("MessageGasOutputs: map struct too large (%d)", extra)
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
		// t.BaseFeeBurn (big.Int) (struct)
		case "basefeeburn":

			{

				if err := t.BaseFeeBurn.UnmarshalCBOR(cr); err != nil {
					return xerrors.Errorf("unmarshaling t.BaseFeeBurn: %w", err)
				}

			}
			// t.OverEstimationBurn (big.Int) (struct)
		case "overestimationburn":

			{

				if err := t.OverEstimationBurn.UnmarshalCBOR(cr); err != nil {
					return xerrors.Errorf("unmarshaling t.OverEstimationBurn: %w", err)
				}

			}
			// t.MinerPenalty (big.Int) (struct)
		case "minerpenalty":

			{

				if err := t.MinerPenalty.UnmarshalCBOR(cr); err != nil {
					return xerrors.Errorf("unmarshaling t.MinerPenalty: %w", err)
				}

			}
			// t.MinerTip (big.Int) (struct)
		case "minertip":

			{

				if err := t.MinerTip.UnmarshalCBOR(cr); err != nil {
					return xerrors.Errorf("unmarshaling t.MinerTip: %w", err)
				}

			}
			// t.Refund (big.Int) (struct)
		case "refund":

			{

				if err := t.Refund.UnmarshalCBOR(cr); err != nil {
					return xerrors.Errorf("unmarshaling t.Refund: %w", err)
				}

			}
			// t.GasRefund (int64) (int64)
		case "gasrufund":
			{
				maj, extra, err := cr.ReadHeader()
				var extraI int64
				if err != nil {
					return err
				}
				switch maj {
				case cbg.MajUnsignedInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 positive overflow")
					}
				case cbg.MajNegativeInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 negative oveflow")
					}
					extraI = -1 - extraI
				default:
					return fmt.Errorf("wrong type for int64 field: %d", maj)
				}

				t.GasRefund = int64(extraI)
			}
			// t.GasBurned (int64) (int64)
		case "gasburned":
			{
				maj, extra, err := cr.ReadHeader()
				var extraI int64
				if err != nil {
					return err
				}
				switch maj {
				case cbg.MajUnsignedInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 positive overflow")
					}
				case cbg.MajNegativeInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 negative oveflow")
					}
					extraI = -1 - extraI
				default:
					return fmt.Errorf("wrong type for int64 field: %d", maj)
				}

				t.GasBurned = int64(extraI)
			}

		default:
			// Field doesn't exist on this type, so ignore it
			cbg.ScanForLinks(r, func(cid.Cid) {})
		}
	}

	return nil
}
func (t *ActorError) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)

	if _, err := cw.Write([]byte{163}); err != nil {
		return err
	}

	// t.Fatal (bool) (bool)
	if len("fatal") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"fatal\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("fatal"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("fatal")); err != nil {
		return err
	}

	if err := cbg.WriteBool(w, t.Fatal); err != nil {
		return err
	}

	// t.RetCode (exitcode.ExitCode) (int64)
	if len("retcode") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"retcode\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("retcode"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("retcode")); err != nil {
		return err
	}

	if t.RetCode >= 0 {
		if err := cw.WriteMajorTypeHeader(cbg.MajUnsignedInt, uint64(t.RetCode)); err != nil {
			return err
		}
	} else {
		if err := cw.WriteMajorTypeHeader(cbg.MajNegativeInt, uint64(-t.RetCode-1)); err != nil {
			return err
		}
	}

	// t.Error (string) (string)
	if len("error") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"error\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("error"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("error")); err != nil {
		return err
	}

	if len(t.Error) > cbg.MaxLength {
		return xerrors.Errorf("Value in field t.Error was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.Error))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string(t.Error)); err != nil {
		return err
	}
	return nil
}

func (t *ActorError) UnmarshalCBOR(r io.Reader) (err error) {
	*t = ActorError{}

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
		return fmt.Errorf("ActorError: map struct too large (%d)", extra)
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
		// t.Fatal (bool) (bool)
		case "fatal":

			maj, extra, err = cr.ReadHeader()
			if err != nil {
				return err
			}
			if maj != cbg.MajOther {
				return fmt.Errorf("booleans must be major type 7")
			}
			switch extra {
			case 20:
				t.Fatal = false
			case 21:
				t.Fatal = true
			default:
				return fmt.Errorf("booleans are either major type 7, value 20 or 21 (got %d)", extra)
			}
			// t.RetCode (exitcode.ExitCode) (int64)
		case "retcode":
			{
				maj, extra, err := cr.ReadHeader()
				var extraI int64
				if err != nil {
					return err
				}
				switch maj {
				case cbg.MajUnsignedInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 positive overflow")
					}
				case cbg.MajNegativeInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 negative oveflow")
					}
					extraI = -1 - extraI
				default:
					return fmt.Errorf("wrong type for int64 field: %d", maj)
				}

				t.RetCode = exitcode.ExitCode(extraI)
			}
			// t.Error (string) (string)
		case "error":

			{
				sval, err := cbg.ReadString(cr)
				if err != nil {
					return err
				}

				t.Error = string(sval)
			}

		default:
			// Field doesn't exist on this type, so ignore it
			cbg.ScanForLinks(r, func(cid.Cid) {})
		}
	}

	return nil
}
func (t *VmMessage) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)

	if _, err := cw.Write([]byte{165}); err != nil {
		return err
	}

	// t.Source (cid.Cid) (struct)
	if len("source") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"source\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("source"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("source")); err != nil {
		return err
	}

	if err := cbg.WriteCid(cw, t.Source); err != nil {
		return xerrors.Errorf("failed to write cid field t.Source: %w", err)
	}

	// t.Message (types.Message) (struct)
	if len("message") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"message\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("message"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("message")); err != nil {
		return err
	}

	if err := t.Message.MarshalCBOR(cw); err != nil {
		return err
	}

	// t.Receipt (types.MessageReceipt) (struct)
	if len("receipt") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"receipt\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("receipt"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("receipt")); err != nil {
		return err
	}

	if err := t.Receipt.MarshalCBOR(cw); err != nil {
		return err
	}

	// t.Error (string) (string)
	if len("error") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"error\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("error"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("error")); err != nil {
		return err
	}

	if len(t.Error) > cbg.MaxLength {
		return xerrors.Errorf("Value in field t.Error was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.Error))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string(t.Error)); err != nil {
		return err
	}

	// t.Index (int64) (int64)
	if len("index") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"index\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("index"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("index")); err != nil {
		return err
	}

	if t.Index >= 0 {
		if err := cw.WriteMajorTypeHeader(cbg.MajUnsignedInt, uint64(t.Index)); err != nil {
			return err
		}
	} else {
		if err := cw.WriteMajorTypeHeader(cbg.MajNegativeInt, uint64(-t.Index-1)); err != nil {
			return err
		}
	}
	return nil
}

func (t *VmMessage) UnmarshalCBOR(r io.Reader) (err error) {
	*t = VmMessage{}

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
		return fmt.Errorf("VmMessage: map struct too large (%d)", extra)
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
		// t.Source (cid.Cid) (struct)
		case "source":

			{

				c, err := cbg.ReadCid(cr)
				if err != nil {
					return xerrors.Errorf("failed to read cid field t.Source: %w", err)
				}

				t.Source = c

			}
			// t.Message (types.Message) (struct)
		case "message":

			{

				b, err := cr.ReadByte()
				if err != nil {
					return err
				}
				if b != cbg.CborNull[0] {
					if err := cr.UnreadByte(); err != nil {
						return err
					}
					t.Message = new(types.Message)
					if err := t.Message.UnmarshalCBOR(cr); err != nil {
						return xerrors.Errorf("unmarshaling t.Message pointer: %w", err)
					}
				}

			}
			// t.Receipt (types.MessageReceipt) (struct)
		case "receipt":

			{

				if err := t.Receipt.UnmarshalCBOR(cr); err != nil {
					return xerrors.Errorf("unmarshaling t.Receipt: %w", err)
				}

			}
			// t.Error (string) (string)
		case "error":

			{
				sval, err := cbg.ReadString(cr)
				if err != nil {
					return err
				}

				t.Error = string(sval)
			}
			// t.Index (int64) (int64)
		case "index":
			{
				maj, extra, err := cr.ReadHeader()
				var extraI int64
				if err != nil {
					return err
				}
				switch maj {
				case cbg.MajUnsignedInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 positive overflow")
					}
				case cbg.MajNegativeInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 negative oveflow")
					}
					extraI = -1 - extraI
				default:
					return fmt.Errorf("wrong type for int64 field: %d", maj)
				}

				t.Index = int64(extraI)
			}

		default:
			// Field doesn't exist on this type, so ignore it
			cbg.ScanForLinks(r, func(cid.Cid) {})
		}
	}

	return nil
}
func (t *VmMessageGasTrace) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)

	if _, err := cw.Write([]byte{168}); err != nil {
		return err
	}

	// t.Name (string) (string)
	if len("name") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"name\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("name"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("name")); err != nil {
		return err
	}

	if len(t.Name) > cbg.MaxLength {
		return xerrors.Errorf("Value in field t.Name was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.Name))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string(t.Name)); err != nil {
		return err
	}

	// t.Location ([]chain.Loc) (slice)
	if len("location") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"location\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("location"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("location")); err != nil {
		return err
	}

	if len(t.Location) > cbg.MaxLength {
		return xerrors.Errorf("Slice value in field t.Location was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajArray, uint64(len(t.Location))); err != nil {
		return err
	}
	for _, v := range t.Location {
		if err := v.MarshalCBOR(cw); err != nil {
			return err
		}
	}

	// t.TotalGas (int64) (int64)
	if len("totalgas") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"totalgas\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("totalgas"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("totalgas")); err != nil {
		return err
	}

	if t.TotalGas >= 0 {
		if err := cw.WriteMajorTypeHeader(cbg.MajUnsignedInt, uint64(t.TotalGas)); err != nil {
			return err
		}
	} else {
		if err := cw.WriteMajorTypeHeader(cbg.MajNegativeInt, uint64(-t.TotalGas-1)); err != nil {
			return err
		}
	}

	// t.ComputeGas (int64) (int64)
	if len("computegas") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"computegas\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("computegas"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("computegas")); err != nil {
		return err
	}

	if t.ComputeGas >= 0 {
		if err := cw.WriteMajorTypeHeader(cbg.MajUnsignedInt, uint64(t.ComputeGas)); err != nil {
			return err
		}
	} else {
		if err := cw.WriteMajorTypeHeader(cbg.MajNegativeInt, uint64(-t.ComputeGas-1)); err != nil {
			return err
		}
	}

	// t.StorageGas (int64) (int64)
	if len("storagegas") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"storagegas\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("storagegas"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("storagegas")); err != nil {
		return err
	}

	if t.StorageGas >= 0 {
		if err := cw.WriteMajorTypeHeader(cbg.MajUnsignedInt, uint64(t.StorageGas)); err != nil {
			return err
		}
	} else {
		if err := cw.WriteMajorTypeHeader(cbg.MajNegativeInt, uint64(-t.StorageGas-1)); err != nil {
			return err
		}
	}

	// t.TotalVirtualGas (int64) (int64)
	if len("totalvirtgas") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"totalvirtgas\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("totalvirtgas"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("totalvirtgas")); err != nil {
		return err
	}

	if t.TotalVirtualGas >= 0 {
		if err := cw.WriteMajorTypeHeader(cbg.MajUnsignedInt, uint64(t.TotalVirtualGas)); err != nil {
			return err
		}
	} else {
		if err := cw.WriteMajorTypeHeader(cbg.MajNegativeInt, uint64(-t.TotalVirtualGas-1)); err != nil {
			return err
		}
	}

	// t.VirtualComputeGas (int64) (int64)
	if len("virtcomputegas") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"virtcomputegas\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("virtcomputegas"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("virtcomputegas")); err != nil {
		return err
	}

	if t.VirtualComputeGas >= 0 {
		if err := cw.WriteMajorTypeHeader(cbg.MajUnsignedInt, uint64(t.VirtualComputeGas)); err != nil {
			return err
		}
	} else {
		if err := cw.WriteMajorTypeHeader(cbg.MajNegativeInt, uint64(-t.VirtualComputeGas-1)); err != nil {
			return err
		}
	}

	// t.VirtualStorageGas (int64) (int64)
	if len("cirtstoragegas") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"cirtstoragegas\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("cirtstoragegas"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("cirtstoragegas")); err != nil {
		return err
	}

	if t.VirtualStorageGas >= 0 {
		if err := cw.WriteMajorTypeHeader(cbg.MajUnsignedInt, uint64(t.VirtualStorageGas)); err != nil {
			return err
		}
	} else {
		if err := cw.WriteMajorTypeHeader(cbg.MajNegativeInt, uint64(-t.VirtualStorageGas-1)); err != nil {
			return err
		}
	}
	return nil
}

func (t *VmMessageGasTrace) UnmarshalCBOR(r io.Reader) (err error) {
	*t = VmMessageGasTrace{}

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
		return fmt.Errorf("VmMessageGasTrace: map struct too large (%d)", extra)
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
		// t.Name (string) (string)
		case "name":

			{
				sval, err := cbg.ReadString(cr)
				if err != nil {
					return err
				}

				t.Name = string(sval)
			}
			// t.Location ([]chain.Loc) (slice)
		case "location":

			maj, extra, err = cr.ReadHeader()
			if err != nil {
				return err
			}

			if extra > cbg.MaxLength {
				return fmt.Errorf("t.Location: array too large (%d)", extra)
			}

			if maj != cbg.MajArray {
				return fmt.Errorf("expected cbor array")
			}

			if extra > 0 {
				t.Location = make([]Loc, extra)
			}

			for i := 0; i < int(extra); i++ {

				var v Loc
				if err := v.UnmarshalCBOR(cr); err != nil {
					return err
				}

				t.Location[i] = v
			}

			// t.TotalGas (int64) (int64)
		case "totalgas":
			{
				maj, extra, err := cr.ReadHeader()
				var extraI int64
				if err != nil {
					return err
				}
				switch maj {
				case cbg.MajUnsignedInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 positive overflow")
					}
				case cbg.MajNegativeInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 negative oveflow")
					}
					extraI = -1 - extraI
				default:
					return fmt.Errorf("wrong type for int64 field: %d", maj)
				}

				t.TotalGas = int64(extraI)
			}
			// t.ComputeGas (int64) (int64)
		case "computegas":
			{
				maj, extra, err := cr.ReadHeader()
				var extraI int64
				if err != nil {
					return err
				}
				switch maj {
				case cbg.MajUnsignedInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 positive overflow")
					}
				case cbg.MajNegativeInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 negative oveflow")
					}
					extraI = -1 - extraI
				default:
					return fmt.Errorf("wrong type for int64 field: %d", maj)
				}

				t.ComputeGas = int64(extraI)
			}
			// t.StorageGas (int64) (int64)
		case "storagegas":
			{
				maj, extra, err := cr.ReadHeader()
				var extraI int64
				if err != nil {
					return err
				}
				switch maj {
				case cbg.MajUnsignedInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 positive overflow")
					}
				case cbg.MajNegativeInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 negative oveflow")
					}
					extraI = -1 - extraI
				default:
					return fmt.Errorf("wrong type for int64 field: %d", maj)
				}

				t.StorageGas = int64(extraI)
			}
			// t.TotalVirtualGas (int64) (int64)
		case "totalvirtgas":
			{
				maj, extra, err := cr.ReadHeader()
				var extraI int64
				if err != nil {
					return err
				}
				switch maj {
				case cbg.MajUnsignedInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 positive overflow")
					}
				case cbg.MajNegativeInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 negative oveflow")
					}
					extraI = -1 - extraI
				default:
					return fmt.Errorf("wrong type for int64 field: %d", maj)
				}

				t.TotalVirtualGas = int64(extraI)
			}
			// t.VirtualComputeGas (int64) (int64)
		case "virtcomputegas":
			{
				maj, extra, err := cr.ReadHeader()
				var extraI int64
				if err != nil {
					return err
				}
				switch maj {
				case cbg.MajUnsignedInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 positive overflow")
					}
				case cbg.MajNegativeInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 negative oveflow")
					}
					extraI = -1 - extraI
				default:
					return fmt.Errorf("wrong type for int64 field: %d", maj)
				}

				t.VirtualComputeGas = int64(extraI)
			}
			// t.VirtualStorageGas (int64) (int64)
		case "cirtstoragegas":
			{
				maj, extra, err := cr.ReadHeader()
				var extraI int64
				if err != nil {
					return err
				}
				switch maj {
				case cbg.MajUnsignedInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 positive overflow")
					}
				case cbg.MajNegativeInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 negative oveflow")
					}
					extraI = -1 - extraI
				default:
					return fmt.Errorf("wrong type for int64 field: %d", maj)
				}

				t.VirtualStorageGas = int64(extraI)
			}

		default:
			// Field doesn't exist on this type, so ignore it
			cbg.ScanForLinks(r, func(cid.Cid) {})
		}
	}

	return nil
}
func (t *Loc) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}

	cw := cbg.NewCborWriter(w)

	if _, err := cw.Write([]byte{163}); err != nil {
		return err
	}

	// t.File (string) (string)
	if len("file") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"file\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("file"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("file")); err != nil {
		return err
	}

	if len(t.File) > cbg.MaxLength {
		return xerrors.Errorf("Value in field t.File was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.File))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string(t.File)); err != nil {
		return err
	}

	// t.Line (int64) (int64)
	if len("line") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"line\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("line"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("line")); err != nil {
		return err
	}

	if t.Line >= 0 {
		if err := cw.WriteMajorTypeHeader(cbg.MajUnsignedInt, uint64(t.Line)); err != nil {
			return err
		}
	} else {
		if err := cw.WriteMajorTypeHeader(cbg.MajNegativeInt, uint64(-t.Line-1)); err != nil {
			return err
		}
	}

	// t.Function (string) (string)
	if len("function") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"function\" was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len("function"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("function")); err != nil {
		return err
	}

	if len(t.Function) > cbg.MaxLength {
		return xerrors.Errorf("Value in field t.Function was too long")
	}

	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.Function))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string(t.Function)); err != nil {
		return err
	}
	return nil
}

func (t *Loc) UnmarshalCBOR(r io.Reader) (err error) {
	*t = Loc{}

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
		return fmt.Errorf("Loc: map struct too large (%d)", extra)
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
		// t.File (string) (string)
		case "file":

			{
				sval, err := cbg.ReadString(cr)
				if err != nil {
					return err
				}

				t.File = string(sval)
			}
			// t.Line (int64) (int64)
		case "line":
			{
				maj, extra, err := cr.ReadHeader()
				var extraI int64
				if err != nil {
					return err
				}
				switch maj {
				case cbg.MajUnsignedInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 positive overflow")
					}
				case cbg.MajNegativeInt:
					extraI = int64(extra)
					if extraI < 0 {
						return fmt.Errorf("int64 negative oveflow")
					}
					extraI = -1 - extraI
				default:
					return fmt.Errorf("wrong type for int64 field: %d", maj)
				}

				t.Line = int64(extraI)
			}
			// t.Function (string) (string)
		case "function":

			{
				sval, err := cbg.ReadString(cr)
				if err != nil {
					return err
				}

				t.Function = string(sval)
			}

		default:
			// Field doesn't exist on this type, so ignore it
			cbg.ScanForLinks(r, func(cid.Cid) {})
		}
	}

	return nil
}

package processor

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lily/lens"
	"github.com/filecoin-project/lily/tasks"
)

type RootState struct {
	// StateVersion contains the version of the State data structure.
	// If changes are made to its structure, this version will increment.
	StateVersion uint64

	// State contains a single StateExtraction.
	State cid.Cid
}

type StateExtraction struct {
	// Current and Parent are the TipSets whose state is diffed to produce Actors.
	// BaseFee is calculated from Parent.
	// FullBlocks contains the blocks and messages from Parent and receipts from Current.
	// ImplicitMessages contains implicit messages applied in Parent. These messages and their respective receipts to no appear on chain as the name suggests.
	// Actors contains actors whos state changed between Parent and Current.

	// Current contains the tipset whose state root is the result of Parent's execution.
	Current *types.TipSet
	// Parent contains the parent tipset of Current. Execution of Parent's state produces Current's state root.
	Parent *types.TipSet
	// BaseFee contains the basefee during Parent's execution.
	BaseFee abi.TokenAmount
	// FullBlocks contains a map of a BlockHeader CID to a FullBlock. Together these BlockHeaders form the Parent TipSet.
	FullBlocks map[cid.Cid]*FullBlock
	// ImplicitMessages contains a list of all implicit messages executed at Parent.
	ImplicitMessages []*ImplicitMessage
	// Actors contains the actors whose state changed while executing Parent. Their current state is contained in Current's state root.
	Actors ActorStateChanges
}

type StateExtractionIPLD struct {
	// Current and Parent are the TipSets whose state is diffed to produce Actors.
	// BaseFee is calculated from Parent.
	// FullBlocks contains the blocks and messages from Parent and the messages respective receipts from Current.
	// ImplicitMessages contains implicit messages applied in Parent. These messages and their respective receipts to no appear on chain as the name suggests.
	// Actors contains actors whos state changed between Parent and Current.

	// Current contains the tipset whose state root is the result of Parent's execution.
	Current *types.TipSet
	// Parent contains the parent tipset of Current. Execution of Parent's state produces Current's state root.
	Parent *types.TipSet

	// BaseFee contains the basefee during Parent's execution.
	BaseFee abi.TokenAmount

	// FullBlocks contains a map of a BlockHeader CID to a FullBlock. Together these BlockHeaders form the Parent TipSet.
	FullBlocks cid.Cid // HAMT[BlockHeaderCID]FullBlock

	// ImplicitMessages contains a map of all implicit messages executed at Parent.
	ImplicitMessages cid.Cid // HAMT[MessageCID]ImplicitMessage

	// Actors contains the actors whose state changed while executing Parent. Their current state is contained in Current's state root.
	Actors cid.Cid // ActorStateChanges
}

type MessageStateChanges struct {
	Current          *types.TipSet
	Executed         *types.TipSet
	BaseFee          abi.TokenAmount
	FullBlocks       map[cid.Cid]*FullBlock
	ImplicitMessages []*ImplicitMessage
}

type FullBlock struct {
	Block        *types.BlockHeader
	SecpMessages []*SignedChainMessage
	BlsMessages  []*ChainMessage
}

// SignedChainMessage is a signed (secp) message appearing on chain. Receipt is null if the message was not executed.
type SignedChainMessage struct {
	Message    *types.SignedMessage
	Receipt    *ChainMessageReceipt
	VmMessages VmMessageList
}

// ChainMessage is an unsigned (bls) message appearing on chain. Receipt is null if the message was not executed.
type ChainMessage struct {
	Message    *types.Message
	Receipt    *ChainMessageReceipt
	VmMessages VmMessageList
}

// ImplicitMessage is an implicitly executed message not appearing on chain.
type ImplicitMessage struct {
	Message    *types.Message
	Receipt    *ImplicitMessageReceipt
	VmMessages VmMessageList
}

// ChainMessageReceipt contains a MessageReceipt and other metadata.
type ChainMessageReceipt struct {
	Receipt    types.MessageReceipt `cborgen:"receipt"`
	GasOutputs *MessageGasOutputs   `cborgen:"gas"`
	ActorError *ActorError          `cborgen:"errors"`
	Index      int64                `cborgen:"index"`
}

type ImplicitMessageReceipt struct {
	Receipt    types.MessageReceipt `cborgen:"receipt"`
	GasOutputs *MessageGasOutputs   `cborgen:"gas"`
	ActorError *ActorError          `cborgen:"errors"`
}

// MessageGasOutputs contains the gas used during a message's execution.
type MessageGasOutputs struct {
	BaseFeeBurn        abi.TokenAmount `cborgen:"basefeeburn"`
	OverEstimationBurn abi.TokenAmount `cborgen:"overestimationburn"`
	MinerPenalty       abi.TokenAmount `cborgen:"minerpenalty"`
	MinerTip           abi.TokenAmount `cborgen:"minertip"`
	Refund             abi.TokenAmount `cborgen:"refund"`
	GasRefund          int64           `cborgen:"gasrufund"`
	GasBurned          int64           `cborgen:"gasburned"`
}

// ActorError contains any errors encountered during a message's execution.
type ActorError struct {
	Fatal   bool              `cborgen:"fatal"`
	RetCode exitcode.ExitCode `cborgen:"retcode"`
	Error   string            `cborgen:"error"`
}

func Messages(ctx context.Context, api tasks.DataSource, current, executed *types.TipSet) (*MessageStateChanges, error) {
	// fist get all messages included in the executed tipset, not all of these messages will have receipts since some were not executed.
	blkMsgs, err := api.TipSetBlockMessages(ctx, executed)
	if err != nil {
		return nil, err
	}
	// build two maps containing all signed (secpMsgs) and unsigned (blsMsgs) messages.
	secpBlkMsgs := make(map[cid.Cid][]*SignedChainMessage)
	blsBlkMsgs := make(map[cid.Cid][]*ChainMessage)
	secpMsgs := make(map[cid.Cid]*SignedChainMessage)
	blsMsgs := make(map[cid.Cid]*ChainMessage)
	for _, blk := range blkMsgs {
		blkCid := blk.Block.Cid()
		for _, msg := range blk.SecpMessages {
			secpMsg := &SignedChainMessage{Message: msg}
			secpBlkMsgs[blkCid] = append(secpBlkMsgs[blkCid], secpMsg)
			secpMsgs[msg.Cid()] = secpMsg
		}
		for _, msg := range blk.BlsMessages {
			blsMsg := &ChainMessage{Message: msg}
			blsBlkMsgs[blkCid] = append(blsBlkMsgs[blkCid], blsMsg)
			blsMsgs[msg.Cid()] = blsMsg
		}
	}

	exeBlkMsgs, err := api.TipSetMessageReceipts(ctx, current, executed)
	if err != nil {
		return nil, err
	}
	for _, ebm := range exeBlkMsgs {
		itr, err := ebm.Iterator()
		if err != nil {
			return nil, err
		}
		for itr.HasNext() {
			msg, recIdx, rec := itr.Next()
			if secpMsg, ok := secpMsgs[msg.Cid()]; ok {
				secpMsg.Receipt = &ChainMessageReceipt{
					Receipt: *rec,
					Index:   int64(recIdx),
				}
			} else if blsMsg, ok := blsMsgs[msg.Cid()]; ok {
				blsMsg.Receipt = &ChainMessageReceipt{
					Receipt: *rec,
					Index:   int64(recIdx),
				}
			} else {
				panic("developer error")
			}
		}
	}

	msgExe, err := api.MessageExecutionsV2(ctx, current, executed)
	if err != nil {
		return nil, err
	}

	var im []*ImplicitMessage
	for _, emsg := range msgExe {
		vmMsgs, err := ProcessVmMessages(ctx, emsg)
		if err != nil {
			return nil, err
		}
		if emsg.Implicit {
			im = append(im, &ImplicitMessage{
				Message:    emsg.Message,
				VmMessages: vmMsgs,
				Receipt: &ImplicitMessageReceipt{
					Receipt:    emsg.Ret.MessageReceipt,
					GasOutputs: GetMessageGasOutputs(emsg),
					ActorError: GetActorError(emsg),
				},
			})
		} else {
			if secpMsg, ok := secpMsgs[emsg.Cid]; ok {
				secpMsg.Receipt.GasOutputs = GetMessageGasOutputs(emsg)
				secpMsg.Receipt.ActorError = GetActorError(emsg)
				secpMsg.VmMessages = vmMsgs
			} else if blsMsg, ok := blsMsgs[emsg.Cid]; ok {
				blsMsg.Receipt.ActorError = GetActorError(emsg)
				blsMsg.Receipt.GasOutputs = GetMessageGasOutputs(emsg)
				blsMsg.VmMessages = vmMsgs
			} else {
				panic("developer error")
			}
		}
	}

	baseFee, err := api.ComputeBaseFee(ctx, executed)
	if err != nil {
		return nil, err
	}

	out := &MessageStateChanges{
		Current:          current,
		Executed:         executed,
		BaseFee:          baseFee,
		FullBlocks:       make(map[cid.Cid]*FullBlock),
		ImplicitMessages: im,
	}
	for _, blk := range blkMsgs {
		out.FullBlocks[blk.Block.Cid()] = &FullBlock{
			Block:        blk.Block,
			SecpMessages: secpBlkMsgs[blk.Block.Cid()],
			BlsMessages:  blsBlkMsgs[blk.Block.Cid()],
		}
	}
	return out, nil
}

func GetMessageGasOutputs(msg *lens.MessageExecutionV2) *MessageGasOutputs {
	if msg.Ret.GasCosts != nil {
		return &MessageGasOutputs{
			BaseFeeBurn:        msg.Ret.GasCosts.BaseFeeBurn,
			OverEstimationBurn: msg.Ret.GasCosts.OverEstimationBurn,
			MinerPenalty:       msg.Ret.GasCosts.MinerPenalty,
			MinerTip:           msg.Ret.GasCosts.MinerTip,
			Refund:             msg.Ret.GasCosts.Refund,
			GasRefund:          msg.Ret.GasCosts.GasRefund,
			GasBurned:          msg.Ret.GasCosts.GasBurned,
		}
	}
	return nil
}

func GetActorError(msg *lens.MessageExecutionV2) *ActorError {
	if msg.Ret.ActorErr != nil {
		return &ActorError{
			Fatal:   msg.Ret.ActorErr.IsFatal(),
			RetCode: msg.Ret.ActorErr.RetCode(),
			Error:   msg.Ret.ActorErr.Error(),
		}
	}
	return nil
}

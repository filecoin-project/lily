package cbor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	adtStore "github.com/filecoin-project/go-state-types/store"
	"github.com/filecoin-project/lotus/blockstore"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	v1car "github.com/ipld/go-car"
	"github.com/ipld/go-car/util"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm/schema"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/filecoin-project/lily/pkg/extract"
	cboractors "github.com/filecoin-project/lily/pkg/transform/cbor/actors"
	cbormessages "github.com/filecoin-project/lily/pkg/transform/cbor/messages"
)

var log = logging.Logger("lily/transform/cbor")

type RootStateIPLD struct {
	StateVersion uint64 `cborgen:"stateversion"`

	NetworkName    string `cborgen:"networkname"`
	NetworkVersion uint64 `cborgen:"networkversion"`

	State cid.Cid `cborgen:"state"` // StateExtractionIPLD
}

type RootStateModel struct {
	gorm.Model
	Height          uint64
	Cid             string
	StateVersion    uint64
	NetworkName     string
	NetworkVersion  uint64
	StateExtraction string
}

func (r *RootStateIPLD) Attributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.Int64("state_version", int64(r.StateVersion)),
		attribute.String("state_root", r.State.String()),
		attribute.String("network_name", r.NetworkName),
		attribute.Int64("network_version", int64(r.NetworkVersion)),
	}
}

func (r *RootStateIPLD) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for _, a := range r.Attributes() {
		enc.AddString(string(a.Key), a.Value.Emit())
	}
	return nil
}

type StateExtractionIPLD struct {
	Current types.TipSet `cborgen:"current"`
	Parent  types.TipSet `cborgen:"parent"`

	BaseFee             abi.TokenAmount `cborgen:"basefee"`
	FilVested           abi.TokenAmount `cborgen:"filvested"`
	FilMined            abi.TokenAmount `cborgen:"filmined"`
	FilBurnt            abi.TokenAmount `cborgen:"filburnt"`
	FilLocked           abi.TokenAmount `cborgen:"fillocked"`
	FilCirculating      abi.TokenAmount `cborgen:"filcirculating"`
	FilReserveDisbursed abi.TokenAmount `cborgen:"filreserveddisbursed"`

	FullBlocks       cid.Cid `cborgen:"fullblocks"`
	ImplicitMessages cid.Cid `cborgen:"implicitmessages"`
	Actors           cid.Cid `cborgen:"actors"`
}

type StateExtractionModel struct {
	gorm.Model
	Height           uint64
	CurrentTipSet    string
	ParentTipSet     string
	BaseFee          string
	FullBlocks       string
	ImplicitMessages string
	Actors           string
}

func (s *StateExtractionIPLD) Attributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("current_tipset", s.Current.Key().String()),
		attribute.String("parent_tipset", s.Parent.Key().String()),
		attribute.String("base_fee", s.BaseFee.String()),
		attribute.String("full_block_root", s.FullBlocks.String()),
		attribute.String("implicit_message_root", s.ImplicitMessages.String()),
		attribute.String("actors_root", s.Actors.String()),
	}
}

func (s *StateExtractionIPLD) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for _, a := range s.Attributes() {
		enc.AddString(string(a.Key), a.Value.Emit())
	}
	return nil
}

func NewTransformer(dir, prefix string) (*Transformer, error) {
	if dirInfo, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path (%s) does not exist: %w", dir, err)
		}
		if !dirInfo.IsDir() {
			return nil, fmt.Errorf("path (%s) is not a directory", dir)
		}
	}
	return &Transformer{carDirectory: dir, prefix: prefix}, nil
}

type Transformer struct {
	carDirectory string
	prefix       string
}

func (t *Transformer) Persist(ctx context.Context, chainState *extract.ChainState) error {
	start := time.Now()
	carPath := filepath.Join(t.carDirectory, fmt.Sprintf("%s_%d_%d.car", t.prefix, chainState.Parent.Height(), chainState.Current.Height()))
	f, err := os.OpenFile(carPath, os.O_CREATE|os.O_TRUNC|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	bs := blockstore.NewMemorySync()
	root, err := PersistToStore(ctx, bs, chainState)
	if err != nil {
		return err
	}
	bw := bufio.NewWriterSize(f, 1<<20)
	if err := WriteCarV1(ctx, root, bs, bw); err != nil {
		return err
	}
	log.Infow("created chain delta", "path", carPath, "duration", time.Since(start))
	return nil
}

func WriteCarV1(ctx context.Context, root cid.Cid, bs blockstore.Blockstore, w io.Writer) error {
	if err := v1car.WriteHeader(&v1car.CarHeader{
		Roots:   []cid.Cid{root},
		Version: 1,
	}, w); err != nil {
		return err
	}

	keyCh, err := bs.AllKeysChan(ctx)
	if err != nil {
		return err
	}

	for key := range keyCh {
		blk, err := bs.Get(ctx, key)
		if err != nil {
			return err
		}
		if err := util.LdWrite(w, blk.Cid().Bytes(), blk.RawData()); err != nil {
			return err
		}
	}
	return nil
}

func PersistToStore(ctx context.Context, bs blockstore.Blockstore, chainState *extract.ChainState) (cid.Cid, error) {
	store := adtStore.WrapBlockStore(ctx, bs)

	implicitMsgsAMT, err := cbormessages.MakeImplicitMessagesHAMT(ctx, store, chainState.Message.ImplicitMessages)
	if err != nil {
		return cid.Undef, err
	}

	fullBlkHAMT, err := cbormessages.MakeFullBlockHAMT(ctx, store, chainState.Message.FullBlocks)
	if err != nil {
		return cid.Undef, err
	}

	actorStateContainer, err := cboractors.ProcessActorsStates(ctx, store, chainState.Actors)
	if err != nil {
		return cid.Undef, err
	}

	actorStatesRoot, err := store.Put(ctx, actorStateContainer)
	if err != nil {
		return cid.Undef, err
	}

	extractedState := &StateExtractionIPLD{
		Current:             *chainState.Current,
		Parent:              *chainState.Parent,
		BaseFee:             chainState.Message.BaseFee,
		FilVested:           chainState.Message.CirculatingSupply.FilVested,
		FilMined:            chainState.Message.CirculatingSupply.FilMined,
		FilBurnt:            chainState.Message.CirculatingSupply.FilBurnt,
		FilLocked:           chainState.Message.CirculatingSupply.FilLocked,
		FilCirculating:      chainState.Message.CirculatingSupply.FilCirculating,
		FilReserveDisbursed: chainState.Message.CirculatingSupply.FilReserveDisbursed,
		FullBlocks:          fullBlkHAMT,
		ImplicitMessages:    implicitMsgsAMT,
		Actors:              actorStatesRoot,
	}

	extractedStateRoot, err := store.Put(ctx, extractedState)
	if err != nil {
		return cid.Undef, err
	}

	rootState := &RootStateIPLD{
		NetworkVersion: chainState.NetworkVersion,
		NetworkName:    chainState.NetworkName,
		StateVersion:   0,
		State:          extractedStateRoot,
	}

	root, err := store.Put(ctx, rootState)
	if err != nil {
		return cid.Undef, err
	}
	dsn := "host=localhost user=postgres password=password dbname=postgres port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "lily_cbor_",
			SingularTable: false,
		},
	})
	if err != nil {
		return cid.Undef, err
	}
	if err := db.AutoMigrate(&RootStateModel{}, &StateExtractionModel{}, &cboractors.ActorStateModel{}); err != nil {
		return cid.Undef, err
	}
	rootModel := RootStateModel{
		Height:          uint64(chainState.Current.Height()),
		Cid:             root.String(),
		StateVersion:    rootState.StateVersion,
		NetworkName:     rootState.NetworkName,
		NetworkVersion:  rootState.NetworkVersion,
		StateExtraction: rootState.State.String(),
	}
	stateModel := StateExtractionModel{
		Height:           uint64(chainState.Current.Height()),
		CurrentTipSet:    chainState.Current.String(),
		ParentTipSet:     chainState.Parent.String(),
		BaseFee:          chainState.Message.BaseFee.String(),
		FullBlocks:       fullBlkHAMT.String(),
		ImplicitMessages: implicitMsgsAMT.String(),
		Actors:           actorStatesRoot.String(),
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&rootModel).Error; err != nil {
			return err
		}
		if err := tx.Create(&stateModel).Error; err != nil {
			return err
		}
		as := actorStateContainer.AsModel()
		as.Height = uint64(chainState.Current.Height())
		if err := tx.Create(as).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return cid.Undef, err
	}
	return root, nil
}

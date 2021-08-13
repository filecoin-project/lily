package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/chain/types"
	cliutil "github.com/filecoin-project/lotus/cli/util"
	"github.com/filecoin-project/sentinel-visor/model/derived"
	"github.com/filecoin-project/sentinel-visor/storage"
	"github.com/go-pg/pg/v10"
	lru "github.com/hashicorp/golang-lru"
	logging "github.com/ipfs/go-log/v2"
	"github.com/labstack/echo/v4"
	"golang.org/x/xerrors"
)

var log = logging.Logger("transaction-api")

type APIMessage struct {
	Cid string `json:"cid"`

	// message details
	To       string `json:"to"`
	From     string `json:"from"`
	Value    string `json:"value"`
	Nonce    uint64 `json:"nonce"`
	Height   int64  `json:"height"`
	ExitCode int64  `json:"exit_code"`

	// gas values
	GasLimit   int64  `json:"gas_limit"`
	GasFeeCap  string `json:"gas_fee_cap"`
	GasPremium string `json:"gas_premium"`
	GasUsed    int64  `json:"gas_used"`

	// fees burns penalties refunds
	ParentBaseFee      string `json:"parent_base_fee"`
	BaseFeeBurn        string `json:"base_fee_burn"`
	OverEstimationBurn string `json:"over_estimation_burn"`
	MinerPenalty       string `json:"miner_penalty"`
	MinerTip           string `json:"miner_tip"`
	Refund             string `json:"refund"`
	GasRefund          int64  `json:"gas_refund"`

	// actor details
	ActorFamily string `json:"actor_family"`
	ActorName   string `json:"actor_name"`
	Method      uint64 `json:"method"`
}

type MessageReceipt struct {
	tableName  struct{} `pg:"message_receipts"`
	Cid        string
	Height     uint64
	To         string
	From       string
	Nonce      uint64
	Value      string
	GasFeeCap  uint64
	GasPremium uint64
	GasLimit   uint64
	Method     int
	GasUsed    uint64
	ExitCode   int
}

type Config struct {
	Listen   string
	URL      string
	Database string
	Schema   string
	Name     string
	PoolSize int
	LotusAPI string
}

func NewMessageAPI(cfg *Config) *MessageAPI {
	cache, err := lru.New(100_000)
	if err != nil {
		panic(err)
	}
	return &MessageAPI{cfg: cfg, resolveCache: cache}
}

type MessageAPI struct {
	cfg          *Config
	resolveCache *lru.Cache

	db       *pg.DB
	server   *echo.Echo
	closer   jsonrpc.ClientCloser
	lotusAPI api.FullNode
}

func (ix *MessageAPI) Init(ctx context.Context) error {
	logging.SetAllLoggers(logging.LevelInfo)

	ainfo := cliutil.ParseApiInfo(ix.cfg.LotusAPI)
	darg, err := ainfo.DialArgs("v1")
	if err != nil {
		return err
	}
	lotusAPI, closer, err := client.NewFullNodeRPCV1(ctx, darg, nil)
	if err != nil {
		return err
	}
	ix.lotusAPI = lotusAPI
	ix.closer = closer

	e := echo.New()
	e.GET("/index/msgs/to/:addr", func(c echo.Context) error {
		a, resolve, err := parseRequest(c)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}

		msgs, err := ix.MessagesTo(a, resolve)
		if err != nil {
			log.Errorw("MessagesTo", "address", a.String(), "error", err)
			return c.JSON(http.StatusInternalServerError, err.Error())
		}

		return c.JSON(http.StatusOK, msgs)
	})

	e.GET("/index/msgs/from/:addr", func(c echo.Context) error {
		a, resolve, err := parseRequest(c)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}

		msgs, err := ix.MessagesFrom(a, resolve)
		if err != nil {
			log.Errorw("MessagesFrom", "address", a.String(), "error", err)
			return c.JSON(http.StatusInternalServerError, err.Error())
		}

		return c.JSON(http.StatusOK, msgs)
	})

	e.GET("/index/msgs/for/:addr", func(c echo.Context) error {
		a, resolve, err := parseRequest(c)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}

		msgs, err := ix.MessagesFor(a, resolve)
		if err != nil {
			log.Errorw("MessagesFor", "address", a.String(), "error", err)
			return err
		}

		return c.JSON(http.StatusOK, msgs)
	})

	e.GET("/index/msgs/count", func(c echo.Context) error {
		count, err := ix.MessagesCount()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err.Error())
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"num_messages": count,
		})
	})
	ix.server = e

	db, err := storage.NewDatabase(ctx, ix.cfg.URL, ix.cfg.PoolSize, ix.cfg.Name, ix.cfg.Schema, false)
	if err != nil {
		return err
	}

	if err := db.Connect(ctx); err != nil {
		return err
	}
	db.AsORM().AddQueryHook(LogDebugHook{})
	ix.db = db.AsORM()
	return nil
}

func parseRequest(c echo.Context) (address.Address, bool, error) {
	addr := c.Param("addr")
	a, err := addrFromString(addr)
	if err != nil {
		return address.Undef, false, err
	}
	r := c.QueryParam("resolve")
	if r == "" {
		return a, false, nil
	}
	resolve, err := strconv.ParseBool(r)
	if err != nil {
		return address.Undef, false, err
	}
	return a, resolve, nil
}

func addrFromString(addrStr string) (address.Address, error) {
	if addrStr[0] == 't' {
		return address.Undef, xerrors.Errorf("cannot query testnet address %s", addrStr)
	}
	return address.NewFromString(addrStr)
}

type LogDebugHook struct {
}

func (l LogDebugHook) BeforeQuery(ctx context.Context, evt *pg.QueryEvent) (context.Context, error) {
	q, err := evt.FormattedQuery()
	if err != nil {
		return nil, err
	}

	if evt.Err != nil {
		log.Errorf("%s executing a query:%s", evt.Err, q)
	}
	fmt.Println(string(q))

	return ctx, nil
}

func (l LogDebugHook) AfterQuery(ctx context.Context, event *pg.QueryEvent) error {
	log.Infow("Executed", "duration", time.Since(event.StartTime), "rows_returned", event.Result.RowsReturned())
	return nil
}

func (ix *MessageAPI) Start() error {
	fmt.Println("Starting transactional api service")
	return ix.server.Start(ix.cfg.Listen)
}

func (ix *MessageAPI) Stop() {
	// close lotus api
	ix.closer()
	// close connection to DB
	if err := ix.db.Close(); err != nil {
		log.Errorw("stopping failed to close db", "error", err)
	}
	// shutdown http server
	if err := ix.server.Close(); err != nil {
		log.Errorw("stopping failed to close server", "error", err)
	}

}

func (ix *MessageAPI) MessagesCount() (int, error) {
	count, err := ix.db.Model(&derived.GasOutputs{}).Count()
	if err != nil {
		return 0, xerrors.Errorf("failed to find messages count: %w", err)
	}

	return count, nil
}

func (ix *MessageAPI) resolveAddress(ctx context.Context, addr address.Address) (address.Address, bool) {
	resAddr, found := ix.resolveCache.Get(addr)
	if found {
		return resAddr.(address.Address), true
	}
	switch addr.Protocol() {
	case address.BLS, address.SECP256K1:
		idAddr, err := ix.lotusAPI.StateLookupID(ctx, addr, types.EmptyTSK)
		if err != nil {
			log.Warnw("failed to look up address", "address", addr)
			return address.Undef, false
		}
		ix.resolveCache.Add(addr, idAddr)
		return idAddr, true
	case address.ID:
		// TODO this will fail for any address that isn't an account actor. The solution is to
		// call ResolveAddress on the Runtime. IDK where this is exposed
		// Problem you need to solve is to go from ID address to multisig address.
		pkAddr, err := ix.lotusAPI.StateAccountKey(ctx, addr, types.EmptyTSK)
		if err != nil {
			log.Warnw("failed to look up account key", "address", addr)
			return address.Undef, false
		}
		ix.resolveCache.Add(addr, pkAddr)
		return pkAddr, true
	case address.Actor:
		// TODO need a way to look this up
		return address.Undef, false
	}
	return address.Undef, false
}

func marshalResults(res []*derived.GasOutputs) []APIMessage {
	out := make([]APIMessage, len(res), len(res))
	for ix, r := range res {
		out[ix] = APIMessage{
			Cid:                r.Cid,
			To:                 r.To,
			From:               r.From,
			Value:              r.Value,
			Nonce:              r.Nonce,
			Height:             r.Height,
			ExitCode:           r.ExitCode,
			GasLimit:           r.GasLimit,
			GasFeeCap:          r.GasFeeCap,
			GasPremium:         r.GasPremium,
			GasUsed:            r.GasUsed,
			ParentBaseFee:      r.ParentBaseFee,
			BaseFeeBurn:        r.BaseFeeBurn,
			OverEstimationBurn: r.OverEstimationBurn,
			MinerPenalty:       r.MinerPenalty,
			MinerTip:           r.MinerTip,
			Refund:             r.Refund,
			GasRefund:          r.GasRefund,
			ActorFamily:        r.ActorFamily,
			ActorName:          r.ActorName,
			Method:             r.Method,
		}
	}
	return out
}

func (ix *MessageAPI) MessagesFor(addr address.Address, resolveAddr bool) ([]APIMessage, error) {
	var (
		ctx          = context.TODO()
		addrResolved = false
		res          []*derived.GasOutputs
		addr2        address.Address
	)

	if resolveAddr {
		addr2, addrResolved = ix.resolveAddress(ctx, addr)
	}
	if addrResolved {
		if err := ix.db.Model(&res).
			Order("height desc").
			Where("\"to\" = ? OR \"from\" = ? OR \"to\" = ? OR \"from\" = ?", addr.String(), addr.String(), addr2.String(), addr2.String()).
			Select(); err != nil {
			return nil, err
		}
	} else {
		if err := ix.db.Model(&res).
			Order("height desc").
			Where("\"to\" = ? OR \"from\" = ?", addr.String(), addr.String()).
			Select(); err != nil {
			return nil, err
		}
	}
	return marshalResults(res), nil
}

func (ix *MessageAPI) MessagesTo(addr address.Address, resolveAddr bool) ([]APIMessage, error) {
	var (
		ctx          = context.TODO()
		addrResolved = false
		res          []*derived.GasOutputs
		addr2        address.Address
	)

	if resolveAddr {
		addr2, addrResolved = ix.resolveAddress(ctx, addr)
	}
	if addrResolved {
		if err := ix.db.Model(&res).
			Order("height desc").
			Where("\"to\" = ? OR \"to\" = ?", addr.String(), addr2.String()).
			Select(); err != nil {
			return nil, err
		}
	} else {
		if err := ix.db.Model(&res).
			Order("height desc").
			Where("\"to\" = ?", addr.String()).
			Select(); err != nil {
			return nil, err
		}
	}
	return marshalResults(res), nil
}

func (ix *MessageAPI) MessagesFrom(addr address.Address, resolveAddr bool) ([]APIMessage, error) {
	var (
		ctx          = context.TODO()
		addrResolved = false
		res          []*derived.GasOutputs
		addr2        address.Address
	)

	if resolveAddr {
		addr2, addrResolved = ix.resolveAddress(ctx, addr)
	}
	if addrResolved {
		if err := ix.db.Model(&res).
			Order("height desc").
			Where("\"from\" = ? OR \"from\" = ?", addr.String(), addr2.String()).
			Select(); err != nil {
			return nil, err
		}
	} else {
		if err := ix.db.Model(&res).
			Order("height desc").
			Where("\"from\" = ?", addr.String()).
			Select(); err != nil {
			return nil, err
		}
	}
	return marshalResults(res), nil
}

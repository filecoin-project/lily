package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/sentinel-visor/model/derived"
	msgmodel "github.com/filecoin-project/sentinel-visor/model/messages"
	"github.com/filecoin-project/sentinel-visor/storage"
	"github.com/go-pg/pg/v10"
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
}

func NewMessageAPI(cfg *Config) *MessageAPI {
	return &MessageAPI{cfg: cfg}
}

type MessageAPI struct {
	cfg *Config

	db     *pg.DB
	server *echo.Echo
}

func (ix *MessageAPI) Init(ctx context.Context) error {
	logging.SetAllLoggers(logging.LevelInfo)
	e := echo.New()
	e.GET("/index/msgs/to/:addr", func(c echo.Context) error {
		addr := c.Param("addr")
		a, err := address.NewFromString(addr)
		if err != nil {
			return err
		}

		msgs, err := ix.MessagesTo(a)
		if err != nil {
			log.Errorw("MessagesTo", "address", a.String(), "error", err)
			return err
		}

		return c.JSON(http.StatusOK, msgs)
	})

	e.GET("/index/msgs/from/:addr", func(c echo.Context) error {
		addr := c.Param("addr")
		a, err := address.NewFromString(addr)
		if err != nil {
			return err
		}

		msgs, err := ix.MessagesFrom(a)
		if err != nil {
			log.Errorw("MessagesFrom", "address", a.String(), "error", err)
			return err
		}

		return c.JSON(http.StatusOK, msgs)
	})

	e.GET("/index/msgs/for/:addr", func(c echo.Context) error {
		addr := c.Param("addr")
		a, err := address.NewFromString(addr)
		if err != nil {
			return err
		}

		msgs, err := ix.MessagesFor(a, 200)
		if err != nil {
			log.Errorw("MessagesFor", "address", a.String(), "error", err)
			return err
		}

		return c.JSON(http.StatusOK, msgs)
	})

	e.GET("/index/msgs/count", func(c echo.Context) error {
		count, err := ix.MessagesCount()
		if err != nil {
			return err
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
	if err := ix.server.Close(); err != nil {
		log.Errorw("stopping failed to close server", "error", err)
	}
	if err := ix.db.Close(); err != nil {
		log.Errorw("stopping failed to close db", "error", err)
	}
}

func (ix *MessageAPI) MessagesCount() (int, error) {
	count, err := ix.db.Model(&msgmodel.Message{}).Count()
	if err != nil {
		return 0, xerrors.Errorf("failed to find messages to target: %w", err)
	}

	return count, nil
}

func (ix *MessageAPI) MessagesFor(addr address.Address, limit int) ([]APIMessage, error) {
	var res []*derived.GasOutputs
	if err := ix.db.Model(&res).
		Order("height desc").
		Where("\"to\" = ? OR \"from\" = ?", addr.String(), addr.String()).
		//Limit(limit).
		Select(); err != nil {
		return nil, err
	}
	out := make([]APIMessage, 0, len(res))
	for _, r := range res {
		out = append(out, APIMessage{
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
		})
	}
	return out, nil
}

func (ix *MessageAPI) MessagesTo(addr address.Address) ([]APIMessage, error) {
	var res derived.GasOutputsList
	if err := ix.db.Model(&res).
		Order("height desc").
		Where("\"to\" = ?", addr.String()).
		Select(); err != nil {
		return nil, err
	}
	out := make([]APIMessage, 0, len(res))
	for _, r := range res {
		out = append(out, APIMessage{
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
		})
	}
	return out, nil
}

func (ix *MessageAPI) MessagesFrom(addr address.Address) ([]APIMessage, error) {
	var res derived.GasOutputsList
	if err := ix.db.Model(&res).
		Order("height desc").
		Where("\"from\" = ?", addr.String()).
		Select(); err != nil {
		return nil, err
	}
	out := make([]APIMessage, 0, len(res))
	for _, r := range res {
		out = append(out, APIMessage{
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
		})
	}
	return out, nil
}

package lizard

import (
	"context"
	"time"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/chain/types"
	cliutil "github.com/filecoin-project/lotus/cli/util"
	"github.com/filecoin-project/sentinel-visor/storage"
	"github.com/go-pg/pg/v10"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("lizard")

type Config struct {
	URL      string
	Database string
	Schema   string
	Name     string
	PoolSize int
	LotusAPI string
}

type LizardAPI struct {
	cfg *Config

	db *pg.DB

	lotusAPI api.FullNode
	closer   jsonrpc.ClientCloser
	ticker   *time.Ticker
}

func NewLizard(cfg *Config) *LizardAPI {
	return &LizardAPI{cfg: cfg}
}

func (l *LizardAPI) Init(ctx context.Context) error {
	logging.SetAllLoggers(logging.LevelInfo)

	ainfo := cliutil.ParseApiInfo(l.cfg.LotusAPI)
	darg, err := ainfo.DialArgs("v1")
	if err != nil {
		return err
	}
	lotusAPI, closer, err := client.NewFullNodeRPCV1(ctx, darg, nil)
	if err != nil {
		return err
	}
	l.lotusAPI = lotusAPI
	l.closer = closer

	db, err := storage.NewDatabase(ctx, l.cfg.URL, l.cfg.PoolSize, l.cfg.Name, l.cfg.Schema, false)
	if err != nil {
		return err
	}

	if err := db.Connect(ctx); err != nil {
		return err
	}
	db.AsORM().AddQueryHook(LogDebugHook{})
	l.db = db.AsORM()
	return nil
}

func (l *LizardAPI) Stop() {
	l.ticker.Stop()
	// close lotus api
	l.closer()
	// close connection to DB
	if err := l.db.Close(); err != nil {
		log.Errorw("stopping failed to close db", "error", err)
	}
}

func (l *LizardAPI) Start(ctx context.Context) error {
	bch, err := l.lotusAPI.SyncIncomingBlocks(ctx)
	if err != nil {
		return err
	}
	go l.monitorSyncIncomingBlocks(ctx, bch)

	mch, err := l.lotusAPI.MpoolSub(ctx)
	if err != nil {
		return err
	}
	go l.monitorMpoolUpdates(ctx, mch)

	cch, err := l.lotusAPI.ChainNotify(ctx)
	if err != nil {
		return err
	}
	go l.monitorHeadEvents(ctx, cch)

	tick := time.NewTicker(time.Second * 10)
	go l.monitorPeers(ctx, tick.C)
	l.ticker = tick

	return nil
}

func (l *LizardAPI) monitorPeers(ctx context.Context, tick <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick:
			peers, err := l.lotusAPI.NetPeers(ctx)
			if err != nil {
				log.Errorw("netpeers", "error", err)
				continue
			}

			scores, err := l.lotusAPI.NetPubsubScores(ctx)
			if err != nil {
				log.Errorw("scores", "error", err)
				continue
			}
			for _, score := range scores {
				log.Infow("PubSub Score", "ID", score.ID, "score", score.Score.Score)
			}

			agents := map[string]int64{}
			for _, peer := range peers {
				pinfo, err := l.lotusAPI.NetPeerInfo(ctx, peer.ID)
				if err != nil {
					log.Debugw("failed to get peer info", "error", err)
					continue
				}
				log.Infow("pinfo", "first_seen", pinfo.ConnMgrMeta.FirstSeen.String(), "ID", pinfo.ID.ShortString())
				agents[pinfo.Agent]++
			}

			for agent, count := range agents {
				log.Infow("observed", "raw_agent", agent, "norm_agent", NormalizeAgent(agent), "count", count)
			}
		}
	}
}

func (l *LizardAPI) monitorHeadEvents(ctx context.Context, heChan <-chan []*api.HeadChange) {
	for {
		select {
		case <-ctx.Done():
			return
		case evts := <-heChan:
			for idx, e := range evts {
				log.Infow("HeadEvent", "idx", idx, "type", e.Type, "tipset", e.Val.String())
			}
		}
	}
}

func (l *LizardAPI) monitorSyncIncomingBlocks(ctx context.Context, blkChan <-chan *types.BlockHeader) {
	for {
		select {
		case <-ctx.Done():
			return
		case b := <-blkChan:
			log.Infow("SyncIncomingBlock", "block", b.Cid().String())
		}
	}
}

func (l *LizardAPI) monitorMpoolUpdates(ctx context.Context, mpoolChan <-chan api.MpoolUpdate) {
	for {
		select {
		case <-ctx.Done():
			return
		case m := <-mpoolChan:
			log.Infow("MpoolUpdate", "type", m.Type, "msg", m.Message.Cid().String())
		}
	}
}

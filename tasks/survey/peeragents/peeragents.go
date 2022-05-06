package peeragents

import (
	"context"
	"fmt"
	"regexp"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/filecoin-project/lily/model"
	"github.com/filecoin-project/lily/model/surveyed"
)

var log = logging.Logger("visor/tasks")

type API interface {
	ID(ctx context.Context) (peer.ID, error)
	NetPeers(context.Context) ([]peer.AddrInfo, error)
	NetAgentVersion(context.Context, peer.ID) (string, error)
}

func NewTask(api API) *Task {
	return &Task{
		api: api,
	}
}

type Task struct {
	api API
}

func (t *Task) Process(ctx context.Context) (model.Persistable, error) {
	pid, err := t.api.ID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get peer id: %w", err)
	}

	peers, err := t.api.NetPeers(ctx)
	if err != nil {
		return nil, fmt.Errorf("get peers: %w", err)
	}

	start := time.Now()
	agents := map[string]int64{}

	for _, peer := range peers {
		agent, err := t.api.NetAgentVersion(ctx, peer.ID)
		if err != nil {
			log.Debugw("failed to get agent version", "error", err)
			continue
		}
		agents[agent]++
	}

	var l observed.PeerAgentList

	for agent, count := range agents {
		pa := &observed.PeerAgent{
			SurveyerPeerID:  pid.String(),
			ObservedAt:      start,
			RawAgent:        agent,
			NormalizedAgent: NormalizeAgent(agent),
			Count:           count,
		}

		log.Debugw("observed", "raw_agent", pa.RawAgent, "norm_agent", pa.NormalizedAgent, "count", pa.Count)
		l = append(l, pa)
	}

	return l, nil
}

var nameAndVersion = regexp.MustCompile(`^(.+?)\+`)

// NormalizeAgent attempts to normalize an agent string to a software name and major/minor version
func NormalizeAgent(agent string) string {
	m := nameAndVersion.FindStringSubmatch(agent)
	if len(m) > 1 {
		return m[1]
	}

	return agent
}

func (t *Task) Close() error {
	return nil
}

package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/urfave/cli/v2"

	lotuscli "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/lotus/lib/addrutil"
)

var NetCmd = &cli.Command{
	Name:  "net",
	Usage: "Manage P2P Network",
	Subcommands: []*cli.Command{
		NetConnect,
		NetDisconnect,
		NetID,
		NetListen,
		NetPeers,
		NetReachability,
		NetScores,
	},
}

var NetID = &cli.Command{
	Name:  "id",
	Usage: "Get peer ID of libp2p node used by daemon",
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		lapi, closer, err := GetAPI(ctx)
		if err != nil {
			return fmt.Errorf("get api: %w", err)
		}
		defer closer()

		pid, err := lapi.ID(ctx)
		if err != nil {
			return fmt.Errorf("get id: %w", err)
		}

		fmt.Println(pid)
		return nil
	},
}

var NetDisconnect = &cli.Command{
	Name:      "disconnect",
	Usage:     "Disconnect from a peer",
	ArgsUsage: "[peerID]",
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		api, closer, err := GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		ids := cctx.Args().Slice()
		for _, id := range ids {
			pid, err := peer.Decode(id)
			if err != nil {
				fmt.Println("failure")
				return err
			}
			fmt.Printf("disconnect %s: ", pid.Pretty())
			err = api.NetDisconnect(ctx, pid)
			if err != nil {
				fmt.Println("failure")
				return err
			}
			fmt.Println("success")
		}
		return nil
	},
}

var NetConnect = &cli.Command{
	Name:      "connect",
	Usage:     "Connect to a peer",
	ArgsUsage: "[peerMultiaddr]",
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		api, closer, err := GetAPI(ctx)
		if err != nil {
			return err
		}
		defer closer()

		pis, err := addrutil.ParseAddresses(ctx, cctx.Args().Slice())
		if err != nil {
			return err
		}

		for _, pi := range pis {
			fmt.Printf("connect %s: ", pi.ID.Pretty())
			err := api.NetConnect(ctx, pi)
			if err != nil {
				fmt.Println("failure")
				return err
			}
			fmt.Println("success")
		}

		return nil
	},
}

var NetListen = &cli.Command{
	Name:  "listen",
	Usage: "List libp2p addresses daemon is listening on",
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		lapi, closer, err := GetAPI(ctx)
		if err != nil {
			return fmt.Errorf("get api: %w", err)
		}
		defer closer()

		addrs, err := lapi.NetAddrsListen(ctx)
		if err != nil {
			return err
		}

		for _, peer := range addrs.Addrs {
			fmt.Printf("%s/p2p/%s\n", peer, addrs.ID)
		}
		return nil
	},
}

var NetPeers = &cli.Command{
	Name:  "peers",
	Usage: "List peers daemon is connected to",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "agent",
			Aliases: []string{"a"},
			Usage:   "Print agent name",
		},
		&cli.BoolFlag{
			Name:    "extended",
			Aliases: []string{"x"},
			Usage:   "Print extended peer information in json",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		lapi, closer, err := GetAPI(ctx)
		if err != nil {
			return fmt.Errorf("get api: %w", err)
		}
		defer closer()

		peers, err := lapi.NetPeers(ctx)
		if err != nil {
			return err
		}

		sort.Slice(peers, func(i, j int) bool {
			return strings.Compare(string(peers[i].ID), string(peers[j].ID)) > 0
		})

		if cctx.Bool("extended") {
			// deduplicate
			seen := make(map[peer.ID]struct{})

			for _, peer := range peers {
				_, dup := seen[peer.ID]
				if dup {
					continue
				}
				seen[peer.ID] = struct{}{}

				info, err := lapi.NetPeerInfo(ctx, peer.ID)
				if err != nil {
					log.Warnf("error getting extended peer info: %s", err)
				} else {
					bytes, err := json.Marshal(&info)
					if err != nil {
						log.Warnf("error marshalling extended peer info: %s", err)
					} else {
						fmt.Println(string(bytes))
					}
				}
			}
		} else {
			w := tabwriter.NewWriter(os.Stdout, 4, 0, 1, ' ', 0)
			for _, peer := range peers {
				var agent string
				if cctx.Bool("agent") {
					agent, err = lapi.NetAgentVersion(ctx, peer.ID)
					if err != nil {
						log.Warnf("getting agent version: %s", err)
					}
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", peer.ID, peer.Addrs, agent)
			}
			if err := w.Flush(); err != nil {
				return err
			}

		}

		return nil
	},
}

var NetReachability = &cli.Command{
	Name:  "reachability",
	Usage: "Print information about reachability from the Internet",
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		lapi, closer, err := GetAPI(ctx)
		if err != nil {
			return fmt.Errorf("get api: %w", err)
		}
		defer closer()

		i, err := lapi.NetAutoNatStatus(ctx)
		if err != nil {
			return err
		}

		fmt.Println("AutoNAT status: ", i.Reachability.String())
		if len(i.PublicAddrs) > 0 {
			fmt.Println("Public address: ", i.PublicAddrs)
		}
		return nil
	},
}

var NetScores = &cli.Command{
	Name:  "scores",
	Usage: "List scores assigned to peers",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "extended",
			Aliases: []string{"x"},
			Usage:   "print extended peer scores in json",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := lotuscli.ReqContext(cctx)
		lapi, closer, err := GetAPI(ctx)
		if err != nil {
			return fmt.Errorf("get api: %w", err)
		}
		defer closer()

		scores, err := lapi.NetPubsubScores(ctx)
		if err != nil {
			return err
		}

		if cctx.Bool("extended") {
			enc := json.NewEncoder(os.Stdout)
			for _, peer := range scores {
				err := enc.Encode(peer)
				if err != nil {
					return err
				}
			}
		} else {
			w := tabwriter.NewWriter(os.Stdout, 4, 0, 1, ' ', 0)
			for _, peer := range scores {
				fmt.Fprintf(w, "%s\t%f\n", peer.ID, peer.Score.Score)
			}
			if err := w.Flush(); err != nil {
				return err
			}
		}

		return nil
	},
}

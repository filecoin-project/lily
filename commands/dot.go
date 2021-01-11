package commands

import (
	"fmt"
	"hash/crc32"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
)

type blockNode struct {
	Block        string
	Height       uint64
	Parent       string
	ParentHeight uint64
	Miner        string
}

func (b *blockNode) dotString() string {
	var result strings.Builder

	// write any null rounds before this block
	nulls := b.Height - b.ParentHeight - 1
	for i := uint64(0); i < nulls; i++ {
		name := b.Block + "NP" + fmt.Sprint(i)
		result.WriteString(fmt.Sprintf("%s [label = \"NULL:%d\", fillcolor = \"#ffddff\", style=filled, forcelabels=true]\n%s -> %s\n", name, b.Height-nulls+i, name, b.Parent))
		b.Parent = name
	}

	result.WriteString(fmt.Sprintf("%s [label = \"%s:%d\", fillcolor = \"#%06x\", style=filled, forcelabels=true]\n%s -> %s\n", b.Block, b.Miner, b.Height, b.dotColor(), b.Block, b.Parent))

	return result.String()
}

var defaultTbl = crc32.MakeTable(crc32.Castagnoli)

// dotColor intends to convert the Miner id into the RGBa color space
func (b *blockNode) dotColor() uint32 {
	return crc32.Checksum([]byte(b.Miner), defaultTbl)&0xc0c0c0c0 + 0x30303000 | 0x000000ff
}

var Dot = &cli.Command{
	Name:      "dot",
	Usage:     "Generate dot graphs for persisted blockchain starting from <minHeight> and includes the following <chainDistance> tipsets",
	ArgsUsage: "<startHeight> <chainDistance>",
	Action: func(cctx *cli.Context) error {
		if err := setupLogging(cctx); err != nil {
			return xerrors.Errorf("setup logging: %w", err)
		}

		db, err := setupDatabase(cctx)
		if err != nil {
			return xerrors.Errorf("setup database: %w", err)
		}
		defer func() {
			if err := db.Close(cctx.Context); err != nil {
				log.Errorw("close database", "error", err)
			}
		}()

		startHeight, err := strconv.ParseInt(cctx.Args().Get(0), 10, 32)
		if err != nil {
			return err
		}
		desiredChainLen, err := strconv.ParseInt(cctx.Args().Get(1), 10, 32)
		if err != nil {
			return err
		}
		endHeight := startHeight + desiredChainLen

		var blks = make([]*blockNode, desiredChainLen*5)
		var start = time.Now()
		_, err = db.DB.QueryContext(cctx.Context, &blks, `
with block_parents as (
	select
		miner_id,
		header_cid,
		regexp_split_to_table(parents, E',') as parent_header_cid
	from observed_headers
	where unix_to_height(header_timestamp) >= ? and unix_to_height(header_timestamp) <= ?
	group by 1, 2, 3
) select distinct
	bp.header_cid as block,
	bp.parent_header_cid as parent,
	b.miner_id as miner,
	unix_to_height(b.header_timestamp) as height,
	unix_to_height(p.header_timestamp) as parent_height
from block_parents bp
join observed_headers b on bp.header_cid = b.header_cid and bp.miner_id = b.miner_id
join observed_headers p on bp.parent_header_cid = p.header_cid
order by 4 desc`, startHeight, endHeight)
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Records received. (duration: %s)\n", time.Since(start))
		start = time.Now()

		fmt.Println("digraph D {")
		for _, b := range blks {
			fmt.Println(b.dotString())
		}
		fmt.Println("}")

		fmt.Fprintf(os.Stderr, "Output written. (duration: %s)\n", time.Since(start))
		return nil
	},
}

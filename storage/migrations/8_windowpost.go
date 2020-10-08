package migrations

import (
	"github.com/go-pg/migrations/v8"
)

// Schema version 8 provides additional tracking of the miner->deadline->partition->sector relation

func init() {

	up := batch(`
	CREATE TABLE IF NOT EXISTS "miner_sector_posts" (
		"miner_id" text not null,
		"sector_id" bigserial not null,
		"epoch" bigint not null,
		"post_message_cid" text,
		PRIMARY KEY ("miner_id", "sector_id", "epoch")
	);
`)
	down := batch(`
DROP TABLE IF EXISTS "miner_sector_posts";
`)
	migrations.MustRegisterTx(up, down)
}

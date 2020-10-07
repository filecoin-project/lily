package migrations

import (
	"github.com/go-pg/migrations/v8"
)

// Schema version 5 provides additional tracking of the miner->deadline->partition->sector relation

func init() {

	up := batch(`
	CREATE TABLE IF NOT EXISTS "miner_sector_posts" (
		"miner_id" text,
		"sector_id" number,
		"epoch" bigint,
		"post_message_id" ?,
		PRIMARY KEY ("miner_id", "sector_id", "epoch")
	);
`)
	down := batch(`
DROP TABLE IF EXISTS "miner_sector_posts";
`)
	migrations.MustRegisterTx(up, down)
}

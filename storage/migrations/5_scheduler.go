package migrations

import (
	"github.com/go-pg/migrations/v8"
)

// Schema version 5 is the new scheduler schema

func init() {
	up := batch(`
CREATE TABLE IF NOT EXISTS "visor_processing_statechanges" (
	"tip_set" text NOT NULL,
	"height" bigint,
	"added_at" timestamptz NOT NULL,
	"claimed_until" timestamptz,
	"completed_at" timestamptz,
	"errors_detected" text,
	PRIMARY KEY ("tip_set")
);

CREATE TABLE IF NOT EXISTS "visor_processing_actors" (
	"head" text NOT NULL,
	"code" text NOT NULL,
	"nonce" text, "balance"
	text, "address" text,
	"parent_state_root" text,
	"tip_set" text,
	"parent_tip_set" text,
	"height" bigint,
	"added_at" timestamptz NOT NULL,
	"claimed_until" timestamptz,
	"completed_at" timestamptz,
	"errors_detected" text,
	PRIMARY KEY ("head", "code")
);

CREATE TABLE IF NOT EXISTS "visor_processing_messages" (
	"tip_set" text NOT NULL,
	"height" bigint,
	"added_at" timestamptz NOT NULL,
	"claimed_until" timestamptz,
	"completed_at" timestamptz,
	"errors_detected" text,
	PRIMARY KEY ("tip_set")
);

`)
	down := batch(`
DROP TABLE IF EXISTS public.visor_processing_statechanges;

DROP TABLE IF EXISTS public.visor_processing_actors;

DROP TABLE IF EXISTS public.visor_processing_messages;
	`)
	migrations.MustRegisterTx(up, down)
}

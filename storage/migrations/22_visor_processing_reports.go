package migrations

import (
	"github.com/go-pg/migrations/v8"
)

// Schema version 22 adds the visor_processing_reports table

func init() {
	up := batch(`
CREATE TABLE IF NOT EXISTS public.visor_processing_reports (
	"height" bigint,
	"state_root" text,
	"reporter" text,
	"task" text,
	"started_at" timestamptz NOT NULL,
	"completed_at" timestamptz NOT NULL,
	"status" text,
	"status_information" text,
	"errors_detected" jsonb,
	PRIMARY KEY ("height","state_root","reporter", "task","started_at")
);
`)

	down := batch(`
	DROP TABLE IF EXISTS public.visor_processing_reports;
`)

	migrations.MustRegisterTx(up, down)
}

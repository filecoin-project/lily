package v0

func init() {
	up := batch(`
CREATE TABLE IF NOT EXISTS "parsed_messages" (
	"cid" text NOT NULL,
	"height" bigint NOT NULL,
	"from" text NOT NULL,
    "to" text NOT NULL,
	"value" text NOT NULL,
	"method" text NOT NULL,
	"params" jsonb,
	PRIMARY KEY ("cid")
);
CREATE INDEX IF NOT EXISTS "message_parsed_method_idx" ON public.parsed_messages USING HASH ("method");
CREATE INDEX IF NOT EXISTS "message_parsed_from_idx" ON public.parsed_messages USING HASH ("from");
CREATE INDEX IF NOT EXISTS "message_parsed_to_idx" ON public.parsed_messages USING HASH ("to");
`)
	down := batch(`
DROP TABLE IF EXISTS public.parsed_messages;
DROP INDEX IF EXISTS public.message_parsed_method_idx;
DROP INDEX IF EXISTS public.message_parsed_from_idx;
DROP INDEX IF EXISTS public.message_parsed_to_idx;
`)

	patches.MustRegisterTx(up, down)
}

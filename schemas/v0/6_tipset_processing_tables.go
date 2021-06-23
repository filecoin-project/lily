package v0

// Schema version 6 merges tipset oriented processing tables

func init() {
	up := batch(`
-- Merge visor_processing_statechanges and visor_processing_messages tables into a new tipset table
ALTER TABLE public.visor_processing_statechanges RENAME TO visor_processing_tipsets;
ALTER TABLE public.visor_processing_tipsets RENAME COLUMN claimed_until TO statechange_claimed_until;
ALTER TABLE public.visor_processing_tipsets RENAME COLUMN completed_at TO statechange_completed_at;
ALTER TABLE public.visor_processing_tipsets RENAME COLUMN errors_detected TO statechange_errors_detected;
ALTER TABLE public.visor_processing_tipsets ADD COLUMN message_claimed_until timestamptz;
ALTER TABLE public.visor_processing_tipsets ADD COLUMN message_completed_at timestamptz;
ALTER TABLE public.visor_processing_tipsets ADD COLUMN message_errors_detected text;

-- Copy data from visor_processing_messages before it is repurposed
UPDATE public.visor_processing_tipsets t
SET message_claimed_until = m.claimed_until,
    message_completed_at = m.completed_at,
    message_errors_detected = m.errors_detected
FROM public.visor_processing_messages m
WHERE m.tip_set = t.tip_set;


-- Repurpose visor_processing_messages table
DROP TABLE IF EXISTS public.visor_processing_messages;

CREATE TABLE public.visor_processing_messages (
	"cid" text NOT NULL,
	"height" bigint,
	"added_at" timestamptz NOT NULL,
	"gas_outputs_claimed_until" timestamptz,
	"gas_outputs_completed_at" timestamptz,
	"gas_outputs_errors_detected" text,
	PRIMARY KEY ("cid")
);

CREATE TABLE IF NOT EXISTS public.derived_gas_outputs (
	"cid" text NOT NULL,
	"from" text NOT NULL,
	"to" text NOT NULL,
	"value" text NOT NULL,
	"gas_fee_cap" text NOT NULL,
	"gas_premium" text NOT NULL,
	"gas_limit" bigint,
	"size_bytes" bigint,
	"nonce" bigint,
	"method" bigint,
	"state_root" text NOT NULL,
	"exit_code" bigint NOT NULL,
	"gas_used" bigint NOT NULL,
	"parent_base_fee" text NOT NULL,
	"base_fee_burn" text NOT NULL,
	"over_estimation_burn" text NOT NULL,
	"miner_penalty" text NOT NULL,
	"miner_tip" text NOT NULL,
	"refund" text NOT NULL,
	"gas_refund" bigint NOT NULL,
	"gas_burned" bigint NOT NULL,
	PRIMARY KEY ("cid")
);

CREATE INDEX derived_gas_outputs_from_index ON public.derived_gas_outputs USING hash ("from");
CREATE INDEX derived_gas_outputs_to_index ON public.derived_gas_outputs USING hash ("to");
CREATE INDEX derived_gas_outputs_method_index ON public.derived_gas_outputs USING btree (method);
CREATE INDEX derived_gas_outputs_exit_code_index ON public.derived_gas_outputs USING btree (exit_code);

`)
	down := batch(`


-- Restore visor_processing_messages table
DROP TABLE IF EXISTS public.visor_processing_messages;

CREATE TABLE public.visor_processing_messages (
	"tip_set" text NOT NULL,
	"height" bigint,
	"added_at" timestamptz NOT NULL,
	"claimed_until" timestamptz,
	"completed_at" timestamptz,
	"errors_detected" text,
	PRIMARY KEY ("tip_set")
);

-- Copy data back from visor_processing_tipsets
UPDATE public.visor_processing_messages m
SET claimed_until = t.message_claimed_until,
    completed_at = t.message_completed_at,
    errors_detected = t.message_errors_detected
FROM public.visor_processing_tipsets t
WHERE m.tip_set = t.tip_set;

-- Merge visor_processing_statechanges and visor_processing_messages tables into a new tipset table
ALTER TABLE public.visor_processing_tipsets RENAME TO visor_processing_statechanges;
ALTER TABLE public.visor_processing_statechanges RENAME COLUMN statechange_claimed_until TO claimed_until;
ALTER TABLE public.visor_processing_statechanges RENAME COLUMN statechange_completed_at TO completed_at;
ALTER TABLE public.visor_processing_statechanges RENAME COLUMN statechange_errors_detected TO errors_detected;
ALTER TABLE public.visor_processing_statechanges DROP COLUMN IF EXISTS message_claimed_until;
ALTER TABLE public.visor_processing_statechanges DROP COLUMN IF EXISTS message_completed_at;
ALTER TABLE public.visor_processing_statechanges DROP COLUMN IF EXISTS message_errors_detected;

DROP INDEX IF EXISTS public.derived_gas_outputs_from_index;
DROP INDEX IF EXISTS public.derived_gas_outputs_to_index;
DROP INDEX IF EXISTS public.derived_gas_outputs_method_index;
DROP INDEX IF EXISTS public.derived_gas_outputs_exit_code_index;


DROP TABLE IF EXISTS public.derived_gas_outputs;

	`)
	patches.MustRegisterTx(up, down)
}

package v0

// Schema version 26 removes work leasing tables https://github.com/filecoin-project/sentinel-visor/issues/311

func init() {
	up := batch(`
	DROP TABLE IF EXISTS public.visor_processing_tipsets;
	DROP TABLE IF EXISTS public.visor_processing_actors;
	DROP TABLE IF EXISTS public.visor_processing_messages;
	DROP TABLE IF EXISTS public.visor_processing_stats;
`)
	down := batch(`
	CREATE TABLE visor_processing_tipsets
	(
		tip_set                     text                     not null,
		height                      bigint                   not null,
		added_at                    timestamp with time zone not null,
		statechange_claimed_until   timestamp with time zone,
		statechange_completed_at    timestamp with time zone,
		statechange_errors_detected text,
		message_claimed_until       timestamp with time zone,
		message_completed_at        timestamp with time zone,
		message_errors_detected     text,
		economics_claimed_until     timestamp with time zone,
		economics_completed_at      timestamp with time zone,
		economics_errors_detected   text,
		constraint visor_processing_tipsets_pkey
			primary key (height, tip_set)
	);

	ALTER TABLE visor_processing_tipsets
		owner to postgres;

	CREATE INDEX visor_processing_tipsets_message_idx
		ON visor_processing_tipsets (height, message_claimed_until, message_completed_at);

	CREATE INDEX visor_processing_tipsets_statechange_idx
		ON visor_processing_tipsets (height, statechange_claimed_until, statechange_completed_at);

	CREATE INDEX visor_processing_tipsets_economics_idx
		ON visor_processing_tipsets (height, economics_claimed_until, economics_completed_at);

	CREATE INDEX visor_processing_tipsets_height_idx
		ON visor_processing_tipsets (height desc);


	CREATE TABLE visor_processing_actors
	(
		head              text                     not null,
		code              text                     not null,
		nonce             text,
		balance           text,
		address           text,
		parent_state_root text,
		tip_set           text,
		parent_tip_set    text,
		height            bigint                   not null,
		added_at          timestamp with time zone not null,
		claimed_until     timestamp with time zone,
		completed_at      timestamp with time zone,
		errors_detected   text,
		constraint visor_processing_actors_pkey
			primary key (height, head, code)
	);

	ALTER TABLE visor_processing_actors
		owner to postgres;

	CREATE INDEX visor_processing_actors_claimed_idx
		ON visor_processing_actors (height, claimed_until, completed_at);

	CREATE INDEX visor_processing_actors_codeclaimed_idx
		ON visor_processing_actors (code, height, claimed_until, completed_at);

	CREATE INDEX visor_processing_actors_height_idx
		ON visor_processing_actors (height desc);

	CREATE TABLE visor_processing_messages
	(
		cid                         text                     not null,
		height                      bigint                   not null,
		added_at                    timestamp with time zone not null,
		gas_outputs_claimed_until   timestamp with time zone,
		gas_outputs_completed_at    timestamp with time zone,
		gas_outputs_errors_detected text,
		constraint visor_processing_messages_pkey
			primary key (height, cid)
	);

	ALTER TABLE visor_processing_messages
		owner to postgres;

	CREATE INDEX visor_processing_messages_gas_outputs_idx
		ON visor_processing_messages (height, gas_outputs_claimed_until, gas_outputs_completed_at);

	CREATE INDEX visor_processing_messages_height_idx
		ON visor_processing_messages (height desc);

	CREATE TABLE visor_processing_stats
	(
		recorded_at timestamp with time zone not null,
		measure     text                     not null,
		tag         text                     not null,
		value       bigint                   not null,
		constraint visor_processing_stats_pkey
			primary key (recorded_at, measure, tag)
	);

	ALTER TABLE visor_processing_stats
		owner to postgres;

	CREATE INDEX visor_processing_stats_recorded_at_idx
		on visor_processing_stats (recorded_at desc);
`)

	patches.MustRegisterTx(up, down)
}

package v0

// Schema version 3 is the specs-actorsv0.9.11 schema used by visor

func init() {
	up := batch(`
ALTER TABLE public.miner_states ALTER COLUMN peer_id TYPE text;

ALTER TABLE public.chain_powers RENAME COLUMN minimum_consensus_miner_count TO participating_miner_count;
ALTER TABLE public.chain_powers DROP COLUMN new_raw_bytes_power;
ALTER TABLE public.chain_powers DROP COLUMN new_qa_bytes_power;
ALTER TABLE public.chain_powers DROP COLUMN new_pledge_collateral;
`)
	down := batch(`
ALTER TABLE public.miner_states ALTER COLUMN peer_id TYPE bytea
USING decode(public.miner_states.peer_id, 'escape');

ALTER TABLE public.chain_powers RENAME COLUMN participating_miner_count TO minimum_consensus_miner_count;
ALTER TABLE public.chain_powers ADD COLUMN new_raw_bytes_power text;
ALTER TABLE public.chain_powers ADD COLUMN new_qa_bytes_power text;
ALTER TABLE public.chain_powers ADD COLUMN new_pledge_collateral text;
`)
	Patches.MustRegisterTx(up, down)
}

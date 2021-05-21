package v0

// Schema version 8 adds columns for chain_economics processing

func init() {
	up := batch(`
-- chain_economics table already exists in chainwatch schema (schema version 1)


ALTER TABLE public.visor_processing_tipsets ADD COLUMN IF NOT EXISTS economics_claimed_until timestamptz;
ALTER TABLE public.visor_processing_tipsets ADD COLUMN IF NOT EXISTS economics_completed_at timestamptz;
ALTER TABLE public.visor_processing_tipsets ADD COLUMN IF NOT EXISTS economics_errors_detected text;

SELECT 1;
`)

	down := batch(`
ALTER TABLE public.visor_processing_tipsets DROP COLUMN IF EXISTS economics_claimed_until;
ALTER TABLE public.visor_processing_tipsets DROP COLUMN IF EXISTS economics_completed_at;
ALTER TABLE public.visor_processing_tipsets DROP COLUMN IF EXISTS economics_errors_detected;
`)
	Patches.MustRegisterTx(up, down)
}

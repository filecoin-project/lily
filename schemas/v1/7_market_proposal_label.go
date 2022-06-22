package v1

func init() {
	patches.Register(
		7,
		`
-- ----------------------------------------------------------------
-- Name: market_deal_proposals_v8
-- Model: market.MarketDealProposal
-- Growth: About 2 rows per epoch
-- ----------------------------------------------------------------
CREATE TABLE {{ .SchemaName | default "public"}}.market_deal_proposals_v8 (
    height bigint NOT NULL,
    deal_id bigint NOT NULL,
    state_root text NOT NULL,
    piece_cid text NOT NULL,
    padded_piece_size bigint NOT NULL,
    unpadded_piece_size bigint NOT NULL,
    is_verified boolean NOT NULL,
    client_id text NOT NULL,
    provider_id text NOT NULL,
    start_epoch bigint NOT NULL,
    end_epoch bigint NOT NULL,
    slashed_epoch bigint,
    storage_price_per_epoch text NOT NULL,
    provider_collateral text NOT NULL,
    client_collateral text NOT NULL,
    label text,
	is_string bool NOT NULL
);
ALTER TABLE ONLY {{ .SchemaName | default "public"}}.market_deal_proposals_v8 ADD CONSTRAINT market_deal_proposals_v8_pkey PRIMARY KEY (height, deal_id);
CREATE INDEX market_deal_proposals_height_v8_idx ON {{ .SchemaName | default "public"}}.market_deal_proposals_v8 USING btree (height DESC);

-- Convert market_deal_proposals_v8 to a hypertable partitioned on height (time)
-- Assume ~5  per epoch, ~350 bytes per table row
-- Height chunked per 7 days so we expect 20160*5 = ~100800 rows per chunk, 34MiB per chunk
SELECT create_hypertable(
	'market_deal_proposals_v8',
	'height',
	chunk_time_interval => 20160,
	if_not_exists => TRUE
);
SELECT set_integer_now_func('market_deal_proposals_v8', 'current_height', replace_if_exists => true);


COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals_v8.height IS 'Epoch at which this deal proposal was added or changed.';
COMMENT ON TABLE {{ .SchemaName | default "public"}}.market_deal_proposals_v8 IS 'All storage deal states with latest values applied to end_epoch when updates are detected on-chain.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals_v8.deal_id IS 'Identifier for the deal.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals_v8.state_root IS 'CID of the parent state root for this deal.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals_v8.piece_cid IS 'CID of a sector piece. A Piece is an object that represents a whole or part of a File.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals_v8.padded_piece_size IS 'The piece size in bytes with padding.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals_v8.unpadded_piece_size IS 'The piece size in bytes without padding.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals_v8.is_verified IS 'Deal is with a verified provider.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals_v8.client_id IS 'Address of the actor proposing the deal.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals_v8.provider_id IS 'Address of the actor providing the services.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals_v8.start_epoch IS 'The epoch at which this deal with begin. Storage deal must appear in a sealed (proven) sector no later than start_epoch, otherwise it is invalid.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals_v8.end_epoch IS 'The epoch at which this deal with end.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals_v8.storage_price_per_epoch IS 'The amount of FIL (in attoFIL) that will be transferred from the client to the provider every epoch this deal is active for.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals_v8.provider_collateral IS 'The amount of FIL (in attoFIL) the provider has pledged as collateral. The Provider deal collateral is only slashed when a sector is terminated before the deal expires.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals_v8.client_collateral IS 'The amount of FIL (in attoFIL) the client has pledged as collateral.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals_v8.label IS 'A base64 encoded arbitrary client chosen label to apply to the deal.';
COMMENT ON COLUMN {{ .SchemaName | default "public"}}.market_deal_proposals_v8.is_string IS 'When true the label columns contains a valid UTF-8 string encoded in bas64. When false Label contains raw bytes encoded in base64.';
`)
}

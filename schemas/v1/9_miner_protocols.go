package v1

// Schema patch 9 adds surveyed miner protocols and agents

func init() {
	patches.Register(
		9,
		`
	-- ----------------------------------------------------------------
	-- Name: surveyed_miner_protocols
	-- Model: surveyed.MinerProtocols
	-- Growth: N/A
	-- ----------------------------------------------------------------

	CREATE TABLE {{ .SchemaName | default "public"}}.surveyed_miner_protocols (
		observed_at			timestamp with time zone NOT NULL,
		miner_id			text NOT NULL,
		peer_id				text,
		agent				text,
		protocols 			jsonb,
		reachable 			boolean,
		error				text
	);
	ALTER TABLE ONLY {{ .SchemaName | default "public"}}.surveyed_miner_protocols ADD CONSTRAINT surveyed_miner_protocols_pkey PRIMARY KEY (observed_at, miner_id);

	COMMENT ON TABLE {{ .SchemaName | default "public"}}.surveyed_miner_protocols IS 'Observations of Filecoin storage provider supported protocols and agents over time.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.surveyed_miner_protocols.observed_at IS 'Timestamp of the observation.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.surveyed_miner_protocols.miner_id IS 'Address (ActorID) of the miner.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.surveyed_miner_protocols.peer_id IS 'PeerID of the miner advertised in on-chain MinerInfo structure.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.surveyed_miner_protocols.agent IS 'Agent string as reported by the peer.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.surveyed_miner_protocols.protocols IS 'List of supported protocol strings supported by the peer.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.surveyed_miner_protocols.reachable IS 'True if the peer could be connected to via its advertised multi-address in on-chain MinerInfo structure. False otherwise.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.surveyed_miner_protocols.error IS 'Contains any error encountered while connecting to peer or while querying its supported protocols or agent string.';
`)
}

package v1

// Schema patch 4 adds surveyed peer agents schema

func init() {
	patches.Register(
		4,
		`
	-- ----------------------------------------------------------------
	-- Name: surveyed_peer_agents
	-- Model: surveyed.PeerAgent
	-- Growth: N/A
	-- ----------------------------------------------------------------

	CREATE TABLE {{ .SchemaName | default "public"}}.surveyed_peer_agents (
		surveyer_peer_id  text NOT NULL,
		observed_at       timestamp with time zone NOT NULL,
		raw_agent         text NOT NULL,
		normalized_agent  text NOT NULL,
    	count             bigint NOT NULL
	);
	ALTER TABLE ONLY {{ .SchemaName | default "public"}}.surveyed_peer_agents ADD CONSTRAINT surveyed_peer_agents_pkey PRIMARY KEY (surveyer_peer_id, observed_at, raw_agent);

	COMMENT ON TABLE {{ .SchemaName | default "public"}}.surveyed_peer_agents IS 'Observations of filecoin peer agent strings over time.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.surveyed_peer_agents.surveyer_peer_id IS 'Peer ID of the node performing the survey.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.surveyed_peer_agents.observed_at IS 'Timestamp of the observation.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.surveyed_peer_agents.raw_agent IS 'Unprocessed agent string as reported by a peer.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.surveyed_peer_agents.normalized_agent IS 'Agent string normalized to a software name with major and minor version.';
	COMMENT ON COLUMN {{ .SchemaName | default "public"}}.surveyed_peer_agents.count IS 'Number of peers that reported the same raw agent.';
`)
}

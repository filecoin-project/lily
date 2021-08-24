package v1

// Schema version 1 adds gap report schema

func init() {
	patches.Register(
		3,
		`
	-- ----------------------------------------------------------------
	-- Name: visor_gap_reports
	-- Model: visor.GapReport
	-- Growth: N/A
	-- ----------------------------------------------------------------

	CREATE TABLE {{ .SchemaName | default "public"}}.visor_gap_reports (
		height 		bigint NOT NULL,
		task 		text NOT NULL,
		status 		text NOT NULL,
    	reporter 	text NOT NULL,
    	reported_at timestamp with time zone NOT NULL
	);
	ALTER TABLE ONLY {{ .SchemaName | default "public"}}.visor_gap_reports ADD CONSTRAINT visor_gap_reports_pkey PRIMARY KEY (height, task, status);
`)
}

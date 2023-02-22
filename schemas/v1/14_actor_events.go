package v1

func init() {
	patches.Register(
		14,
		`
	CREATE TABLE IF NOT EXISTS {{ .SchemaName | default "public"}}.actor_events (
	    height 			bigint 	NOT NULL,
	    state_root 		text 	NOT NULL,
	    event_index 	bigint 	NOT NULL,
	    message_cid 	text	NOT NULL,
	    
	    emitter 		text	NOT NULL,
	    flags 			bytea	NOT NULL,
	    codec			bigint 	NOT NULL,
	    key 			text 	NOT NULL,
	    value			bytea	NOT NULL,
	    
		PRIMARY KEY ("height", "state_root", "event_index", "message_cid")
	);
`,
	)
}

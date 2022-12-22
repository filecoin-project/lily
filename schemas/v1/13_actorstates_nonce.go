package v1

func init() {
	patches.Register(
		13,
		`
ALTER TABLE {{ .SchemaName | default "public"}}.actor_states
ADD COLUMN nonce BIGINT;

COMMENT ON COLUMN {{ .SchemaName | default "public"}}.actor_states.nonce IS 'The nonce of the actor expected to appear on the chain after the actor has been modified or created at each epoch. More precisely, this nonce tracks the number of messages sent by an actor.';
`,
	)
}

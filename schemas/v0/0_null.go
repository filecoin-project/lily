package v0

func init() {
	up := batch(`SELECT 1;`)
	down := batch(`SELECT 1;`)
	patches.MustRegisterTx(up, down)
}

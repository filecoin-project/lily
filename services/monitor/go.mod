module github.com/filecoin-project/monitor

go 1.16

require (
	github.com/filecoin-project/go-jsonrpc v0.1.4-0.20210217175800-45ea43ac2bec
	github.com/filecoin-project/lotus v1.11.0
	github.com/filecoin-project/sentinel-visor v0.7.5
	github.com/go-pg/pg/v10 v10.10.3
	github.com/ipfs/go-log/v2 v2.1.3
	github.com/urfave/cli/v2 v2.3.0
)

replace (
	github.com/filecoin-project/fil-blst => ../../extern/fil-blst
	github.com/filecoin-project/filecoin-ffi => ../../extern/filecoin-ffi
	github.com/filecoin-project/sentinel-visor => ../../.
	github.com/supranational/blst => ../../extern/fil-blst/blst
)

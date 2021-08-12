module github.com/filecoin-project/sentinel-visor/services/transaction-api

go 1.16

require (
	github.com/filecoin-project/go-address v0.0.6
	github.com/filecoin-project/sentinel-visor v0.7.2
	github.com/go-pg/pg/v10 v10.10.3
	github.com/ipfs/go-log v1.0.5 // indirect
	github.com/ipfs/go-log/v2 v2.1.3
	github.com/labstack/echo/v4 v4.4.0
	github.com/urfave/cli/v2 v2.3.0
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
)

replace (
	github.com/filecoin-project/fil-blst => ../../extern/fil-blst
	github.com/filecoin-project/filecoin-ffi => ../../extern/filecoin-ffi
	github.com/filecoin-project/sentinel-visor => ../../.
	github.com/supranational/blst => ../../extern/fil-blst/blst
)

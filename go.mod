module github.com/filecoin-project/sentinel-visor

go 1.14

require (
	github.com/filecoin-project/go-address v0.0.3
	github.com/filecoin-project/go-bitfield v0.2.0
	github.com/filecoin-project/go-jsonrpc v0.1.2-0.20200822201400-474f4fdccc52
	github.com/filecoin-project/go-state-types v0.0.0-20200905071437-95828685f9df
	github.com/filecoin-project/lotus v0.6.1
	github.com/filecoin-project/specs-actors v0.9.7
	github.com/go-pg/pg/v10 v10.3.1
	github.com/go-pg/pgext v0.1.4
	github.com/gocraft/work v0.5.1
	github.com/gomodule/redigo v1.8.2
	github.com/hashicorp/golang-lru v0.5.4
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-log/v2 v2.1.2-0.20200626104915-0016c0b4b3e4
	github.com/lib/pq v1.8.0
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/robfig/cron v1.2.0 // indirect
	github.com/stretchr/testify v1.6.1
	github.com/urfave/cli/v2 v2.2.0
	github.com/whyrusleeping/cbor-gen v0.0.0-20200826160007-0b9f6c5fb163
	go.opentelemetry.io/otel v0.12.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.12.0
	go.opentelemetry.io/otel/sdk v0.12.0
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
)

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi

replace github.com/supranational/blst => github.com/supranational/blst v0.1.2-alpha.1

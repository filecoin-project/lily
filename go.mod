module github.com/filecoin-project/sentinel-visor

go 1.14

require (
	contrib.go.opencensus.io/exporter/prometheus v0.1.0
	github.com/AndreasBriese/bbloom v0.0.0-20190825152654-46b345b51c96 // indirect
	github.com/davidlazar/go-crypto v0.0.0-20200604182044-b73af7476f6c // indirect
	github.com/filecoin-project/go-address v0.0.3
	github.com/filecoin-project/go-amt-ipld/v2 v2.1.1-0.20200731171407-e559a0579161 // indirect
	github.com/filecoin-project/go-bitfield v0.2.0
	github.com/filecoin-project/go-jsonrpc v0.1.2-0.20200822201400-474f4fdccc52
	github.com/filecoin-project/go-state-types v0.0.0-20200911004822-964d6c679cfc
	github.com/filecoin-project/lotus v0.8.0
	github.com/filecoin-project/specs-actors v0.9.11
	github.com/go-pg/migrations/v8 v8.0.1
	github.com/go-pg/pg/v10 v10.3.1
	github.com/go-pg/pgext v0.1.4
	github.com/gocraft/work v0.5.1
	github.com/gomodule/redigo v1.8.2
	github.com/hashicorp/golang-lru v0.5.4
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-log/v2 v2.1.2-0.20200626104915-0016c0b4b3e4
	github.com/lib/pq v1.8.0
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/prometheus/client_golang v1.6.0
	github.com/robfig/cron v1.2.0 // indirect
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/stretchr/testify v1.6.1
	github.com/urfave/cli/v2 v2.2.0
	github.com/whyrusleeping/cbor-gen v0.0.0-20200826160007-0b9f6c5fb163
	go.opencensus.io v0.22.4
	go.opentelemetry.io/otel v0.12.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.12.0
	go.opentelemetry.io/otel/sdk v0.12.0
	golang.org/x/exp v0.0.0-20200924195034-c827fd4f18b9 // indirect
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
)

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi

replace github.com/supranational/blst => ./extern/fil-blst/blst

replace github.com/filecoin-project/fil-blst => ./extern/fil-blst

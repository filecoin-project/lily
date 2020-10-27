module github.com/filecoin-project/sentinel-visor

go 1.14

require (
	contrib.go.opencensus.io/exporter/prometheus v0.1.0
	github.com/AndreasBriese/bbloom v0.0.0-20190825152654-46b345b51c96 // indirect
	github.com/filecoin-project/go-address v0.0.4
	github.com/filecoin-project/go-bitfield v0.2.1
	github.com/filecoin-project/go-fil-markets v1.0.0
	github.com/filecoin-project/go-jsonrpc v0.1.2-0.20201008195726-68c6a2704e49
	github.com/filecoin-project/go-multistore v0.0.3
	github.com/filecoin-project/go-state-types v0.0.0-20201013222834-41ea465f274f
	github.com/filecoin-project/lotus v1.1.0
	github.com/filecoin-project/specs-actors v0.9.12
	github.com/filecoin-project/specs-actors/v2 v2.2.0
	github.com/filecoin-project/statediff v0.0.8-0.20201027195725-7eaa5391a639
	github.com/go-pg/migrations/v8 v8.0.1
	github.com/go-pg/pg/v10 v10.3.1
	github.com/go-pg/pgext v0.1.4
	github.com/hashicorp/golang-lru v0.5.4
	github.com/ipfs/go-block-format v0.0.2
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-ipfs-blockstore v1.0.1
	github.com/ipfs/go-ipld-cbor v0.0.5-0.20200428170625-a0bd04d3cbdf
	github.com/ipfs/go-log/v2 v2.1.2-0.20200626104915-0016c0b4b3e4
	github.com/ipld/go-car v0.1.1-0.20201015032735-ff6ccdc46acc // indirect
	github.com/ipld/go-ipld-prime v0.5.1-0.20200910124733-350032422383
	github.com/jackc/pgx/v4 v4.9.0
	github.com/lib/pq v1.8.0
	github.com/libp2p/go-libp2p-peer v0.2.0
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/multiformats/go-multibase v0.0.3
	github.com/multiformats/go-multihash v0.0.14
	github.com/prometheus/client_golang v1.6.0
	github.com/raulk/clock v1.1.0
	github.com/stretchr/testify v1.6.1
	github.com/urfave/cli/v2 v2.2.0
	github.com/whyrusleeping/cbor-gen v0.0.0-20200826160007-0b9f6c5fb163
	github.com/willscott/carbs v0.0.3
	go.opencensus.io v0.22.4
	go.opentelemetry.io/otel v0.12.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.12.0
	go.opentelemetry.io/otel/sdk v0.12.0
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
)

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi

replace github.com/supranational/blst => ./extern/fil-blst/blst

replace github.com/filecoin-project/fil-blst => ./extern/fil-blst

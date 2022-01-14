module github.com/filecoin-project/lily

go 1.16

require (
	contrib.go.opencensus.io/exporter/prometheus v0.4.0
	github.com/BurntSushi/toml v0.4.1
	github.com/OneOfOne/xxhash v1.2.8 // indirect
	github.com/deckarep/golang-set v1.7.1
	github.com/filecoin-project/go-address v0.0.6
	github.com/filecoin-project/go-amt-ipld/v2 v2.1.1-0.20201006184820-924ee87a1349 // indirect
	github.com/filecoin-project/go-amt-ipld/v3 v3.1.1
	github.com/filecoin-project/go-bitfield v0.2.4
	github.com/filecoin-project/go-ds-versioning v0.1.1 // indirect
	github.com/filecoin-project/go-hamt-ipld/v3 v3.1.0
	github.com/filecoin-project/go-jsonrpc v0.1.5
	github.com/filecoin-project/go-paramfetch v0.0.3-0.20220111000201-e42866db1a53
	github.com/filecoin-project/go-state-types v0.1.3
	github.com/filecoin-project/lotus v1.14.0-rc1
	github.com/filecoin-project/specs-actors v0.9.14
	github.com/filecoin-project/specs-actors/v2 v2.3.6
	github.com/filecoin-project/specs-actors/v3 v3.1.1
	github.com/filecoin-project/specs-actors/v4 v4.0.1
	github.com/filecoin-project/specs-actors/v5 v5.0.4
	github.com/filecoin-project/specs-actors/v6 v6.0.1
	github.com/gammazero/workerpool v1.1.2
	github.com/go-pg/migrations/v8 v8.0.1
	github.com/go-pg/pg/v10 v10.10.6
	github.com/hashicorp/golang-lru v0.5.4
	github.com/ipfs/go-block-format v0.0.3
	github.com/ipfs/go-cid v0.1.0
	github.com/ipfs/go-ipfs-blockstore v1.1.2
	github.com/ipfs/go-ipld-cbor v0.0.6
	github.com/ipfs/go-log/v2 v2.4.0
	github.com/ipfs/go-metrics-prometheus v0.0.2
	github.com/ipld/go-car v0.3.3
	github.com/ipld/go-ipld-prime v0.14.3
	github.com/lib/pq v1.9.0
	github.com/libp2p/go-libp2p-core v0.13.0
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1
	github.com/minio/sha256-simd v1.0.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multiaddr v0.4.1
	github.com/multiformats/go-multihash v0.1.0
	github.com/polydawn/refmt v0.0.0-20201211092308-30ac6d18308e
	github.com/prometheus/client_golang v1.11.0
	github.com/raulk/clock v1.1.0
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/urfave/cli/v2 v2.3.0
	github.com/whyrusleeping/cbor-gen v0.0.0-20210713220151-be142a5ae1a8
	go.opencensus.io v0.23.0
	go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.20.0
	go.opentelemetry.io/otel/sdk v0.20.0
	go.opentelemetry.io/otel/trace v0.20.0
	go.uber.org/fx v1.15.0
	go.uber.org/zap v1.19.1
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	gopkg.in/cheggaaa/pb.v1 v1.0.28
)

replace (
	github.com/filecoin-project/fil-blst => ./extern/fil-blst
	github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi
	github.com/supranational/blst => ./extern/fil-blst/blst
)

module github.com/filecoin-project/sentinel-visor

go 1.15

require (
	contrib.go.opencensus.io/exporter/prometheus v0.1.0
	github.com/AndreasBriese/bbloom v0.0.0-20190825152654-46b345b51c96 // indirect
	github.com/OneOfOne/xxhash v1.2.8 // indirect
	github.com/fatih/color v1.10.0 // indirect
	github.com/filecoin-project/go-address v0.0.5
	github.com/filecoin-project/go-bitfield v0.2.4
	github.com/filecoin-project/go-bs-postgres-chainnotated v0.0.0-20210317220859-00d7e157c2a9
	github.com/filecoin-project/go-fil-markets v1.1.9
	github.com/filecoin-project/go-multistore v0.0.3
	github.com/filecoin-project/go-state-types v0.1.0
	github.com/filecoin-project/lotus v1.5.0
	github.com/filecoin-project/specs-actors v0.9.13
	github.com/filecoin-project/specs-actors/v2 v2.3.4
	github.com/filecoin-project/specs-actors/v3 v3.0.3
	github.com/filecoin-project/statediff v0.0.8-0.20201027195725-7eaa5391a639
	github.com/go-pg/migrations/v8 v8.0.1
	github.com/go-pg/pg/v10 v10.3.1
	github.com/go-pg/pgext v0.1.4
	github.com/google/go-cmp v0.5.4
	github.com/hashicorp/golang-lru v0.5.4
	github.com/ipfs/go-block-format v0.0.3
	github.com/ipfs/go-blockservice v0.1.4
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-datastore v0.4.5
	github.com/ipfs/go-graphsync v0.6.1-0.20210122235421-90b4d163a1bf // indirect
	github.com/ipfs/go-ipfs-exchange-offline v0.0.1
	github.com/ipfs/go-ipld-cbor v0.0.5
	github.com/ipfs/go-ipld-format v0.2.0
	github.com/ipfs/go-log/v2 v2.1.2
	github.com/ipfs/go-merkledag v0.3.2
	github.com/ipld/go-car v0.1.1-0.20201119040415-11b6074b6d4d
	github.com/ipld/go-ipld-prime v0.7.0
	github.com/kr/text v0.2.0 // indirect
	github.com/lib/pq v1.9.0
	github.com/libp2p/go-libp2p-core v0.7.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/multiformats/go-multihash v0.0.14
	github.com/onsi/ginkgo v1.14.2 // indirect
	github.com/onsi/gomega v1.10.4 // indirect
	github.com/prometheus/client_golang v1.6.0
	github.com/raulk/clock v1.1.0
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sirupsen/logrus v1.8.0 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/urfave/cli/v2 v2.3.0
	github.com/whyrusleeping/cbor-gen v0.0.0-20210303213153-67a261a1d291
	github.com/willscott/carbs v0.0.4
	go.opencensus.io v0.22.5
	go.opentelemetry.io/otel v0.12.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.12.0
	go.opentelemetry.io/otel/sdk v0.12.0
	golang.org/x/mod v0.4.0 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	gopkg.in/yaml.v2 v2.4.0 // indirect
	honnef.co/go/tools v0.1.2 // indirect
)

replace (
	github.com/filecoin-project/fil-blst => ./extern/fil-blst
	github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi-stub
	github.com/supranational/blst => ./extern/fil-blst/blst
)

// Supports go-ipld-prime v7
// TODO: remove once https://github.com/filecoin-project/statediff/pull/155 is merged
replace github.com/filecoin-project/statediff => github.com/filecoin-project/statediff v0.0.19-0.20210225063407-9e38aa4b7ede

// Supports go-ipld-prime v7
// TODO: remove once https://github.com/filecoin-project/go-hamt-ipld/pull/70 is merged to github.com/filecoin-project/go-hamt-ipld
replace github.com/filecoin-project/go-hamt-ipld/v2 => github.com/willscott/go-hamt-ipld/v2 v2.0.1-0.20210225034344-6d6dfa9b3960

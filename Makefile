PG_IMAGE?=postgres:10
REDIS_IMAGE?=redis:6
COMMIT := $(shell git rev-parse --short HEAD)

unexport GOFLAGS

BINS:=

GOFLAGS:=

.PHONY: all
all: build

.PHONY: build
build: sentinel-visor

.PHONY: deps
deps:
	git submodule update --init --recursive

.PHONY: sentinel-visor
sentinel-visor: extern/filecoin-ffi/.install-filcrypto
	rm -f ./sentinel-visor
	go build -o sentinel-visor .

# test starts dependencies and runs all tests
.PHONY: test
test: dockerup testfull dockerdown

.PHONY: dockerup
dockerup:
	docker-compose up -d

.PHONY: dockerdown
dockerdown:
	docker-compose down

# testfull runs all tests
.PHONY: testfull
testfull:
	TZ= PGSSLMODE=disable go test ./... -v

# testshort runs tests that don't require external dependencies such as postgres or redis
.PHONY: testshort
testshort:
	go test -short ./... -v

# only build filecoin-ffi if .install-filcrypto is missing
extern/filecoin-ffi/.install-filcrypto: deps
	$(MAKE) -C extern/filecoin-ffi

.PHONY: sentinel-visor
sentinel-visor: extern/filecoin-ffi/.install-filcrypto
	rm -f sentinel-visor
	go build $(GOFLAGS) -o sentinel-visor .

BINS+=sentinel-visor

.PHONY: docker-image
docker-image:
	docker build -t "filecoin/sentinel-visor" .
	docker tag "filecoin/sentinel-visor:latest" "filecoin/sentinel-visor:$(COMMIT)"


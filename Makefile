SHELL=/usr/bin/env bash

PG_IMAGE?=postgres:10
REDIS_IMAGE?=redis:6
COMMIT := $(shell git rev-parse --short HEAD)

# GITVERSION is the nearest tag plus number of commits and short form of most recent commit since the tag, if any
GITVERSION=$(shell git describe --always --tag --dirty)

unexport GOFLAGS

CLEAN:=
BINS:=

GOFLAGS:=

.PHONY: all
all: build

## FFI

FFI_PATH:=extern/filecoin-ffi/
FFI_DEPS:=.install-filcrypto
FFI_DEPS:=$(addprefix $(FFI_PATH),$(FFI_DEPS))

$(FFI_DEPS): build/.filecoin-install ;

build/.filecoin-install: $(FFI_PATH)
	$(MAKE) -C $(FFI_PATH) $(FFI_DEPS:$(FFI_PATH)%=%)
	@touch $@

MODULES+=$(FFI_PATH)
BUILD_DEPS+=build/.filecoin-install
CLEAN+=build/.filecoin-install

ffi-version-check:
	@[[ "$$(awk '/const Version/{print $$5}' extern/filecoin-ffi/version.go)" -eq 3 ]] || (echo "FFI version mismatch, update submodules"; exit 1)
BUILD_DEPS+=ffi-version-check

.PHONY: ffi-version-check


$(MODULES): build/.update-modules ;
# dummy file that marks the last time modules were updated
build/.update-modules:
	git submodule update --init --recursive
	touch $@

CLEAN+=build/.update-modules

# tools
toolspath:=support/tools

ldflags=-X=github.com/filecoin-project/sentinel-visor/version.GitVersion=$(GITVERSION)
ifneq ($(strip $(LDFLAGS)),)
	ldflags+=-extldflags=$(LDFLAGS)
endif
GOFLAGS+=-ldflags="$(ldflags)"

.PHONY: all
all: build

.PHONY: build
build: deps visor

.PHONY: deps
deps: $(BUILD_DEPS)

.PHONY: vector-setup
vector-setup: ./vector/data/.complete

./vector/data/.complete:
	cd ./vector; ./fetch_vectors.sh
	touch $@
CLEAN+=./vector/data/.complete
CLEAN+=./vector/data/*.json

# test starts dependencies and runs all tests
.PHONY: test
test: testfull

.PHONY: dockerup
dockerup:
	docker-compose up -d

.PHONY: dockerdown
dockerdown:
	docker-compose down

# testfull runs all tests
.PHONY: testfull
testfull: build
	docker-compose up -d
	sleep 2
	./visor migrate --latest
	-TZ= PGSSLMODE=disable go test ./... -v
	docker-compose down

# testshort runs tests that don't require external dependencies such as postgres or redis
.PHONY: testshort
testshort:
	go test -short ./... -v

.PHONY: visor
visor:
	rm -f visor
	go build $(GOFLAGS) -o visor -mod=readonly .
BINS+=visor

.PHONY: docker-image
docker-image:
	docker build -t "filecoin/sentinel-visor" .
	docker tag "filecoin/sentinel-visor:latest" "filecoin/sentinel-visor:$(COMMIT)"

.PHONY: clean
clean:
	rm -rf $(CLEAN) $(BINS)

.PHONY: dist-clean
dist-clean:
	git clean -xdff
	git submodule deinit --all -f

.PHONY: test-coverage
test-coverage:
	VISOR_TEST_DB="postgres://postgres:password@localhost:5432/postgres?sslmode=disable" go test -coverprofile=coverage.out ./...

# tools

$(toolspath)/bin/golangci-lint: $(toolspath)/go.mod
	@mkdir -p $(dir $@)
	(cd $(toolspath); go build -tags tools -o $(@:$(toolspath)/%=%) github.com/golangci/golangci-lint/cmd/golangci-lint)

$(toolspath)/bin/gen: $(toolspath)/go.mod
	@mkdir -p $(dir $@)
	(cd $(toolspath); go build -tags tools -o $(@:$(toolspath)/%=%) github.com/filecoin-project/statediff/types/gen)


.PHONY: lint
lint: $(toolspath)/bin/golangci-lint
	$(toolspath)/bin/golangci-lint run ./...

.PHONY: actors-gen
actors-gen:
	go run ./chain/actors/agen
	go fmt ./...


.PHONY: types-gen
types-gen: $(toolspath)/bin/gen
	$(toolspath)/bin/gen ./tasks/messages/types
	go fmt ./tasks/messages/types/...

# dev-nets
2k: GOFLAGS+=-tags=2k
2k: build

calibnet: GOFLAGS+=-tags=calibnet
calibnet: build

nerpanet: GOFLAGS+=-tags=nerpanet
nerpanet: build

butterflynet: GOFLAGS+=-tags=butterflynet
butterflynet: build

interopnet: GOFLAGS+=-tags=interopnet
interopnet: build

SHELL=/usr/bin/env bash

GO_BUILD_IMAGE?=golang:1.16.5
PG_IMAGE?=postgres:10
REDIS_IMAGE?=redis:6
VISOR_IMAGE_NAME?=filecoin/sentinel-visor
COMMIT := $(shell git rev-parse --short=8 HEAD)

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

# alias to match other network-specific targets
mainnet: build


# Dockerfiles

docker-files: Dockerfile Dockerfile.dev

Dockerfile:
	@echo "Writing ./Dockerfile..."
	@cat build/docker/header.tpl \
		build/docker/builder.tpl \
		build/docker/prod_entrypoint.tpl \
		> ./Dockerfile
CLEAN+=Dockerfile

Dockerfile.dev:
	@echo "Writing ./Dockerfile.dev..."
	@cat build/docker/header.tpl \
		build/docker/builder.tpl \
		build/docker/dev_entrypoint.tpl \
		> ./Dockerfile.dev
CLEAN+=Dockerfile.dev

# Docker images

# MAINNET
.PHONY: docker-mainnet
docker-mainnet: VISOR_DOCKER_FILE ?= Dockerfile
docker-mainnet: VISOR_NETWORK_TARGET ?= mainnet
docker-mainnet: docker-files docker-build-image-template

.PHONY: docker-mainnet-push
docker-mainnet-push: VISOR_IMAGE_TAG ?= $(COMMIT)
docker-mainnet-push: docker-mainnet docker-tag-and-push-template

.PHONY: docker-mainnet-dev
docker-mainnet-dev: VISOR_DOCKER_FILE ?= Dockerfile.dev
docker-mainnet-dev: VISOR_NETWORK_TARGET ?= mainnet
docker-mainnet-dev: docker-files docker-build-image-template

.PHONY: docker-mainnet-dev-push
docker-mainnet-dev-push: VISOR_IMAGE_TAG ?= $(COMMIT)
docker-mainnet-dev-push: docker-mainnet-dev docker-tag-and-push-template

# CALIBNET
.PHONY: docker-calibnet
docker-calibnet: VISOR_DOCKER_FILE ?= Dockerfile
docker-calibnet: VISOR_NETWORK_TARGET ?= calibnet
docker-calibnet: docker-files docker-build-image-template

.PHONY: docker-calibnet-push
docker-calibnet-push: VISOR_IMAGE_TAG ?= $(COMMIT)
docker-calibnet-push: docker-calibnet docker-tag-and-push-template

.PHONY: docker-calibnet-dev
docker-calibnet-dev: VISOR_DOCKER_FILE ?= Dockerfile.dev
docker-calibnet-dev: VISOR_NETWORK_TARGET ?= calibnet
docker-calibnet-dev: docker-files docker-build-image-template

.PHONY: docker-calibnet-dev-push
docker-calibnet-dev-push: VISOR_IMAGE_TAG ?= $(COMMIT)
docker-calibnet-dev-push: docker-calibnet-dev docker-tag-and-push-template


.PHONY: docker-build-image-template
docker-build-image-template:
	docker build -f $(VISOR_DOCKER_FILE) \
		--build-arg network_target=$(VISOR_NETWORK_TARGET) \
		--build-arg build_image=$(GO_BUILD_IMAGE) \
		-t $(VISOR_IMAGE_NAME) \
		-t $(VISOR_IMAGE_NAME):latest \
		-t $(VISOR_IMAGE_NAME):$(COMMIT) \
		.

.PHONY: docker-tag-and-push-template
docker-tag-and-push-template:
	./scripts/push-docker-tags.sh $(VISOR_IMAGE_NAME) deprecatedvalue $(VISOR_IMAGE_TAG)

.PHONY: docker-image
docker-image: docker-mainnet
	@echo "*** Deprecated make target 'docker-image': Please use 'make docker-mainnet' instead. ***"

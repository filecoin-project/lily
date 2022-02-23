SHELL=/usr/bin/env bash

GO_BUILD_IMAGE?=golang:1.17.6
PG_IMAGE?=postgres:10
REDIS_IMAGE?=redis:6
LILY_IMAGE_NAME?=filecoin/lily

# LILY_VERSION is the nearest tag plus number of commits and short form of
# most recent commit since the tag, if any. It may be overriden by the environment.
LILY_VERSION?=$(shell git describe --always --tag --dirty)
docker_sanitized_lily_version=$(shell echo ${LILY_VERSION} | sed 's:/:-:g')

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

ldflags=-X=github.com/filecoin-project/lily/version.GitVersion=$(LILY_VERSION)
ifneq ($(strip $(LDFLAGS)),)
	ldflags+=-extldflags=$(LDFLAGS)
endif
GOFLAGS+=-ldflags="$(ldflags)"

.PHONY: build
build: deps lily

.PHONY: deps
deps: $(BUILD_DEPS)

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
	./lily migrate --latest
	-TZ= PGSSLMODE=disable go test ./... -v
	docker-compose down

# testshort runs tests that don't require external dependencies such as postgres or redis
.PHONY: testshort
testshort:
	go test -short ./... -v


.PHONY: lily
lily:
	rm -f lily
	go build $(GOFLAGS) -o lily -mod=readonly .
BINS+=lily

.PHONY: clean
clean:
	rm -rf $(CLEAN) $(BINS)

.PHONY: dist-clean
dist-clean:
	git clean -xdff
	git submodule deinit --all -f

.PHONY: test-coverage
test-coverage:
	LILY_TEST_DB="postgres://postgres:password@localhost:5432/postgres?sslmode=disable" go test -coverprofile=coverage.out ./...

# tools

$(toolspath)/bin/golangci-lint: $(toolspath)/go.mod
	@mkdir -p $(dir $@)
	(cd $(toolspath); go build -tags tools -o $(@:$(toolspath)/%=%) github.com/golangci/golangci-lint/cmd/golangci-lint)

.PHONY: lint
lint: $(toolspath)/bin/golangci-lint
	$(toolspath)/bin/golangci-lint run ./...

.PHONY: actors-gen
actors-gen:
	go run ./chain/actors/agen
	go fmt ./...

.PHONY: itest
itest:
	LILY_TEST_DB="postgres://postgres:password@localhost:5432/postgres?sslmode=disable" go test --tags=integration ./itests/

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
docker-mainnet: LILY_DOCKER_FILE ?= Dockerfile
docker-mainnet: LILY_NETWORK_TARGET ?= mainnet
docker-mainnet: LILY_IMAGE_TAG ?= $(docker_sanitized_lily_version)
docker-mainnet: docker-build-image-template

.PHONY: docker-mainnet-push
docker-mainnet-push: docker-mainnet docker-tag-and-push-template

.PHONY: docker-mainnet-dev
docker-mainnet-dev: LILY_DOCKER_FILE ?= Dockerfile.dev
docker-mainnet-dev: LILY_NETWORK_TARGET ?= mainnet
docker-mainnet-dev: LILY_IMAGE_TAG ?= $(docker_sanitized_lily_version)-dev
docker-mainnet-dev: docker-build-image-template

.PHONY: docker-mainnet-dev-push
docker-mainnet-dev-push: docker-mainnet-dev docker-tag-and-push-template

# CALIBNET
.PHONY: docker-calibnet
docker-calibnet: LILY_DOCKER_FILE ?= Dockerfile
docker-calibnet: LILY_NETWORK_TARGET ?= calibnet
docker-calibnet: LILY_IMAGE_TAG ?= $(docker_sanitized_lily_version)-calibnet
docker-calibnet: docker-build-image-template

.PHONY: docker-calibnet-push
docker-calibnet-push: docker-calibnet docker-tag-and-push-template

.PHONY: docker-calibnet-dev
docker-calibnet-dev: LILY_DOCKER_FILE ?= Dockerfile.dev
docker-calibnet-dev: LILY_NETWORK_TARGET ?= calibnet
docker-calibnet-dev: LILY_IMAGE_TAG ?= $(docker_sanitized_lily_version)-calibnet-dev
docker-calibnet-dev: docker-build-image-template

.PHONY: docker-calibnet-dev-push
docker-calibnet-dev-push: docker-calibnet-dev docker-tag-and-push-template

# INTEROPNET
.PHONY: docker-interopnet
docker-interopnet: LILY_DOCKER_FILE ?= Dockerfile
docker-interopnet: LILY_NETWORK_TARGET ?= interopnet
docker-interopnet: LILY_IMAGE_TAG ?= $(docker_sanitized_lily_version)-interopnet
docker-interopnet: docker-build-image-template

.PHONY: docker-interopnet-push
docker-interopnet-push: docker-interopnet docker-tag-and-push-template

.PHONY: docker-interopnet-dev
docker-interopnet-dev: LILY_DOCKER_FILE ?= Dockerfile.dev
docker-interopnet-dev: LILY_NETWORK_TARGET ?= interopnet
docker-interopnet-dev: LILY_IMAGE_TAG ?= $(docker_sanitized_lily_version)-interopnet-dev
docker-interopnet-dev: docker-build-image-template

.PHONY: docker-interopnet-dev-push
docker-interopnet-dev-push: docker-interopnet-dev docker-tag-and-push-template

# BUTTERFLYNET
.PHONY: docker-butterflynet
docker-butterflynet: LILY_DOCKER_FILE ?= Dockerfile
docker-butterflynet: LILY_NETWORK_TARGET ?= butterflynet
docker-butterflynet: LILY_IMAGE_TAG ?= $(docker_sanitized_lily_version)-butterflynet
docker-butterflynet: docker-build-image-template

.PHONY: docker-butterflynet-push
docker-butterflynet-push: docker-butterflynet docker-tag-and-push-template

.PHONY: docker-butterflynet-dev
docker-butterflynet-dev: LILY_DOCKER_FILE ?= Dockerfile.dev
docker-butterflynet-dev: LILY_NETWORK_TARGET ?= butterflynet
docker-butterflynet-dev: LILY_IMAGE_TAG ?= $(docker_sanitized_lily_version)-butterflynet-dev
docker-butterflynet-dev: docker-build-image-template

.PHONY: docker-butterflynet-dev-push
docker-butterflynet-dev-push: docker-butterflynet-dev docker-tag-and-push-template

# NERPANET
.PHONY: docker-nerpanet
docker-nerpanet: LILY_DOCKER_FILE ?= Dockerfile
docker-nerpanet: LILY_NETWORK_TARGET ?= nerpanet
docker-nerpanet: LILY_IMAGE_TAG ?= $(docker_sanitized_lily_version)-nerpanet
docker-nerpanet: docker-build-image-template

.PHONY: docker-nerpanet-push
docker-nerpanet-push: docker-nerpanet docker-tag-and-push-template

.PHONY: docker-nerpanet-dev
docker-nerpanet-dev: LILY_DOCKER_FILE ?= Dockerfile.dev
docker-nerpanet-dev: LILY_NETWORK_TARGET ?= nerpanet
docker-nerpanet-dev: LILY_IMAGE_TAG ?= $(docker_sanitized_lily_version)-nerpanet-dev
docker-nerpanet-dev: docker-build-image-template

.PHONY: docker-nerpanet-dev-push
docker-nerpanet-dev-push: docker-nerpanet-dev docker-tag-and-push-template

# 2K
.PHONY: docker-2k
docker-2k: LILY_DOCKER_FILE ?= Dockerfile
docker-2k: LILY_NETWORK_TARGET ?= 2k
docker-2k: LILY_IMAGE_TAG ?= $(docker_sanitized_lily_version)-2k
docker-2k: docker-build-image-template

.PHONY: docker-2k-push
docker-2k-push: docker-2k docker-tag-and-push-template

.PHONY: docker-2k-dev
docker-2k-dev: LILY_DOCKER_FILE ?= Dockerfile.dev
docker-2k-dev: LILY_NETWORK_TARGET ?= 2k
docker-2k-dev: LILY_IMAGE_TAG ?= $(docker_sanitized_lily_version)-2k-dev
docker-2k-dev: docker-build-image-template

.PHONY: docker-2k-dev-push
docker-2k-dev-push: docker-2k-dev docker-tag-and-push-template

.PHONY: docker-build-image-template
docker-build-image-template: clean docker-files
	@echo "Building $(LILY_IMAGE_NAME):$(LILY_IMAGE_TAG) docker image for $(LILY_NETWORK_TARGET)..."
	docker build -f $(LILY_DOCKER_FILE) \
		--build-arg LILY_VERSION=$(LILY_VERSION) \
		--build-arg LILY_NETWORK_TARGET=$(LILY_NETWORK_TARGET) \
		--build-arg GO_BUILD_IMAGE=$(GO_BUILD_IMAGE) \
		-t $(LILY_IMAGE_NAME) \
		-t $(LILY_IMAGE_NAME):latest \
		-t $(LILY_IMAGE_NAME):$(LILY_IMAGE_TAG) \
		.

.PHONY: docker-tag-and-push-template
docker-tag-and-push-template:
	./scripts/push-docker-tags.sh $(LILY_IMAGE_NAME) $(LILY_IMAGE_TAG)

.PHONY: docker-image
docker-image: docker-mainnet
	@echo "*** Deprecated make target 'docker-image': Please use 'make docker-mainnet' instead. ***"

PG_IMAGE?=postgres:10
REDIS_IMAGE?=redis:6
COMMIT := $(shell git rev-parse --short HEAD)

# GITVERSION is the nearest tag plus number of commits and short form of most recent commit since the tag, if any
GITVERSION=$(shell git describe --always --tag --dirty)

unexport GOFLAGS

MODULES:=
CLEAN:=
BINS:=

GOFLAGS:=

ldflags=-X=github.com/filecoin-project/sentinel-visor/version.GitVersion=$(GITVERSION)
ifneq ($(strip $(LDFLAGS)),)
	ldflags+=-extldflags=$(LDFLAGS)
endif
GOFLAGS+=-ldflags="$(ldflags)"

.PHONY: all
all: build

.PHONY: build
build: deps visor

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

$(MODULES): build/.update-modules ;

# dummy file that marks the last time modules were updated
build/.update-modules:
	git submodule update --init --recursive
	touch $@

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
testfull:
	docker-compose up -d
	TZ= PGSSLMODE=disable go test ./... -v || echo ""
	docker-compose down

# testshort runs tests that don't require external dependencies such as postgres or redis
.PHONY: testshort
testshort:
	go test -short ./... -v

.PHONY: visor
visor:
	rm -f visor
	go build $(GOFLAGS) -o visor .

BINS+=visor

.PHONY: docker-image
docker-image:
	docker build -t "filecoin/sentinel-visor" .
	docker tag "filecoin/sentinel-visor:latest" "filecoin/sentinel-visor:$(COMMIT)"

clean:
	rm -rf $(CLEAN) $(BINS)
	-$(MAKE) -C $(FFI_PATH) clean
.PHONY: clean

dist-clean:
	git clean -xdff
	git submodule deinit --all -f
.PHONY: dist-clean

PG_IMAGE?=postgres:10
REDIS_IMAGE?=redis:6

unexport GOFLAGS

MODULES:=
CLEAN:=
BINS:=

GOFLAGS:=

.PHONY: all
all: build

.PHONY: build
build: sentinel-visor

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

.PHONY: sentinel-visor
sentinel-visor:
	rm -f sentinel-visor
	go build $(GOFLAGS) -o sentinel-visor .

BINS+=sentinel-visor

clean:
	rm -rf $(CLEAN) $(BINS)
	-$(MAKE) -C $(FFI_PATH) clean
.PHONY: clean

dist-clean:
	git clean -xdff
	git submodule deinit --all -f
.PHONY: dist-clean

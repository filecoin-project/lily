PG_IMAGE?=postgres:10

.PHONY: deps
deps:
	git submodule update --init --recursive

# test starts dependencies and runs all tests
.PHONY: test
test: pgstart testfull pgstop

# pgstart starts postgres in docker
.PHONY: pgstart
pgstart:
	docker run -d --name pg -p 5432:5432 -e POSTGRES_HOST_AUTH_METHOD=trust $(PG_IMAGE)
	sleep 10

# pgstop stops postgres in docker
.PHONY: pgstop
pgstop:
	docker rm -fv pg || true

# testfull runs all tests
.PHONY: testfull
testfull:
	TZ= PGSSLMODE=disable go test ./... -v

# testshort runs tests that don't require external dependencies such as postgres or redis
.PHONY: testshort
testshort:
	go test -short ./... -v

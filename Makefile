PG_IMAGE?=postgres:10
REDIS_IMAGE?=redis

.PHONY: deps
deps:
	git submodule update --init --recursive

# test starts dependencies and runs all tests
.PHONY: test
test: pgstart redisstart testfull redisstop pgstop

# pgstart starts postgres in docker
.PHONY: pgstart
pgstart:
	docker run -d --name pg -p 5432:5432 -e POSTGRES_HOST_AUTH_METHOD=trust $(PG_IMAGE)
	sleep 10

# pgstop stops postgres in docker
.PHONY: pgstop
pgstop:
	docker rm -fv pg || true

# redisstart starts redis in docker
.PHONY: redisstart
redisstart:
	docker run -d --name redis -p 6379:6379 $(REDIS_IMAGE)
	sleep 10

# redisstop stops redis in docker
.PHONY: redisstop
redisstop:
	docker rm -fv redis || true

# testfull runs all tests
.PHONY: testfull
testfull:
	TZ= PGSSLMODE=disable go test ./... -v

# testshort runs tests that don't require external dependencies such as postgres or redis
.PHONY: testshort
testshort:
	go test -short ./... -v

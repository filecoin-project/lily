PG_IMAGE?=postgres:10
REDIS_IMAGE?=redis:6

.PHONY: deps
deps:
	git submodule update --init --recursive

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

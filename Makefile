MAIN_PACKAGE_PATH := ./examples
BINARY_NAME := cache

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

.PHONY: no-dirty
no-dirty:
	git diff --exit-code


# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## upgrade: upgrade modfile
.PHONY: upgrade
upgrade:
	go get -u ./...

## tidy: format code and tidy modfile
.PHONY: tidy
tidy:
	go fmt ./...
	go mod tidy -v

## audit: run quality control checks
.PHONY: audit
audit:
	go mod verify
	go vet ./...
	golangci-lint run
	go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	go test -race -buildvcs -vet=off ./...


# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## clean: clean caches
.PHONY: clean
clean:
	go clean -testcache

## cleantest: clean testcache and run tests
.PHONY: cleantest
cleantest: clean test

## test: run all tests
.PHONY: test
test:
	go test -v -race -buildvcs ./...

## test/cover: run all tests and display coverage
.PHONY: test/cover
test/cover:
	go test -v -race -buildvcs -coverprofile=/tmp/coverage.out ./...
	go tool cover -html=/tmp/coverage.out

## bench: run all benchmarks
.PHONY: bench
bench:
	go test ./pkg/... -bench=.

## bench/all: run all benchmarks
.PHONY: bench/all
bench/all:
	go test ./... -bench=.

.PHONY: generate
generate:
	go generate ./...

## build: build the application
.PHONY: build
build: generate
	go run examples/main.go

## run: run the  application
.PHONY: run
run: build
	/tmp/bin/${BINARY_NAME}

# ==================================================================================== #
# OPERATIONS
# ==================================================================================== #

## install: install dependencies
.PHONY: install
install:
	go install github.com/tinylib/msgp@latest
	brew install golangci-lint
	go mod download

## push: push changes to the remote Git repository
.PHONY: push
push: tidy audit no-dirty
	git push

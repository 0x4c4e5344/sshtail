ifneq (, $(shell which tput))
	GREEN  := $(shell tput -Txterm setaf 2)
	YELLOW := $(shell tput -Txterm setaf 3)
	WHITE  := $(shell tput -Txterm setaf 7)
	CYAN   := $(shell tput -Txterm setaf 6)
	RESET  := $(shell tput -Txterm sgr0)
endif

GO=go

## Git version
GIT_VERSION ?= $(shell git describe --abbrev=6 --always --tags)


BINARY_NAME=sshtail
VERSION?=$(GIT_VERSION)
CI?=false

BUILD_DIR?=./build

.DEFAULT_GOAL := all

.PHONY: all test build

all: clean build test


## Bootstrap:
bootstrap:  ## Bootstrap project (sub-tasks: deps)
bootstrap: deps

deps:	## Install dependencies
	$(GO) get ./...
	$(GO) get -t ./...

## Format:
format:	## Format source files (alias: fmt)
	$(GO) fmt ./...

fmt: format


## Test:
test:	## Run all tests (unit and integration tests)
test: test-unit

test-unit:	## Run unit tests
	$(GO) test -v -race ./cmd/... ./specfile/... $(BUILD_OPTIONS)

## Build:
build:	## Build main (output dir: ./build/bin/sshtail)
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/bin/$(BINARY_NAME) ./main.go

clean: 	## Remove output files
	@rm -fr $(BUILD_DIR)


## Help:
help:	## Show this help
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} { \
		if (/^[a-zA-Z_-]+:.*?##.*$$/) {printf "    ${YELLOW}%-20s${GREEN}%s${RESET}\n", $$1, $$2} \
		else if (/^## .*$$/) {printf "  ${CYAN}%s${RESET}\n", substr($$1,4)} \
		}' $(MAKEFILE_LIST)

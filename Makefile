APP := dispatch
BIN_DIR := bin
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//')
ifeq ($(strip $(VERSION)),)
VERSION := dev
endif
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || printf none)
ifeq ($(strip $(COMMIT)),)
COMMIT := none
endif
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X github.com/matthias/dispatch/cmd.version=$(VERSION) -X github.com/matthias/dispatch/cmd.commit=$(COMMIT) -X github.com/matthias/dispatch/cmd.date=$(DATE)

.PHONY: help build install run test test-version clean

help:
	@echo "dispatch targets:"
	@echo "  make build    build ./$(BIN_DIR)/$(APP)"
	@echo "  make install  install $(APP) into GOPATH/bin"
	@echo "  make run      run the TUI with go run"
	@echo "  make test     run all Go tests"
	@echo "  make clean    remove build artifacts"

build:
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP) .

install:
	go install -ldflags "$(LDFLAGS)" .

run:
	go run .

test:
	sh scripts/test_version_metadata.sh
	go test ./...

test-version:
	sh scripts/test_version_metadata.sh

clean:
	rm -rf $(BIN_DIR)

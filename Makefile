APP := dispatch
BIN_DIR := bin
VERSION ?= dev
COMMIT ?= none
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X github.com/matthias/dispatch/cmd.version=$(VERSION) -X github.com/matthias/dispatch/cmd.commit=$(COMMIT) -X github.com/matthias/dispatch/cmd.date=$(DATE)

.PHONY: help build install run test clean

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
	go test ./...

clean:
	rm -rf $(BIN_DIR)

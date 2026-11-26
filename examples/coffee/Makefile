.DEFAULT_GOAL := help
GO ?= go
GOLANGCI_LINT ?= golangci-lint
ARGS ?=
override export GOWORK := off
override export GOTOOLCHAIN := local
IOC := github.com/longxi-go/ioc
NEMO := github.com/longxi-go/nemo
GIN := $(IOC)/adaptors/gin
NEMO_VERSION := $(shell GOWORK=off GOTOOLCHAIN=local $(GO) list -m -f '{{.Version}}' $(NEMO))

.PHONY: help setup generate check-generated format check-format verify-mod vet test race lint check build run
help:
	@printf '%s\n' 'Usage: make <target>' '' \
	  '  setup             Verify the parent shared formatter' \
	  '  generate          Generate configuration bindings and IoC composition' \
	  '  check-generated   Verify both generated outputs without writing' \
	  '  format            Format owned Go sources using shared golayout' \
	  '  check-format      Check layout and source notices' \
	  '  verify-mod        Check standalone module tidiness and checksums' \
	  '  vet               Run go vet' \
	  '  test              Run unit and SQLite/HTTP integration tests with coverage' \
	  '  race              Run shuffled race tests' \
	  '  lint              Run the repository linter policy' \
	  '  check             Run all example quality gates' \
	  '  build             Build bin/coffee' \
	  '  run               Start HTTP (optional ARGS="-profile dev -db /tmp/coffee.db")'
setup:
	$(MAKE) -C ../.. setup
generate:
	$(GO) run $(NEMO)/cmd/nemoctl@$(NEMO_VERSION) generate --roots ./internal/configuration
	$(GO) tool $(GIN)/cmd/ginctl generate --entry ./cmd/coffee --roots ./internal/...
check-generated:
	$(GO) run $(NEMO)/cmd/nemoctl@$(NEMO_VERSION) check --roots ./internal/configuration
	$(GO) tool $(GIN)/cmd/ginctl check --entry ./cmd/coffee --roots ./internal/...
format:
	$(MAKE) -C ../.. format
check-format:
	$(MAKE) -C ../.. check-format check-license-headers
verify-mod:
	$(GO) mod tidy -diff
	$(GO) mod verify
vet:
	$(GO) vet ./...
test:
	$(GO) test -coverpkg=./... -coverprofile=coverage.out ./...
race:
	$(GO) test -race -count=1 -shuffle=on ./...
lint:
	$(GOLANGCI_LINT) run --config ../../.golangci.yml --timeout=10m
check: check-format verify-mod vet test race lint check-generated
build:
	$(GO) build -o bin/coffee ./cmd/coffee
run:
	$(GO) run ./cmd/coffee $(ARGS)
.DEFAULT:
	@printf 'Unknown target: %s. Run make help for available targets.\n' "$@" >&2
	@exit 2

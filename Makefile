.DEFAULT_GOAL := help

GO ?= go
UV ?= uv
GOLANGCI_LINT ?= golangci-lint
TAG ?=
REPOSITORY ?= longxi-go/ioc
WORKSPACE ?=
export GO GOLANGCI_LINT
ifeq ($(strip $(WORKSPACE)),)
override export GOWORK := off
override export IOC_TEST_WORKSPACE :=
else
# Empty GOWORK keeps Go's directory-based discovery, including isolated fixtures.
override export GOWORK :=
override export IOC_TEST_WORKSPACE := $(abspath $(WORKSPACE))
endif
override export GOTOOLCHAIN := local

.PHONY: help setup format check-format test-format test race vet lint verify-mod check generate check-generated build

.PHONY: docs-setup docs-serve docs-build compatibility test-scripts validate-release preflight-release verify-environment
.PHONY: license-headers check-license-headers

help:
	@printf '%s\n' 'Usage: make <target>' '' \
	  '  help              Show this target list (default)' \
	  '  check WORKSPACE=../go.work  Verify explicitly selected local dependencies' \
	  '  license-headers   Refresh owned Go source license headers' \
	  '  check-license-headers  Check source license headers without writes' \
	  '  docs-setup        Install locked documentation dependencies (Python 3.12)' \
	  '  docs-serve        Preview bilingual documentation locally' \
	  '  docs-build        Strictly build both documentation languages' \
	  '  setup             Verify shared golayout tooling' \
	  '  verify-environment Reject hidden Go module/workspace overrides' \
	  '  format            Rewrite owned Go files with shared golayout' \
	  '  check-format      Check shared Go layout without writes' \
	  '  test-format       Test shared layout, ownership and failures' \
	  '  test              Run all tests with statement coverage' \
	  '  race              Run shuffled race tests' \
	  '  vet               Run go vet' \
	  '  lint              Run golangci-lint (must be installed)' \
	  '  verify-mod        Check module dependency tidiness' \
	  '  generate          Regenerate owned application examples' \
	  '  check-generated   Verify generated output freshness' \
	  '  build             Build ioctl into .tmp/bin' \
	  '  test-scripts      Test release and formatter adapters' \
	  '  compatibility     Verify the selected Go toolchain' \
	  '  validate-release  Validate TAG=vX.Y.Z and English notes' \
	  '  preflight-release Check published dependencies and Pages settings' \
	  '  check             Run all required local quality gates'

verify-environment:
	@python3 scripts/go_environment.py
	@if [ -n "$(WORKSPACE)" ]; then \
	  test "$$($(GO) env GOWORK)" = "$(abspath $(WORKSPACE))" || \
	  { printf '%s\n' 'WORKSPACE must match the go.work discovered from this checkout' >&2; exit 2; }; \
	fi

setup format check-format test-format license-headers check-license-headers test race vet lint verify-mod generate check-generated build validate-release: verify-environment

setup:
	@python3 scripts/golayout.py setup
license-headers:
	@python3 scripts/golayout.py headers-write
check-license-headers:
	@python3 scripts/golayout.py headers-check
format:
	@python3 scripts/golayout.py write
check-format:
	@python3 scripts/golayout.py check
test-format:
	@python3 scripts/golayout.py test

test:
	$(GO) test -cover ./...
race:
	$(GO) test -race -shuffle=on ./...
vet:
	$(GO) vet ./...
lint:
	$(GOLANGCI_LINT) run --timeout=10m
verify-mod:
	$(GO) mod tidy -diff
	$(GO) mod verify
generate:
	$(GO) run ./cmd/ioctl generate --entry ./examples/hello --roots ./examples/hello/...
	$(GO) run ./cmd/ioctl generate --entry ./examples/events --roots ./examples/events/...
	$(GO) run ./cmd/ioctl generate --entry ./examples/prerequisites --roots ./examples/prerequisites/...
	$(GO) run ./cmd/ioctl generate --entry ./examples/environment --roots ./examples/environment/...
check-generated:
	$(GO) run ./cmd/ioctl check --entry ./examples/hello --roots ./examples/hello/...
	$(GO) run ./cmd/ioctl check --entry ./examples/events --roots ./examples/events/...
	$(GO) run ./cmd/ioctl check --entry ./examples/prerequisites --roots ./examples/prerequisites/...
	$(GO) run ./cmd/ioctl check --entry ./examples/environment --roots ./examples/environment/...
compatibility: verify-mod vet test

test-scripts:
	python3 -m unittest discover -s scripts/tests -v

validate-release:
	python3 scripts/release.py --tag "$(TAG)"

preflight-release: validate-release verify-mod
	python3 scripts/release.py --tag "$(TAG)" --check-github --repository "$(REPOSITORY)"

check: check-license-headers check-format test-format verify-mod vet test race lint check-generated test-scripts
build:
	$(GO) build -o .tmp/bin/ioctl ./cmd/ioctl

docs-setup:
	cd docs/design && $(UV) sync --locked
docs-serve: docs-setup
	cd docs/design && $(UV) run --locked mkdocs serve
docs-build: docs-setup
	cd docs/design && $(UV) run --locked mkdocs build --strict

.DEFAULT:
	@printf 'Unknown target: %s. Available targets follow.\n' "$@" >&2
	@$(MAKE) --no-print-directory help
	@exit 2

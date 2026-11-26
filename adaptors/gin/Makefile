.DEFAULT_GOAL := help

GO ?= go
GOLANGCI_LINT ?= golangci-lint
GO_LANGUAGE = $(shell $(GO) list -m -f '{{.GoVersion}}' github.com/longxi-go/ioc/adaptors/gin)
GO_SOURCES = $(shell rg --files -g '*.go' -g '**/testdata/**/*.go.tmpl' -g '!vendor/**' -g '!.tmp/**')

.PHONY: help setup format check-format license-headers check-license-headers test race vet lint verify-mod build check
help:
	@printf '%s\n' \
	  'ioc Gin adapter targets:' \
	  '  setup                  Verify pinned golayout tool resolution' \
	  '  format/check-format    Write/check owned Go formatting' \
	  '  license-headers        Refresh owned Go notices' \
	  '  check-license-headers  Check owned Go notices' \
	  '  test/race/vet/lint     Run focused module quality gates' \
	  '  verify-mod             Check dependency tidiness and checksums' \
	  '  build                  Build ginctl into .tmp/bin' \
	  '  check                  Run all module quality gates'

setup:
	$(GO) list -mod=readonly -deps github.com/longxi-go/golayout/cmd/golayout >/dev/null
format:
	$(GO) run -mod=readonly github.com/longxi-go/golayout/cmd/golayout -lang go$(GO_LANGUAGE) -w $(GO_SOURCES)
check-format:
	$(GO) run -mod=readonly github.com/longxi-go/golayout/cmd/golayout -lang go$(GO_LANGUAGE) $(GO_SOURCES)
license-headers:
	$(GO) run -mod=readonly github.com/longxi-go/golayout/cmd/golayout -header codequality/HEADER -w $(filter %.go,$(GO_SOURCES))
check-license-headers:
	$(GO) run -mod=readonly github.com/longxi-go/golayout/cmd/golayout -header codequality/HEADER $(filter %.go,$(GO_SOURCES))
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
build:
	$(GO) build -o .tmp/bin/ginctl ./cmd/ginctl
check: setup check-license-headers check-format verify-mod vet test race lint

.DEFAULT:
	@printf 'Unknown target: %s. Available targets follow.\n' "$@" >&2
	@$(MAKE) --no-print-directory help
	@exit 2

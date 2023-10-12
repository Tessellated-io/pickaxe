#!/usr/bin/make -f

CURRENT_DIR = $(shell pwd)
BUILDDIR ?= $(CURDIR)/build

BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)'
# check for nostrip option
ifeq (,$(findstring nostrip,$(COSMOS_BUILD_OPTIONS)))
  BUILD_FLAGS += -trimpath
endif

# Check for debug option
ifeq (debug,$(findstring debug,$(COSMOS_BUILD_OPTIONS)))
  BUILD_FLAGS += -gcflags "all=-N -l"
endif

###############################################################################
###                                  Build                                  ###
###############################################################################

BUILD_TARGETS := build

build: BUILD_ARGS=

build-linux-amd64:
	GOOS=linux GOARCH=amd64 LEDGER_ENABLED=false $(MAKE) build

build-linux-arm64:
	GOOS=linux GOARCH=arm64 LEDGER_ENABLED=false $(MAKE) build

$(BUILD_TARGETS): go.sum $(BUILDDIR)/
	cd ${CURRENT_DIR} && go $@ -mod=readonly $(BUILD_FLAGS) $(BUILD_ARGS) ./...

$(BUILDDIR)/:
	mkdir -p $(BUILDDIR)/

###############################################################################
###                                  Test                                  ###
###############################################################################

test:
	@go test ./...

###############################################################################
###                          Tools & Dependencies                           ###
###############################################################################

go.sum: go.mod
	@go mod verify
	@go mod tidy


###############################################################################
###                                Linting                                  ###
###############################################################################

golangci_version=v1.53.3

lint-install:
	@echo "--> Installing golangci-lint $(golangci_version)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(golangci_version)

lint:
	@echo "--> Running linter"
	$(MAKE) lint-install
	@golangci-lint run  -c "./.golangci.yml"

lint-fix:
	@echo "--> Running linter"
	$(MAKE) lint-install
	@golangci-lint run  -c "./.golangci.yml" --fix


.PHONY: lint lint-fix
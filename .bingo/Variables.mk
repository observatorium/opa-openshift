# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo v0.4.3. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
BINGO_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO     ?= $(shell which go)

# Below generated variables ensure that every time a tool under each variable is invoked, the correct version
# will be used; reinstalling only if needed.
# For example for dex variable:
#
# In your main Makefile (for non array binaries):
#
#include .bingo/Variables.mk # Assuming -dir was set to .bingo .
#
#command: $(DEX)
#	@echo "Running dex"
#	@$(DEX) <flags/args..>
#
DEX := $(GOBIN)/dex-v0.0.0-20200512115545-709d4169d646
$(DEX): $(BINGO_DIR)/dex.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/dex-v0.0.0-20200512115545-709d4169d646"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=dex.mod -o=$(GOBIN)/dex-v0.0.0-20200512115545-709d4169d646 "github.com/dexidp/dex/cmd/dex"

EMBEDMD := $(GOBIN)/embedmd-v1.0.0
$(EMBEDMD): $(BINGO_DIR)/embedmd.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/embedmd-v1.0.0"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=embedmd.mod -o=$(GOBIN)/embedmd-v1.0.0 "github.com/campoy/embedmd"

GOFUMPT := $(GOBIN)/gofumpt-v0.1.1
$(GOFUMPT): $(BINGO_DIR)/gofumpt.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/gofumpt-v0.1.1"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=gofumpt.mod -o=$(GOBIN)/gofumpt-v0.1.1 "mvdan.cc/gofumpt"

GOLANGCI_LINT := $(GOBIN)/golangci-lint-v1.41.1
$(GOLANGCI_LINT): $(BINGO_DIR)/golangci-lint.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/golangci-lint-v1.41.1"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=golangci-lint.mod -o=$(GOBIN)/golangci-lint-v1.41.1 "github.com/golangci/golangci-lint/cmd/golangci-lint"

LOKI_2_2_1 := $(GOBIN)/loki_2_2_1-v1.6.2-0.20210406003638-babea82ef558
$(LOKI_2_2_1): $(BINGO_DIR)/loki_2_2_1.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/loki_2_2_1-v1.6.2-0.20210406003638-babea82ef558"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=loki_2_2_1.mod -o=$(GOBIN)/loki_2_2_1-v1.6.2-0.20210406003638-babea82ef558 "github.com/grafana/loki/cmd/loki"

UP := $(GOBIN)/up-v0.0.0-20210212114231-03ef2f2bb89b
$(UP): $(BINGO_DIR)/up.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/up-v0.0.0-20210212114231-03ef2f2bb89b"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=up.mod -o=$(GOBIN)/up-v0.0.0-20210212114231-03ef2f2bb89b "github.com/observatorium/up/cmd/up"


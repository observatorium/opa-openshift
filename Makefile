include .bingo/Variables.mk

SHELL=/usr/bin/env bash -o pipefail
TMP_DIR := $(shell pwd)/tmp
BIN_DIR ?= $(TMP_DIR)/bin
LIB_DIR ?= $(TMP_DIR)/lib
LOG_DIR ?= $(TMP_DIR)/logs
CERT_DIR ?= $(TMP_DIR)/certs
FIRST_GOPATH := $(firstword $(subst :, ,$(shell go env GOPATH)))
OS ?= $(shell uname -s | tr '[A-Z]' '[a-z]')
ARCH ?= $(shell uname -m)

VERSION := $(strip $(shell [ -d .git ] && git describe --always --tags --dirty))
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
GIT_REVISION := $(shell git rev-parse --short HEAD)f
BUILD_DATE := $(shell date -u +"%Y-%m-%d")
BUILD_TIMESTAMP := $(shell date -u +"%Y-%m-%dT%H:%M:%S%Z")
VCS_BRANCH := $(strip $(shell git rev-parse --abbrev-ref HEAD))
VCS_REF := $(strip $(shell [ -d .git ] && git rev-parse --short HEAD))
DOCKER_REPO ?= quay.io/observatorium/opa-openshift

OBSERVATORIUM ?= $(BIN_DIR)/api
SHELLCHECK ?= $(BIN_DIR)/shellcheck

GENERATE_TLS_CERT ?= $(BIN_DIR)/tls
SERVER_CERT ?= $(CERT_DIR)/server.pem

ifeq (,$(shell which podman 2>/dev/null))
OCI_BIN ?= docker
else
OCI_BIN ?= podman
endif

default: opa-openshift
all: clean lint test opa-openshift

tmp/help.txt: opa-openshift
	./opa-openshift --help 2>&1 | head -n -1 > tmp/help.txt || true

README.md: $(EMBEDMD) tmp/help.txt
	$(EMBEDMD) -w README.md

.PHONY: deps
deps: go.mod go.sum
	go mod tidy
	go mod download
	go mod verify

opa-openshift: deps main.go $(wildcard *.go) $(wildcard */*.go)
	CGO_ENABLED=0 GOOS=$(OS) GOARCH=amd64 GO111MODULE=on GOPROXY=https://proxy.golang.org go build -a -ldflags '-s -w -X main.Version=$(VERSION) -X main.Branch=$(GIT_BRANCH) -X main.Revision=$(GIT_REVISION)' -o $@ .

.PHONY: go-generate
go-generate:
	go generate ./...

.PHONY: build
build: opa-openshift

.PHONY: format
format: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run --fix --enable-all -c .golangci.yml

.PHONY: go-fmt
go-fmt: $(GOFUMPT)
	@fmt_res=$$($(GOFUMPT) -l -w $$(find . -type f -name '*.go' -not -path './internal/external/k8s/k8sfakes/*' -not -path './internal/external/ocp/ocpfakes/*' -not -path '${TMP_DIR}/*')); if [ -n "$$fmt_res" ]; then printf '\nGofmt found style issues. Please check the reported issues\nand fix them if necessary before submitting the code for review:\n\n%s' "$$fmt_res"; exit 1; fi

.PHONY: shellcheck
shellcheck: $(SHELLCHECK)
	$(SHELLCHECK) $(shell find . -type f -name "*.sh" -not -path "*vendor*" -not -path "${TMP_DIR}/*")

.PHONY: lint
lint: $(GOLANGCI_LINT) go-fmt shellcheck
	$(GOLANGCI_LINT) run -c .golangci.yml

.PHONY: test
test: build test-unit test-integration

.PHONY: test-unit
test-unit: go-generate
	CGO_ENABLED=1 GO111MODULE=on go test -v -race -short ./...

.PHONY: test-integration
test-integration: build integration-test-dependencies generate-cert
	PATH=$(BIN_DIR):$(FIRST_GOPATH)/bin:$$PATH LD_LIBRARY_PATH=$$LD_LIBRARY_PATH:$(LIB_DIR) LOG_DIR=$(LOG_DIR) ./test/integration.sh

.PHONY: clean
clean:
	-rm tmp/help.txt
	-rm -rf tmp/bin
	-rm -rf tmp/certs
	-rm -rf tmp/logs
	-rm -rf wal
	-rm opa-openshift

.PHONY: container
container: Dockerfile
	@$(OCI_BIN) build --build-arg BUILD_DATE="$(BUILD_TIMESTAMP)" \
		--build-arg VERSION="$(VERSION)" \
		--build-arg VCS_REF="$(VCS_REF)" \
		--build-arg VCS_BRANCH="$(VCS_BRANCH)" \
		--build-arg DOCKERFILE_PATH="/Dockerfile" \
		-t $(DOCKER_REPO):$(VCS_BRANCH)-$(BUILD_DATE)-$(VERSION) .
	@$(OCI_BIN) tag $(DOCKER_REPO):$(VCS_BRANCH)-$(BUILD_DATE)-$(VERSION) $(DOCKER_REPO):latest

.PHONY: container-push
container-push: container
	$(OCI_BIN) push $(DOCKER_REPO):$(VCS_BRANCH)-$(BUILD_DATE)-$(VERSION)
	$(OCI_BIN) push $(DOCKER_REPO):latest

.PHONY: container-release
container-release: VERSION_TAG = $(strip $(shell [ -d .git ] && git tag --points-at HEAD))
container-release: container
	# https://git-scm.com/docs/git-tag#Documentation/git-tag.txt---points-atltobjectgt
	@$(OCI_BIN) tag $(DOCKER_REPO):$(VCS_BRANCH)-$(BUILD_DATE)-$(VERSION) $(DOCKER_REPO):$(VERSION_TAG)
	$(OCI_BIN) push $(DOCKER_REPO):$(VERSION_TAG)
	$(OCI_BIN) push $(DOCKER_REPO):latest

.PHONY: integration-test-dependencies
integration-test-dependencies: $(LOG_DIR) $(DEX) $(LOKI) $(UP) $(OBSERVATORIUM)

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

$(CERT_DIR):
	mkdir -p $(CERT_DIR)

$(LIB_DIR):
	mkdir -p $@

$(LOG_DIR):
	mkdir -p $@

$(OBSERVATORIUM): | $(BIN_DIR)
	GOBIN=$(BIN_DIR) go install github.com/observatorium/api@main

# A thin wrapper around github.com/cloudflare/cfssl
$(GENERATE_TLS_CERT): | $(BIN_DIR)
	GOBIN=$(BIN_DIR) go install -tags tools github.com/observatorium/api/test/tls@2458175c6c2043fd5093da3e6b7ad623979d21c0

$(SERVER_CERT): | $(GENERATE_TLS_CERT) $(CERT_DIR)
	cd $(CERT_DIR) && $(GENERATE_TLS_CERT)

# Generate TLS certificates for local development.
generate-cert: $(SERVER_CERT) | $(GENERATE_TLS_CERT)

$(SHELLCHECK): | $(BIN_DIR)
	curl -sNL "https://github.com/koalaman/shellcheck/releases/download/stable/shellcheck-stable.$(OS).$(ARCH).tar.xz" | tar --strip-components=1 -xJf - -C $(BIN_DIR)

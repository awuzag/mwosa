GO ?= go
BIN_DIR ?= bin
BINARY ?= mwosa
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
DEV_DIR ?= .mwosa
DEV_CONFIG_PATH ?= $(DEV_DIR)/config.json
DEV_DATABASE_PATH ?= $(DEV_DIR)/mwosa.db

CMD_PKG := ./cmd/mwosa
CONFIG_PKG := github.com/ev3rlit/mwosa/app/config
BIN_PATH := $(BIN_DIR)/$(BINARY)
BASE_LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)
DEV_LDFLAGS := $(BASE_LDFLAGS) -X $(CONFIG_PKG).defaultConfigPath=$(DEV_CONFIG_PATH) -X $(CONFIG_PKG).defaultDatabasePath=$(DEV_DATABASE_PATH)
CLIENT_MODULES := \
	clients/datago-corpfin \
	clients/datago-etp \
	clients/datago-krxlisted \
	clients/datago-stock-price \
	clients/kis \
	clients/krx
LOAD_DOTENV := set -a; [ ! -f .env ] || . ./.env; set +a;

.PHONY: help build build-release install run fmt-check test test-clients test-e2e-krx-client test-e2e-krx-cli test-e2e-krx pre-commit install-hooks verify clean

help:
	@printf "%s\n" "mwosa make targets"
	@printf "%s\n" "  make build         Build $(BIN_PATH) with project-local dev paths"
	@printf "%s\n" "  make build-release Build with OS default paths and GOWORK=off"
	@printf "%s\n" "  make install       Install mwosa with go install"
	@printf "%s\n" "  make run ARGS='...' Run mwosa from source"
	@printf "%s\n" "  make fmt-check     Check Go formatting"
	@printf "%s\n" "  make test          Run root module tests"
	@printf "%s\n" "  make test-clients  Run provider client module tests"
	@printf "%s\n" "  make test-e2e-krx-client Run opt-in live KRX client e2e tests"
	@printf "%s\n" "  make test-e2e-krx-cli    Run opt-in live KRX CLI e2e tests"
	@printf "%s\n" "  make test-e2e-krx        Run all opt-in live KRX e2e tests"
	@printf "%s\n" "  make pre-commit    Run local pre-commit checks"
	@printf "%s\n" "  make install-hooks Install repo-managed git hooks"
	@printf "%s\n" "  make verify        Run all repo checks"
	@printf "%s\n" "  make clean         Remove build outputs"

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -ldflags "$(DEV_LDFLAGS)" -o $(BIN_PATH) $(CMD_PKG)

build-release:
	@mkdir -p $(BIN_DIR)
	GOWORK=off $(GO) build -ldflags "$(BASE_LDFLAGS)" -o $(BIN_PATH) $(CMD_PKG)

install:
	$(GO) install -ldflags "$(BASE_LDFLAGS)" $(CMD_PKG)

run:
	$(GO) run -ldflags "$(DEV_LDFLAGS)" $(CMD_PKG) $(ARGS)

fmt-check:
	@files="$$(git ls-files '*.go')"; \
	if [ -n "$$files" ]; then \
		unformatted="$$(gofmt -l $$files)"; \
		if [ -n "$$unformatted" ]; then \
			printf "%s\n" "gofmt required:"; \
			printf "%s\n" "$$unformatted"; \
			exit 1; \
		fi; \
	fi

test:
	$(GO) test ./...

test-clients:
	@for module in $(CLIENT_MODULES); do \
		printf "%s\n" "==> $$module"; \
		(cd "$$module" && $(GO) test ./... && $(GO) mod verify) || exit $$?; \
	done

test-e2e-krx-client:
	@$(LOAD_DOTENV) if [ -z "$$MWOSA_KRX_AUTH_KEY" ]; then printf "%s\n" "MWOSA_KRX_AUTH_KEY is not set; live KRX client e2e tests will skip"; fi
	$(LOAD_DOTENV) (cd clients/krx && KRX_E2E=1 $(GO) test -tags=e2e -count=1 ./...)

test-e2e-krx-cli:
	@$(LOAD_DOTENV) if [ -z "$$MWOSA_KRX_AUTH_KEY" ]; then printf "%s\n" "MWOSA_KRX_AUTH_KEY is not set; live KRX CLI e2e tests will skip"; fi
	$(LOAD_DOTENV) KRX_E2E=1 $(GO) test -tags=e2e -count=1 ./testing/e2e

test-e2e-krx: test-e2e-krx-client test-e2e-krx-cli

pre-commit:
	scripts/check/pre-commit.sh

install-hooks:
	git config core.hooksPath .githooks
	@printf "%s\n" "installed git hooks from .githooks"

verify: pre-commit

clean:
	rm -rf $(BIN_DIR)

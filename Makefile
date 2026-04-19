GO ?= $(or $(shell command -v go 2>/dev/null),/home/node/.local/go/bin/go)
VP ?= vp
ADDR ?= 127.0.0.1:8080
APP_VERSION ?= v.0.1.0

ROOT_DIR := $(CURDIR)
WEB_DIR := $(ROOT_DIR)/web
EMBED_DIR := $(ROOT_DIR)/cmd/studio/web/dist
OUT_BIN := $(ROOT_DIR)/out/confdiff-studio

.PHONY: help web-install web-generate sync-web-dist run build test-go check-web check-web-fix

help:
	@echo "Targets:"
	@echo "  make run            # web generate -> embed sync -> go run"
	@echo "  make build          # web generate -> embed sync -> go build"
	@echo "  make sync-web-dist  # sync web/.output/public to embedded dir"
	@echo "  make web-install    # install web dependencies"
	@echo "  make web-generate   # generate static web assets"
	@echo "  make test-go        # run go test ./..."
	@echo "  make check-web      # run web check"
	@echo "  make check-web-fix  # run web check --fix"

web-install:
	cd "$(WEB_DIR)" && $(VP) install

web-generate:
	cd "$(WEB_DIR)" && NUXT_PUBLIC_APP_VERSION="$(APP_VERSION)" $(VP) run generate

sync-web-dist: web-generate
	mkdir -p "$(EMBED_DIR)"
	rsync -a --delete "$(WEB_DIR)/.output/public/" "$(EMBED_DIR)/"

run: sync-web-dist
	@PORT="$${ADDR##*:}"; \
	if command -v ss >/dev/null 2>&1 && ss -ltn | awk 'NR > 1 {print $$4}' | grep -Eq '(^|:)'"$$PORT"'$$'; then \
		echo "Address $(ADDR) is already in use."; \
		echo "Stop the existing studio process or run: make run ADDR=127.0.0.1:18081"; \
		exit 1; \
	fi
	$(GO) run ./cmd/studio -addr $(ADDR)

build: sync-web-dist
	mkdir -p "$(ROOT_DIR)/out"
	$(GO) build -o "$(OUT_BIN)" ./cmd/studio

test-go:
	$(GO) test ./...

check-web:
	cd "$(WEB_DIR)" && $(VP) check

check-web-fix:
	cd "$(WEB_DIR)" && $(VP) check --fix

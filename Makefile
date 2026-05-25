BINARY     ?= stacklit
BUILD_DIR  ?= build
CMD        := ./cmd/stacklit
GO         ?= go
INSTALL_DIR ?= $(HOME)/.local/bin
SOURCE_REF ?= local
SOURCE_REVISION ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
VERSION    ?= source:$(SOURCE_REF):$(SOURCE_REVISION)
LDFLAGS    := -ldflags "-X github.com/glincker/stacklit/internal/cli.Version=$(VERSION)"

.PHONY: install build run test clean

install: build
	@dest="$(INSTALL_DIR)/$(BINARY)"; \
	mkdir -p "$(INSTALL_DIR)" || { printf 'INSTALL_DIR is not usable: %s\n' "$(INSTALL_DIR)" >&2; exit 1; }; \
	cp "$(BUILD_DIR)/$(BINARY)" "$$dest" || { printf 'INSTALL_DIR is not usable: %s\n' "$(INSTALL_DIR)" >&2; exit 1; }; \
	chmod 0755 "$$dest" || { printf 'INSTALL_DIR is not usable: %s\n' "$(INSTALL_DIR)" >&2; exit 1; }; \
	if [ ! -x "$$dest" ]; then \
		printf 'installed stacklit is not executable: %s\n' "$$dest" >&2; \
		exit 1; \
	fi; \
	if ! "$$dest" --version >/dev/null 2>&1; then \
		printf 'installed stacklit at %s failed --version\n' "$$dest" >&2; \
		exit 1; \
	fi; \
	printf 'Installed stacklit %s to %s\n' "$(VERSION)" "$$dest"

build:
	@mkdir -p "$(BUILD_DIR)"
	$(GO) build $(LDFLAGS) -o "$(BUILD_DIR)/$(BINARY)" $(CMD)

run: build
	./$(BUILD_DIR)/$(BINARY)

test:
	go test ./...

clean:
	rm -rf "$(BUILD_DIR)" dist/

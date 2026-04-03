MODULE   := github.com/medincident/medincident-zitadel-actions
ENTRY    := ./cmd/server
BIN_NAME := server
DIST     := ./dist

GOOS   ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# ── Platforms ────────────────────────────────────────────────────────────────
# Edit the list below to add or remove build targets.
PLATFORMS := \
	linux/amd64   \
	linux/arm64   \
	linux/386     \
	darwin/amd64  \
	darwin/arm64  \
	windows/amd64 \
	windows/arm64 \
	windows/386

# ── Helpers ──────────────────────────────────────────────────────────────────
bin_name = $(BIN_NAME)-$(1)-$(2)$(if $(filter windows,$(1)),.exe)

# ── Targets ──────────────────────────────────────────────────────────────────

.PHONY: build build-all run clean

## build: compile for the current host platform
build:
	@mkdir -p $(DIST)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(DIST)/$(call bin_name,$(GOOS),$(GOARCH)) $(ENTRY)

## build-all: cross-compile for every platform in PLATFORMS
build-all:
	@mkdir -p $(DIST)
	$(foreach p,$(PLATFORMS), \
		$(eval os   = $(word 1,$(subst /, ,$(p)))) \
		$(eval arch = $(word 2,$(subst /, ,$(p)))) \
		GOOS=$(os) GOARCH=$(arch) go build -o $(DIST)/$(call bin_name,$(os),$(arch)) $(ENTRY) ; \
	)

## run: build and run the server locally
run: build
	$(DIST)/$(call bin_name,$(GOOS),$(GOARCH)) $(ARGS)

## clean: remove build artifacts
clean:
	rm -rf $(DIST)

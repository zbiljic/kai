XDG_CACHE_HOME ?= $(HOME)/.cache
TOOLS_BIN_DIR := $(if $(PROJECT_NAME),$(abspath $(XDG_CACHE_HOME)/$(PROJECT_NAME)/bin),$(error ERROR: Project name missing))

GOFUMPT := $(TOOLS_BIN_DIR)/gofumpt
GOLANGCI_LINT := $(TOOLS_BIN_DIR)/golangci-lint
GORELEASER := $(shell which goreleaser)
ifndef GORELEASER
override GORELEASER = $(TOOLS_BIN_DIR)/goreleaser
endif
GOTESTSUM := $(TOOLS_BIN_DIR)/gotestsum

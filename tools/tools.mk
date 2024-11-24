include tools/common.mk
include tools/versions.mk

$(TOOLS_BIN_DIR):
	@mkdir -p $@

$(GOFUMPT): $(TOOLS_BIN_DIR)
	@GOBIN=$(TOOLS_BIN_DIR) go install mvdan.cc/gofumpt@$(GOFUMPT_VERSION)

$(GOLANGCI_LINT): $(TOOLS_BIN_DIR)
	@command -v $(GOLANGCI_LINT) >/dev/null 2>&1 || ( \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(TOOLS_BIN_DIR) $(GOLANGCI_LINT_VERSION) \
		)

$(GORELEASER): $(TOOLS_BIN_DIR)
	@command -v $(GORELEASER) >/dev/null 2>&1 || ( \
		curl -sfLo $(GORELEASER).tar.gz \
			"https://github.com/goreleaser/goreleaser/releases/download/${GORELEASER_VERSION}/goreleaser_$(shell uname -s)_$(shell uname -m | sed 's/aarch64/arm64/').tar.gz" \
			&& tar -C $(TOOLS_BIN_DIR) -xf $(GORELEASER).tar.gz goreleaser \
			&& rm $(GORELEASER).tar.gz \
		)

$(GOTESTSUM): $(TOOLS_BIN_DIR)
	@GOBIN=$(TOOLS_BIN_DIR) go install gotest.tools/gotestsum@$(GOTESTSUM_VERSION)

.PHONY: tools
tools: deps lint-deps build-deps

.PHONY: deps
deps: $(GOTESTSUM)

.PHONY: lint-deps
lint-deps: $(GOFUMPT)
lint-deps: $(GOLANGCI_LINT)

.PHONY: build-deps
build-deps: $(GORELEASER)

.PHONY: clean-tools
clean-tools:
	@rm -rf $(TOOLS_BIN_DIR)

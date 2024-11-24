SHELL = bash
PROJECT_ROOT := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

# Using directory as project name.
PROJECT_NAME := $(shell basename $(PROJECT_ROOT))
PROJECT_MODULE := $(shell go list -m)

default: help

ifeq ($(CI),true)
$(info Running in a CI environment, verbose mode is disabled)
else
VERBOSE="true"
endif

include tools/tools.mk

# include per-user customization after all variables are defined
-include Makefile.local

HELP_FORMAT="    \033[36m%-20s\033[0m %s\n"
.PHONY: help
help: ## Display this usage information
	@echo "Valid targets:"
	@{ \
		echo $(MAKEFILE_LIST) \
			| xargs grep -E '^[^ \$$]+:.*?## .*$$' -h \
		; \
		echo $(MAKEFILE_LIST) \
			| xargs cat 2> /dev/null \
			| sed -e 's/$\(eval/$\(info/' \
			| make -f- 2> /dev/null \
			| grep -E '^[^ ]+:.*?## .*$$' -h \
		; \
	} \
		| sort \
		| awk 'BEGIN {FS = ":.*?## "}; \
			{printf $(HELP_FORMAT), $$1, $$2}'
	@echo ""

# Only for CI compliance
.PHONY: bootstrap
bootstrap: tools # Install all dependencies

GO_VERSION := $(shell go mod edit -json | sed -En 's/"Go": "([^"]*).*/\1/p' | tr -d '[:blank:]')

GO_MOD_TIDY_CMD   := go mod tidy -compat=$(GO_VERSION)
GO_MOD_TIDY_E_CMD := go mod tidy -e -compat=$(GO_VERSION)

.PHONY: go-mod-tidy
go-mod-tidy:
	@cd $(PROJECT_ROOT) && $(GO_MOD_TIDY_E_CMD) && $(GO_MOD_TIDY_CMD)

.PHONY: tidy
tidy: go-mod-tidy

.PHONY: gofmt
gofmt: lint-deps
gofmt: ## Format Go code
	@$(GOFUMPT) -extra -l -w .

.PHONY: lint
lint: lint-deps
lint: ## Lint the source code
	@echo "==> Linting source code..."
	@$(GOLANGCI_LINT) run --config=.golangci.yml --fix

	@echo "==> Checking Go mod..."
	@$(MAKE) tidy
	@if (git status --porcelain | grep -Eq "go\.(mod|sum)"); then \
		echo go.mod or go.sum needs updating; \
		git --no-pager diff go.mod; \
		git --no-pager diff go.sum; \
		exit 1; fi

.PHONY: test
test: $(GOTESTSUM)
test: ## Run the test suite and/or any other tests
	CGO_ENABLED=1 $(if $(ENABLE_RACE),GORACE="strip_path_prefix=$(GOPATH)/src") \
		$(GOTESTSUM) \
		--junitfile=junit.unit.xml \
		--junitfile-project-name=$(PROJECT_NAME) \
		-- \
		$(if $(ENABLE_RACE),-race) $(if $(VERBOSE),-v) \
		-cover \
		-coverprofile=unit.cover \
		$(if $(ENABLE_RACE),-covermode=atomic,-covermode=count) \
		-timeout=15m \
		./...

.PHONY: coverage
coverage: ## Open a web browser displaying coverage
	go tool cover -html=unit.cover

.PHONY: compile
compile: # Compiles the packages but discards the resulting object, serving only as a check that the packages can be built
	CGO_ENABLED=0 go build -o /dev/null ./...

.PHONY: install
install: install-$(PROJECT_NAME)
install: ## Compile and install the main packages

.PHONY: install-$(PROJECT_NAME)
install-$(PROJECT_NAME):
	@if [ -x "$$(command -v $(PROJECT_NAME))" ]; then \
		echo "$(PROJECT_NAME) is already installed, do you want to re-install it? [y/N] " && read ans; \
			if [ "$$ans" = "y" ] || [ "$$ans" = "Y" ]  ; then \
				go install .; \
			else \
				echo "aborting install"; \
			exit -1; \
		fi; \
	else \
		go install .; \
	fi;

.PHONY: package
package: $(GORELEASER)
	$(GORELEASER) release --config=.goreleaser.yaml --snapshot --skip=publish --clean

.PHONY: release
release: $(GORELEASER)
	$(GORELEASER) release --config=.goreleaser.yaml --clean

.PHONY: nightly
nightly: $(GORELEASER)
	@if [ ! -z $${GORELEASER_CURRENT_TAG+x} ]; then \
		git tag $(GORELEASER_CURRENT_TAG); \
		$(MAKE) release; \
		git tag -d $(GORELEASER_CURRENT_TAG); \
	else \
		echo "missing nightly build tag"; \
		exit -1; \
	fi;

.PHONY: build
build: lint test package

.PHONY: pre-commit
pre-commit: lint test

.PHONY: clean
clean: ## Remove build artifacts
	@echo "==> Removing build artifacts..."
	@rm -f $(if $(VERBOSE),-v) *.cover coverage.xml junit.*.xml
	@rm -f $(if $(VERBOSE),-v) "$(GOPATH)/bin/$(PROJECT_NAME)"
	@rm -rf $(if $(VERBOSE),-v) "$(PROJECT_ROOT)/dist/"

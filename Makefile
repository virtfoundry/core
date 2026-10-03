NPM ?= npm
UI  := ui

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show available targets.
	@awk 'BEGIN {FS = ":.*##"; printf "Targets:\n"} \
	/^[a-zA-Z_-]+:.*##/ {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' \
	$(MAKEFILE_LIST)

.PHONY: deps
deps: ## Install UI dependencies (npm ci in ui/).
	cd $(UI) && $(NPM) ci

.PHONY: test
test: ## Run UI tests (vitest).
	cd $(UI) && $(NPM) test

.PHONY: build
build: deps ## Build UI production bundle.
	cd $(UI) && $(NPM) run build

.PHONY: ci
ci: deps test build ## Mirror what CI does: install, test, build.

.PHONY: clean
clean: ## Remove UI build/coverage artifacts.
	rm -rf $(UI)/dist $(UI)/coverage
.PHONY: help build-index update-graph test-fbc

# Default target
help:  ## Show this help
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build-index: ## Append a bundle to the FBC catalogs: SNAPSHOT=<snapshot-or-url> [TARGETS=<targets>] [OS=rhel9|rhel10]
	@if [ -z "$(SNAPSHOT)" ]; then \
		echo "Error: SNAPSHOT variable not set. Use: make build-index SNAPSHOT=<snapshot-or-url> [TARGETS=<targets>] [OS=rhel9|rhel10]"; \
		exit 1; \
		fi
	FBC_TARGETS="$(TARGETS)" FBC_OS="$(OS)" ./hack/build-indexs.sh $(SNAPSHOT)

update-graph: ## Add a version to the FBC channel graphs: VERSION=<version> [TARGETS=<targets>] [OS=rhel9|rhel10]
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION variable not set. Use: make update-graph VERSION=<version> [TARGETS=<targets>] [OS=rhel9|rhel10]"; \
		exit 1; \
		fi
	FBC_TARGETS="$(TARGETS)" FBC_OS="$(OS)" ./hack/update-graph.sh $(VERSION)

test-fbc: ## Run the FBC catalog-update helper tests
	./hack/tests/test-fbc-helpers.sh

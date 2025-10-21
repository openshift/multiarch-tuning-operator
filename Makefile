.PHONY: help build-index update-graph

# Default target
help:  ## Show this help
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS=":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build-index: ## Run build-indexs.sh with SNAPSHOT=<snapshot-or-url>
	@if [ -z "$(SNAPSHOT)" ]; then \
		echo "Error: SNAPSHOT variable not set. Use: make build-index SNAPSHOT=<snapshot-or-url>"; \
		exit 1; \
	fi
	./hack/build-indexs.sh $(SNAPSHOT)

update-graph: ## Run update-graph.sh with VERSION=<version>
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION variable not set. Use: make update-graph VERSION=<version>"; \
		exit 1; \
	fi
	./hack/update-graph.sh $(VERSION)

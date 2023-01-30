all: help

clean: ## Clean artifacts
	rm -rf target

build: ## Compile the app
	mkdir -p target
	go build -o target/clae .

run: build ## Run the app locally
	TOKEN=my-dirty-secret LISTEN=":8081" ./target/clae

docker: ## Build Docker container
	docker build -t ghcr.io/nivenly/clae .

.PHONY: help
help:  ## Show help messages for make targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(firstword $(MAKEFILE_LIST)) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[32m%-30s\033[0m %s\n", $$1, $$2}'

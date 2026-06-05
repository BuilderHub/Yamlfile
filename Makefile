# yamlfile — BuildKit frontend Makefile
# Follows BuilderHub patterns (see root Makefile, BUILD.md, AGENTS.md)

REGISTRY ?= ghcr.io/builderhub
TAG ?= dev
IMAGE_NAME ?= $(REGISTRY)/yamlfile:$(TAG)

.PHONY: help build test lint vet revive ci docs docs-serve docker-build docker-build-multiarch docker-push clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the yamlfile-frontend binary (linux/amd64 for image)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/yamlfile-frontend ./cmd/yamlfile-frontend

test: ## Run unit tests
	go test ./... -count=1 -race

lint: ## Run golangci-lint
	golangci-lint run ./...

vet: ## Run go vet
	go vet ./...

revive: ## Run revive linter
	revive -formatter stylish ./...

ci: test lint vet revive ## Run CI checks: tests, lint, revive, vet (etc.)

DOCS_DIR ?= docs

docs: ## Build Hugo documentation site (output: docs/public)
	hugo -s $(DOCS_DIR) --gc --minify

docs-serve: ## Serve Hugo docs locally with live reload (http://localhost:1313)
	hugo server -s $(DOCS_DIR) -D --disableFastRender --noHTTPCache

docker-build: ## Build yamlfile frontend image (current arch) using buildx
	# Note: context must be monorepo root (for go replace ../buildkit-hive + ./yamlfile paths in Dockerfile)
	docker buildx build \
		-f cmd/yamlfile-frontend/Dockerfile \
		-t $(IMAGE_NAME) \
		--load \
		..

docker-build-multiarch: ## Build multi-arch image (push required for manifest)
	docker buildx build \
		-f cmd/yamlfile-frontend/Dockerfile \
		-t $(IMAGE_NAME) \
		--platform linux/amd64,linux/arm64 \
		--push \
		..

docker-push: docker-build ## Push current-arch image (use build-multiarch for real multi)
	docker push $(IMAGE_NAME)

clean: ## Clean build artifacts
	rm -rf bin/ /tmp/yfout /tmp/yftest

# For root orchestration (see root Makefile)
docker-build-yamlfile: docker-build
docker-build-yamlfile-multiarch: docker-build-multiarch
docker-push-yamlfile: docker-push
clean-yamlfile: clean

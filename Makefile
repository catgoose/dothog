# Default target
.DEFAULT_GOAL := help

MAGE := go tool mage
IMAGE := ghcr.io/catgoose/dothog
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# .PHONY targets
.PHONY: watch docker-build docker-login docker-push docker-release

# Watch mode
watch:
	OPEN_BROWSER=false $(MAGE) watch

# Build Docker image with the current version
docker-build:
	docker build --build-arg APP_VERSION=$(VERSION) -t $(IMAGE):$(VERSION) -t $(IMAGE):latest .

# Log in to GHCR using gh CLI token
docker-login:
	gh auth token | docker login ghcr.io -u $(shell gh api user --jq .login) --password-stdin

# Push Docker image to GHCR
docker-push: docker-login
	docker push $(IMAGE):$(VERSION)
	docker push $(IMAGE):latest

# Build and push in one step
docker-release: docker-build docker-push

# Help target to display available targets and their descriptions
help:
	@echo "Available targets:"
	@echo ""
	@echo "  watch              - Run templ, air, and tailwind in watch mode without opening browser"
	@echo "  docker-build       - Build Docker image (VERSION=$(VERSION))"
	@echo "  docker-login       - Log in to GHCR via gh CLI"
	@echo "  docker-push        - Push Docker image to GHCR (auto-logins)"
	@echo "  docker-release     - Build and push Docker image"
	@echo ""
	@echo "  Override version:    make docker-release VERSION=v0.0.29"

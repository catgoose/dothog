# Default target
.DEFAULT_GOAL := help

MAGE := go tool mage
IMAGE := ghcr.io/catgoose/dothog
SHA := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)

# .PHONY targets
.PHONY: watch docker-build docker-login docker-push docker-release help

# Watch mode
watch:
	OPEN_BROWSER=false $(MAGE) watch

# Build Docker image tagged with commit SHA and latest
docker-build:
	docker build --build-arg APP_VERSION=$(SHA) -t $(IMAGE):$(SHA) -t $(IMAGE):latest .

# Log in to GHCR using gh CLI token
docker-login:
	gh auth token | docker login ghcr.io -u $(shell gh api user --jq .login) --password-stdin

# Push Docker image to GHCR
docker-push: docker-login
	docker push $(IMAGE):$(SHA)
	docker push $(IMAGE):latest

# Build and push in one step (versioned tags are created by CI)
docker-release: docker-build docker-push

# Help target to display available targets and their descriptions
help:
	@echo "Available targets:"
	@echo ""
	@echo "  watch              - Run templ, air, and tailwind in watch mode without opening browser"
	@echo "  docker-build       - Build Docker image (SHA=$(SHA))"
	@echo "  docker-login       - Log in to GHCR via gh CLI"
	@echo "  docker-push        - Push Docker image to GHCR (auto-logins)"
	@echo "  docker-release     - Build and push Docker image"
	@echo ""
	@echo "  Semver tags are managed by CI on merge to main."

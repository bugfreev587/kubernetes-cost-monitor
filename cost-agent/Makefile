### ============================================================
### Variables (edit these)
### ============================================================
GH_USER        ?= bugfreev587
IMAGE_NAME     ?= cost-agent
# Generate build metadata once and reuse (prevents timestamp mismatch)
BUILD_DATE     := $(shell date +%Y%m%d)
BUILD_TIME     := $(shell date +%H%M%S)
GIT_COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Auto-detect platform and architecture based on host
UNAME_M := $(shell uname -m)
ifeq ($(UNAME_M),arm64)
	DEFAULT_PLATFORM := linux/arm64
	DEFAULT_ARCH := arm64
else ifeq ($(UNAME_M),aarch64)
	DEFAULT_PLATFORM := linux/arm64
	DEFAULT_ARCH := arm64
else
	DEFAULT_PLATFORM := linux/amd64
	DEFAULT_ARCH := amd64
endif

# Allow override of platform via environment variable
PLATFORM ?= $(DEFAULT_PLATFORM)

# Extract ARCH from PLATFORM if PLATFORM is explicitly set, otherwise use default
ifdef PLATFORM
	# Extract architecture from platform (e.g., linux/amd64 -> amd64)
	ARCH := $(lastword $(subst /, ,$(PLATFORM)))
else
	ARCH := $(DEFAULT_ARCH)
endif

# Compose unique image tag: cost-agent-<arch>-<date>-<time>-<commit>
IMAGE_TAG      ?= cost-agent-$(ARCH)-$(BUILD_DATE)-$(BUILD_TIME)-$(GIT_COMMIT)
REGISTRY       ?= ghcr.io/$(GH_USER)
FULL_IMAGE     ?= $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

DOCKERFILE	   ?= Dockerfile

API_SERVER_URL ?= http://host.docker.internal:8080

### ============================================================
### help
### ============================================================

.PHONY: help build push login release run run-detached deploy undeploy info
help:
	@echo "Available targets:"
	@echo "  build           - Build the agent Docker image"
	@echo "  push            - Push the agent Docker image to GHCR"
	@echo "  login           - Login to GitHub Container Registry"
	@echo "  release         - Build and push the agent Docker image"
	@echo "  run             - Run the agent Docker container"
	@echo "  run-detached    - Run the agent Docker container in detached mode"
	@echo "  deploy          - Deploy the agent to Kubernetes cluster"
	@echo "  undeploy        - Remove the agent deployment from Kubernetes cluster"
	@echo "  info            - Show image and configuration info"

### ============================================================
### Build & Push
### ============================================================

# Build the agent image
build:
	@echo "Building for platform: $(PLATFORM)"
	@echo "Architecture: $(ARCH)"
	@echo "Image tag: $(IMAGE_TAG)"
	@echo "Full image: $(FULL_IMAGE)"
	docker build --platform=$(PLATFORM) -f $(DOCKERFILE) -t $(FULL_IMAGE) .

# Login to GitHub Container Registry
login:
	@echo "Logging in to GHCR..."
	echo $(GHCR_TOKEN) | docker login ghcr.io -u $(GH_USER) --password-stdin

# Push image to GHCR
push: login
	@echo "Pushing image: $(FULL_IMAGE)"
	@echo "Image tag: $(IMAGE_TAG)"
	docker push $(FULL_IMAGE)

# Build & Push in one step
# Note: Both build and push use the same IMAGE_TAG generated at Makefile parse time
release: build push
	@echo "Successfully released: $(FULL_IMAGE)"

### ============================================================
### Local Run
### ============================================================

# Run the agent container and connect it to the API server
run:
	docker run --rm \
		-e API_SERVER_URL=$(API_SERVER_URL) \
		$(FULL_IMAGE)

# Run in background mode
run-detached:
	docker run -d \
		-e API_SERVER_URL=$(API_SERVER_URL) \
		--name k8s-agent \
		$(FULL_IMAGE)

### ============================================================
### Kubernetes Deployment
### ============================================================

# Apply k8s deployment manifest
deploy:
	kubectl apply -f deploy/agent.yaml

# Delete deployment
undeploy:
	kubectl delete -f deploy/agent.yaml

### ============================================================
### Utility
### ============================================================

# Show image name being used
info:
	@echo "Image: $(FULL_IMAGE)"
	@echo "Image Tag: $(IMAGE_TAG)"
	@echo "Platform: $(PLATFORM)"
	@echo "Architecture: $(ARCH)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "API Server URL: $(API_SERVER_URL)"
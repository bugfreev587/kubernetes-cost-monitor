### ============================================================
### Variables (edit these)
### ============================================================
GH_USER        ?= bugfreev587
CHART_NAME     := helm-cost-agent
CHART_DIR      := .

# Read version from Chart.yaml
CHART_VERSION  := $(shell grep '^version:' Chart.yaml | awk '{print $$2}')
APP_VERSION    := $(shell grep '^appVersion:' Chart.yaml | awk '{print $$2}' | tr -d '"')

# Generate build metadata
BUILD_DATE     := $(shell date +%Y%m%d)
GIT_COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Registry settings
REGISTRY       := oci://ghcr.io/$(GH_USER)

### ============================================================
### help
### ============================================================

.PHONY: help package push login release clean lint template info

help:
	@echo "Available targets:"
	@echo "  package         - Package the Helm chart"
	@echo "  push            - Push the Helm chart to GHCR"
	@echo "  login           - Login to GitHub Container Registry"
	@echo "  release         - Package and push the Helm chart"
	@echo "  clean           - Remove packaged chart files"
	@echo "  lint            - Lint the Helm chart"
	@echo "  template        - Render chart templates locally"
	@echo "  info            - Show chart and configuration info"

### ============================================================
### Build & Push
### ============================================================

# Lint the chart
lint:
	@echo "Linting Helm chart..."
	helm lint $(CHART_DIR)

# Package the chart
package: lint
	@echo "Packaging Helm chart..."
	@echo "Chart: $(CHART_NAME)"
	@echo "Version: $(CHART_VERSION)"
	helm package $(CHART_DIR)
	@echo "Package created: $(CHART_NAME)-$(CHART_VERSION).tgz"

# Login to GitHub Container Registry
login:
	@echo "Logging in to GHCR..."
	echo $(GHCR_TOKEN) | helm registry login ghcr.io -u $(GH_USER) --password-stdin

# Push chart to GHCR
push: login
	@echo "Pushing chart to: $(REGISTRY)/$(CHART_NAME):$(CHART_VERSION)"
	helm push $(CHART_NAME)-$(CHART_VERSION).tgz $(REGISTRY)

# Package & Push in one step
release: package push
	@echo "Successfully released: $(REGISTRY)/$(CHART_NAME):$(CHART_VERSION)"

### ============================================================
### Utility
### ============================================================

# Render templates locally for debugging
template:
	@echo "Rendering templates..."
	helm template $(CHART_NAME) $(CHART_DIR)

# Clean up packaged files
clean:
	@echo "Cleaning up packaged chart files..."
	rm -f $(CHART_NAME)-*.tgz

# Show chart info
info:
	@echo "Chart Name: $(CHART_NAME)"
	@echo "Chart Version: $(CHART_VERSION)"
	@echo "App Version: $(APP_VERSION)"
	@echo "Registry: $(REGISTRY)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo ""
	@echo "To install from GHCR:"
	@echo "  helm install $(CHART_NAME) $(REGISTRY)/$(CHART_NAME) --version $(CHART_VERSION)"
	@echo ""
	@echo "To pull from GHCR:"
	@echo "  helm pull $(REGISTRY)/$(CHART_NAME) --version $(CHART_VERSION)"

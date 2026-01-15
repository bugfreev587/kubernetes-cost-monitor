.PHONY: infra api build-agent run-agent kind create-kind deploy-k8s

help:
	@echo "Available targets:"
	@echo "  infra-up          - Start infrastructure services using Docker Compose"
	@echo "  infra-down        - Stop infrastructure services using Docker Compose"
	@echo "  api               - Run the API server locally"
	@echo "  build-api         - Build the API server Docker image"
	@echo "  build-agent       - Build the agent Docker image"
	@echo "  run-agent         - Run the agent Docker container"
	@echo "  kind              - Create local Kubernetes clusters using Kind"
	@echo "  deploy-k8s        - Deploy the agent and sample app to the Kind cluster"
	@echo "  clear-timescale   - Clear all data from TimescaleDB (preserves schema)"
	@echo "  reset-timescale   - Complete reset: drop and recreate TimescaleDB tables"

infra-up:
	cd docker && docker-compose up -d

infra-down:
	cd docker && docker-compose down -v

clear-timescale:
	@echo "Clearing TimescaleDB data (preserving schema)..."
	docker exec -i k8s_cost_timescaledb psql -U ts_user -d timeseries < docker/clear-timescale-data.sql

reset-timescale:
	@echo "WARNING: This will DROP and recreate all TimescaleDB tables!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker exec -i k8s_cost_timescaledb psql -U ts_user -d timeseries < docker/clear-timescale-complete.sql; \
		docker exec -i k8s_cost_timescaledb psql -U ts_user -d timeseries < docker/timescale-init.sql; \
	else \
		echo "Cancelled."; \
	fi

api:
	go run ./cmd/server

build-api:
	docker build -t k8s-cost-api:local .

# build-agent:
# 	cd agent && docker build -t k8s-cost-agent:local .

# run-agent:
# 	docker run --rm -e AGENT_ENDPOINT="http://host.docker.internal:8080/v1/ingest" -e AGENT_API_KEY="yourkeyid:secret" k8s-cost-agent:local

kind:
	./k8s/create-kind-clusters.sh

deploy-k8s:
	kubectl --context kind-kind-cluster-1 apply -f k8s/manifests/agent-configmap.yaml
	kubectl --context kind-kind-cluster-1 apply -f k8s/manifests/sample-deployment.yaml
	kubectl --context kind-kind-cluster-1 apply -f k8s/manifests/agent-deployment.yaml
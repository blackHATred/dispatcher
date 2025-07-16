.PHONY: start-frontend
start-frontend:
	@echo "Starting frontend..."
	cd web && npm run dev

.PHONY: build-frontend
build-frontend:
	@echo "Building frontend..."
	cd web && npm run build

.PHONY: start-dispatcher-client
start-dispatcher-client: build-frontend
	@echo "Starting backend..."
	go run cmd/dispatcher-client/main.go
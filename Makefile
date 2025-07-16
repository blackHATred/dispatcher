.PHONY: start-frontend
start-frontend:
	@echo "Starting frontend..."
	cd web && npm run dev

.PHONY: build-frontend
build-frontend:
	@echo "Building frontend..."
	cd web && npm run build

.PHONY: lidar
lidar:
	@echo "Starting LiDAR simulation..."
	go run cmd/lidar/main.go

.PHONY: start-dispatcher-client
start-dispatcher-client:
	@echo "Starting backend..."
	go run cmd/dispatcher-client/main.go --filter-radius 0.01 --voxel-size 0.01

.PHONY: start-dispatcher-server
start-dispatcher-server: build-frontend
	@echo "Starting backend..."
	go run cmd/dispatcher-server/main.go
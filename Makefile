ENGINE := services/engine
LANDING := apps/landing
WORKSPACE := apps/workspace

.PHONY: setup dev test build lint migrate docker-up docker-down workspace

setup:
	cd $(LANDING) && npm install
	cd $(WORKSPACE) && npm install
	cd $(ENGINE) && go mod download

dev:
	cd $(LANDING) && npm run dev

workspace:
	cd $(WORKSPACE) && npm run dev

engine:
	cd $(ENGINE) && go run ./cmd/server

test:
	cd $(ENGINE) && go test ./...
	cd $(LANDING) && npm run build
	cd $(WORKSPACE) && npm test
	cd $(WORKSPACE) && npm run build

build:
	cd $(ENGINE) && mkdir -p bin && go build -o bin/server ./cmd/server
	cd $(LANDING) && npm run build
	cd $(WORKSPACE) && npm run build

lint:
	cd $(ENGINE) && go vet ./...
	cd $(WORKSPACE) && npm run lint

sync-openapi:
	cp docs/openapi.yaml services/engine/internal/http/openapi.yaml

migrate:
	cd $(ENGINE) && go run -run=Migrate ./cmd/server

docker-up:
	cd $(ENGINE) && docker compose up -d

docker-down:
	cd $(ENGINE) && docker compose down

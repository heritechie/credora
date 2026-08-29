ENGINE := services/engine
LANDING := apps/landing

.PHONY: setup dev test build lint

setup:
	cd $(LANDING) && npm install
	cd $(ENGINE) && go mod download

dev:
	cd $(LANDING) && npm run dev

engine:
	cd $(ENGINE) && go run ./cmd/server

test:
	cd $(ENGINE) && go test ./...
	cd $(LANDING) && npm run build

build:
	cd $(ENGINE) && mkdir -p bin && go build -o bin/server ./cmd/server
	cd $(LANDING) && npm run build

lint:
	cd $(ENGINE) && go vet ./...
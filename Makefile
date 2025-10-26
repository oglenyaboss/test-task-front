APP_NAME=wms

.PHONY: run build tidy docker-up docker-down

run:
	go run ./cmd/api

build:
	mkdir -p bin
	go build -o bin/$(APP_NAME) ./cmd/api

tidy:
	go mod tidy

docker-up:
	docker compose up --build

docker-down:
	docker compose down -v
